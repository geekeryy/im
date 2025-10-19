package common

import (
	"fmt"
	"im/pkg/plato"
	"io"
	"log"

	"google.golang.org/protobuf/proto"
)

func CreateConn(ctx *Context, token string) {
	msg, err := proto.Marshal(&plato.MessageCreateConn{
		Token: token,
	})
	if err != nil {
		log.Fatalf("failed to marshal: %v", err)
	}
	ctx.IMGatewayLongConn.Write(plato.Marshal(1, plato.MsgTypeCreateConn, nil, msg))
}

func Read(ctx *Context) {
	for {
		buf := make([]byte, 10)
		n, err := ctx.IMGatewayLongConn.Read(buf)
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
		if _, err := ctx.IMGatewayLongConn.Read(content); err != nil {
			if err != io.EOF {
				fmt.Printf("\nfailed to read content: %v\n", err)
				break
			}
		}
		switch fixHeader.GetMsgType() {
		case plato.MsgTypeMessageDownLink:
			msg := plato.MessageDownLink{}
			proto.Unmarshal(content[fixHeader.GetVarHeaderLen():], &msg)
			fmt.Println("msg:", msg.GetSessionUuid(), msg.GetSenderUserUuid(), msg.GetPayload(), msg.GetSeqId())
			var avatarURI string
			if users, ok := ctx.SessionUserTable[msg.GetSessionUuid()]; ok {
				if user, ok := users[msg.GetSenderUserUuid()]; ok {
					avatarURI = user.Avatar
				}
			}
			ctx.MessageReadChan <- ChatMessage{
				SessionUuid: msg.GetSessionUuid(),
				Content:     msg.GetPayload(),
				IsSent:      false,
				AvatarURI:   avatarURI,
			}
		}
	}
}

func Write(ctx *Context) {
	messageWriteChan := ctx.MessageWriteChan
	for message := range messageWriteChan {
		msg, err := proto.Marshal(&plato.MessageUpLink{
			SessionUuid: message.SessionUuid,
			Payload:     message.Content,
		})
		if err != nil {
			log.Fatalf("failed to marshal: %v", err)
		}
		data := plato.Marshal(1, plato.MsgTypeMessageUpLink, nil, msg)
		if _, err := ctx.IMGatewayLongConn.Write(data); err != nil {
			log.Printf("failed to write: %v", err)
		}
	}
}
