package imgateway

import (
	"context"
	"im/pkg/config"
	"im/pkg/grpcmiddreware"
	"im/pkg/plato"
	"im/server/imgateway/rpc/service"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
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

	go serve(conf.Addr, logger)

	listener, err := net.Listen("tcp", conf.RpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("im gateway server listening", "address", conf.Addr)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type Session struct {
	userids   []string
}

type Connection struct {
	user_id string
	conn    net.Conn
}

var sessions = make(map[string]*Session)
var connections = make(map[string]*Connection)
var user_connections = make(map[string]string)

func serve(addr string, logger *slog.Logger) {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("failed to accept: %v", err)
		}
		conn_id := uuid.New().String()
		connections[conn_id] = &Connection{
			conn:    conn,
		}
		logger = logger.With("conn_id", conn_id)
		go accept(conn_id, conn, logger)
	}

}

func accept(conn_id string, conn net.Conn, logger *slog.Logger) {
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
			// 创建连接
			msg := plato.MessageCreateConn{}
			proto.Unmarshal(content[fixHeader.GetVarHeaderLen():], &msg)
			logger.Info("create conn", "user_id", msg.GetUserId())
			connections[conn_id].user_id = msg.GetUserId()
			user_connections[msg.GetUserId()] = conn_id
		case plato.MsgTypeMessageUpLink:
			// 发送消息
			if connections[conn_id] == nil {
				logger.Error("connection not found")
				continue
			}
			msg := plato.MessageUpLink{}
			proto.Unmarshal(content[fixHeader.GetVarHeaderLen():], &msg)
			logger.Info("receive msg", "session_id", msg.GetSessionId(), "from_user_id", msg.GetFromUserId(), "payload", msg.GetPayload(), "timestamp", msg.GetTimestamp())

			session, ok := sessions[msg.GetSessionId()]
			if !ok {
				logger.Error("session not found")
				continue
			}
			for _, userid := range session.userids {
				connid, ok := user_connections[userid]
				if !ok {
					logger.Error("connection not found")
					continue
				}
				if connid == conn_id {
					continue
				}
				connection, ok := connections[connid]
				if !ok {
					logger.Error("connection not found")
					continue
				}
				msg := &plato.MessageDownLink{
					SessionId:  msg.GetSessionId(),
					SeqId:      int64(time.Now().UnixNano()),
					FromUserId: msg.GetFromUserId(),
					Payload:    msg.GetPayload(),
					Timestamp:  msg.GetTimestamp(),
				}
				downLinkmsg, _ := proto.Marshal(msg)
				connection.conn.Write(plato.Marshal(1, plato.MsgTypeMessageDownLink, nil, downLinkmsg))
				logger.Info("send msg", "to_conn_id", connid, "session_id", msg.GetSessionId(), "from_user_id", msg.GetFromUserId(), "payload", msg.GetPayload(), "timestamp", msg.GetTimestamp())
			}

		case plato.MsgTypeOpenSession:
			// 打开会话
			msg := plato.MessageOpenSessionReq{}
			proto.Unmarshal(content[fixHeader.GetVarHeaderLen():], &msg)
			sessionid := uuid.New().String()
			logger.Info("open session", "with_user_ids", msg.GetWithUserIds(), "session_id", sessionid)			
			sessions[sessionid] = &Session{
				userids: append(msg.GetWithUserIds(),connections[conn_id].user_id),
			}
			openSessionmsgResp, _ := proto.Marshal(&plato.MessageOpenSessionResp{
				SessionId: sessionid,
			})
			connections[conn_id].conn.Write(plato.Marshal(1, plato.MsgTypeOpenSession, nil, openSessionmsgResp))
		case plato.MsgTypeJoinSession:
			// 加入会话
			msg := plato.MessageJoinSession{}
			proto.Unmarshal(content[fixHeader.GetVarHeaderLen():], &msg)
			sessionid := msg.GetSessionId()
			logger.Info("join session", "session_id", sessionid)
			session, ok := sessions[string(sessionid)]
			if !ok {
				logger.Error("session not found")
				continue
			}
			session.userids = append(session.userids, connections[conn_id].user_id)
		case plato.MsgTypeLeaveSession:
			// 离开会话
			msg := plato.MessageLeaveSession{}
			proto.Unmarshal(content[fixHeader.GetVarHeaderLen():], &msg)
			sessionid := msg.GetSessionId()
			logger.Info("leave session", "session_id", sessionid)
			session, ok := sessions[string(sessionid)]
			if !ok {
				logger.Error("session not found")
				continue
			}
			
			session.userids = remove(session.userids, connections[conn_id].user_id)
		}

	}
	logger.Info("close")
}

func remove(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
