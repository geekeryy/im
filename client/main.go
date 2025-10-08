package main

import (
	"fmt"
	"im/pkg/plato"
	"io"
	"log"
	"net"
	"time"

	"google.golang.org/protobuf/proto"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8086")
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	var userId string
	fmt.Println("请输入用户ID: ")
	fmt.Scanln(&userId)

	msg, err := proto.Marshal(&plato.MessageCreateConn{
		UserId: userId,
	})
	if err != nil {
		log.Fatalf("failed to marshal: %v", err)
	}
	conn.Write(plato.Marshal(1, plato.MsgTypeCreateConn, nil, msg))

	var withUserId string
	fmt.Println("你要和谁聊天: ")
	fmt.Scanln(&withUserId)

	if userId == withUserId {
		fmt.Println("不能和自己聊天")
		return
	}
	sessionidChan := make(chan string)
	if withUserId != "" {
		msg, err := proto.Marshal(&plato.MessageOpenSessionReq{
			WithUserIds: []string{withUserId},
		})
		if err != nil {
			log.Fatalf("failed to marshal: %v", err)
		}
		conn.Write(plato.Marshal(1, plato.MsgTypeOpenSession, nil, msg))
		go read(conn, sessionidChan)
		go write(conn, userId, sessionidChan)
	} else {
		fmt.Println("等待其他人给你发送消息")
		go read(conn, sessionidChan)
	}

	select {}

}

func write(conn net.Conn, userId string, sessionidChan chan string) {
	sessionid := <-sessionidChan
	for {
		var input string
		fmt.Print("Enter message: ")
		fmt.Scanln(&input)

		payload := input
		msg, err := proto.Marshal(&plato.MessageUpLink{
			SessionId:  sessionid,
			FromUserId: userId,
			Payload:    payload,
			Timestamp:  time.Now().Unix(),
		})
		if err != nil {
			log.Fatalf("failed to marshal: %v", err)
		}

		data := plato.Marshal(1, plato.MsgTypeMessageUpLink, nil, msg)
		if _, err := conn.Write(data); err != nil {
			log.Printf("failed to write: %v", err)
		}
	}

}

func read(conn net.Conn, sessionidChan chan string) {
	for {
		buf := make([]byte, 10)
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("\nfailed to read: %v\n", err)
			}
			break
		}
		fixHeader := &plato.FixHeaderProtocol{}
		if err := fixHeader.Unmarshal(buf[:n]); err != nil {
			fmt.Printf("\nfailed to unmarshal: %v\n", err)
			break
		}
		content := make([]byte, fixHeader.GetVarHeaderLen()+fixHeader.GetBodyLen())
		if _, err := conn.Read(content); err != nil {
			if err != io.EOF {
				fmt.Printf("\nfailed to read content: %v\n", err)
				break
			}
		}
		switch fixHeader.GetMsgType() {
		case plato.MsgTypeMessageDownLink:
			msg := plato.MessageDownLink{}
			proto.Unmarshal(content[fixHeader.GetVarHeaderLen():], &msg)
			fmt.Println("msg:", msg.GetSessionId(), msg.GetFromUserId(), msg.GetPayload(), msg.GetTimestamp())
		case plato.MsgTypeOpenSession:
			msg := plato.MessageOpenSessionResp{}
			proto.Unmarshal(content[fixHeader.GetVarHeaderLen():], &msg)
			sessionidChan <- msg.GetSessionId()
			fmt.Println("sessionid:", msg.GetSessionId())
		}
	}
}
