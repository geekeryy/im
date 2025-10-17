package main

import (
	"fmt"
	"im/client/common"
	"im/client/page"
	"im/pkg/plato"
	"image/color"
	"io"
	"log"
	"net"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"google.golang.org/protobuf/proto"
)

func main() {
	a := app.New()
	// 应用自定义主题以隐藏滚动条
	a.Settings().SetTheme(&customTheme{Theme: theme.DefaultTheme()})

	ctx := &common.Context{
		App:       a,
		LoginPage: nil,
		HomePage:  nil,
		Account:   "",
		Password:  "",
	}
	ctx.LoginPage = page.LoginPage(ctx)
	ctx.HomePage = page.HomePage(ctx)

	ctx.LoginPage.Show()
	a.Run()

}

// customTheme 自定义主题，用于隐藏滚动条
type customTheme struct {
	fyne.Theme
}

// Color 重写颜色方法，将滚动条设置为透明
func (ct *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// 隐藏滚动条：将滚动条背景和前景色设为透明
	if name == theme.ColorNameScrollBar {
		return color.Transparent
	}
	// 其他颜色使用默认主题
	return theme.DefaultTheme().Color(name, variant)
}

func commandLine() {
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
