package imgateway

import (
	"context"
	"im/model"
	"im/pkg/config"
	"im/pkg/grpcmiddreware"
	"im/pkg/jwt"
	"im/pkg/plato"
	"im/server/imgateway/rpc/service"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"time"

	apigatewayService "im/server/apigateway/rpc/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

//go:generate protoc --go_out=rpc/service --go-grpc_out=rpc/service rpc/service/imgateway.proto

func Run() {
	ctx := context.Background()
	conf := config.NewConf().GetIMGatewayConfig()

	level := slog.LevelDebug
	if conf.Mode == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpcmiddreware.TraceUnaryInterceptor(), grpcmiddreware.LogUnaryInterceptor(logger)),
	)

	service.RegisterIMGatewayServer(server, service.NewIMGatewayService(ctx, logger, conf))

	go serve(conf, logger)

	listener, err := net.Listen("tcp", conf.RpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("im gateway server listening", "address", conf.Addr)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func serve(conf *config.IMGatewayConfig, logger *slog.Logger) {
	listener, err := net.Listen("tcp", conf.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	manager := NewConnManager()

	apiGatewayConn, err := grpc.NewClient(conf.APIGatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to create client", "error", err)
		return
	}
	apiGatewayClient := apigatewayService.NewAPIGatewayClient(apiGatewayConn)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("failed to accept: %v", err)
		}
		go accept(manager, apiGatewayClient, conn, logger)
	}

}

func accept(manager *ConnManager, apiGatewayClient apigatewayService.APIGatewayClient, conn net.Conn, logger *slog.Logger) {
	conn_uuid := ""
	user_uuid := ""
	for {
		buf := make([]byte, 10)
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				logger.Error("failed to read", "error", err)
			}
			break
		}
		fixHeader := &plato.FixHeaderProtocol{}
		if err := fixHeader.Unmarshal(buf[:n]); err != nil {
			logger.Error("failed to unmarshal", "error", err)
			break
		}
		content := make([]byte, fixHeader.GetVarHeaderLen()+fixHeader.GetBodyLen())
		if _, err := conn.Read(content); err != nil {
			if err != io.EOF {
				logger.Error("failed to read content", "error", err)
				break
			}
		}
		switch fixHeader.GetMsgType() {
		case plato.MsgTypeCreateConn:
			msg := plato.MessageCreateConn{}
			proto.Unmarshal(content[fixHeader.GetVarHeaderLen():], &msg)
			logger.Info("receive create conn", "token", msg.GetToken())
			claims, err := jwt.ValidateToken(msg.GetToken())
			if err != nil {
				logger.Error("failed to verify token", "error", err)
				continue
			}
			user_uuid, err = claims.GetSubject()
			if err != nil || len(user_uuid) == 0 {
				logger.Error("validate claims error", "error", err)
				continue
			}
			conn_uuid = manager.AddConnection(user_uuid, conn)
			logger.Info("create conn success", "conn_uuid", conn_uuid, "user_uuid", user_uuid)
		case plato.MsgTypeMessageUpLink:
			// 发送消息
			if len(conn_uuid) == 0 || manager.GetConnection(conn_uuid) == nil {
				logger.Error("connection not found", "conn_uuid", conn_uuid)
				continue
			}
			msg := plato.MessageUpLink{}
			proto.Unmarshal(content[fixHeader.GetVarHeaderLen():], &msg)
			logger.Info("receive msg", "session_uuid", msg.GetSessionUuid(), "payload", msg.GetPayload())
			session := manager.GetSession(msg.GetSessionUuid())
			if session == nil {
				sessionUserList, err := apiGatewayClient.GetSessionUserList(context.Background(), &apigatewayService.GetSessionUserListRequest{
					SessionUuid: msg.GetSessionUuid(),
				})
				if err != nil {
					logger.Error("failed to get session user list", "error", err)
					continue
				}
				userUuids := make([]string, 0)
				for _, user := range sessionUserList.Users {
					userUuids = append(userUuids, user.UserUuid)
				}
				session = manager.AddSession(msg.GetSessionUuid(), userUuids)
				if session == nil {
					logger.Error("failed to add session", "session_uuid", msg.GetSessionUuid())
					continue
				}
			}
			_, err = apiGatewayClient.SendMessage(context.Background(), &apigatewayService.SendMessageRequest{
				SessionUuid: msg.GetSessionUuid(),
				Payload:     msg.GetPayload(),
				SenderUuid:  user_uuid,
				MessageType: int64(model.MessageTypeText),
				SeqId:       int64(time.Now().UnixNano()),
				Timestamp:   int64(time.Now().Unix()),
			})
			if err != nil {
				logger.Error("failed to send message", "error", err)
				continue
			}
			for _, user := range session.user_uuids {
				if user == user_uuid {
					continue
				}
				connid := manager.GetUserConnUUID(user)
				if connid == "" {
					logger.Error("connection not found", "user_uuid", user)
					continue
				}
				if connid == conn_uuid {
					logger.Error("self send msg", "session_uuid", msg.GetSessionUuid(), "payload", msg.GetPayload(), "user_uuid", user, "conn_uuid", conn_uuid)
					continue
				}
				connection := manager.GetConnection(connid)
				if connection == nil {
					logger.Error("connection not found")
					continue
				}
				msg := &plato.MessageDownLink{
					SessionUuid:    msg.GetSessionUuid(),
					SenderUserUuid: user_uuid,
					SeqId:          int64(time.Now().UnixNano()),
					Payload:        msg.GetPayload(),
				}
				downLinkmsg, _ := proto.Marshal(msg)
				connection.conn.Write(plato.Marshal(1, plato.MsgTypeMessageDownLink, nil, downLinkmsg))
				logger.Info("send msg", "from_user_uuid", user_uuid, "to_conn_id", connid, "session_uuid", msg.GetSessionUuid(), "payload", msg.GetPayload())

			}

		}

	}
	logger.Info("close")
}
