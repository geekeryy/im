package test

import (
	"fmt"
	"im/pkg/plato"
	"io"
	"log"
	"net"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

func TestImgateway(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:8086")
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	conn.Write(plato.Marshal(1, plato.MsgTypeOpenSession, nil, nil))

	sessionidChan := make(chan string)
	go read(conn, sessionidChan)

	go write(conn, sessionidChan)

	select {}

}

func write(conn net.Conn, sessionidChan chan string) {
	sessionid := <-sessionidChan
	for {
		select {
		case <-time.After(2 * time.Second):
			msg, err := proto.Marshal(&plato.MessageUpLink{
				SessionId:  sessionid,
				FromUserId: "456",
				Payload:    "Hello, World!",
				Timestamp:  time.Now().Unix(),
			})
			if err != nil {
				log.Fatalf("failed to marshal: %v", err)
			}

			data := plato.Marshal(1, 1, nil, msg)
			if n, err := conn.Write(data); err != nil {
				log.Printf("failed to write: %v", err)
			} else {
				log.Printf("write: %d", n)
			}
		case <-time.After(10 * time.Second):
			return
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
			sessionid := string(content[fixHeader.GetVarHeaderLen():])
			sessionidChan <- sessionid
			fmt.Println("sessionid:", sessionid)
		}
	}
}
