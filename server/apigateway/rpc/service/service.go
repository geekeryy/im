package service

import (
	context "context"
	"fmt"
	"im/model"
	"im/pkg/config"
	"log"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type APIGatewayService struct {
	UnimplementedAPIGatewayServer
	ctx                 context.Context
	logger              *slog.Logger
	conf                *config.APIGatewayConfig
	MysqlClient         sqlx.SqlConn
	RedisClient         *redis.Client
	SessionsModel       model.SessionsModel
	MessagesModel       model.MessagesModel
	UserBaseModel       model.UserBaseModel
	SessionMembersModel model.SessionMembersModel
}

func NewAPIGatewayService(ctx context.Context, logger *slog.Logger, conf *config.APIGatewayConfig) *APIGatewayService {
	mysqlClient, err := sqlx.NewConn(sqlx.SqlConf{
		DataSource: fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", conf.MysqlConfig.Username, conf.MysqlConfig.Password, conf.MysqlConfig.Addr, conf.MysqlConfig.DB),
		DriverName: "mysql",
		Replicas:   nil,
		Policy:     "",
	})
	if err != nil {
		log.Fatalf("failed to open mysql: %v", err)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     conf.RedisConfig.Addr,
		Password: conf.RedisConfig.Password,
		DB:       conf.RedisConfig.DB,
	})
	return &APIGatewayService{
		ctx:                 ctx,
		logger:              logger,
		conf:                conf,
		MysqlClient:         mysqlClient,
		RedisClient:         redisClient,
		SessionsModel:       model.NewSessionsModel(mysqlClient),
		UserBaseModel:       model.NewUserBaseModel(mysqlClient),
		MessagesModel:       model.NewMessagesModel(mysqlClient),
		SessionMembersModel: model.NewSessionMembersModel(mysqlClient),
	}
}

func (s *APIGatewayService) SessionList(ctx context.Context, req *SessionListRequest) (*SessionListResponse, error) {
	sessionListResponse := &SessionListResponse{
		Sessions: make([]*Session, 0),
	}
	sessionList, err := s.SessionMembersModel.FindSessionsByUserUuid(ctx, req.UserUuid)
	if err != nil {
		return nil, err
	}
	for _, sessionUuid := range sessionList {
		session, err := s.SessionsModel.FindByUuid(ctx, sessionUuid)
		if err != nil {
			return nil, err
		}
		latestMessage, err := s.MessagesModel.FindLatestMessageBySessionUuid(ctx, sessionUuid)
		if err != nil {
			return nil, err
		}

		sessionListResponse.Sessions = append(sessionListResponse.Sessions, &Session{
			Uuid:        session.Uuid,
			Name:        session.Name,
			Avatar:      session.Avatar,
			LastMessage: latestMessage.Content,
			LastTime:    latestMessage.CreatedAt.Format("1月2日 15:04"),
			UnreadCount: 0,
		})
	}
	return sessionListResponse, nil
}

func (s *APIGatewayService) HistoryMessage(ctx context.Context, req *HistoryMessageRequest) (*HistoryMessageResponse, error) {
	messageListResponse := &HistoryMessageResponse{
		Messages: make([]*Message, 0),
	}
	messageList, err := s.MessagesModel.FindMessagesBySeqidGreaterThan(ctx, req.SessionUuid, req.StartSeqid)
	if err != nil {
		return nil, err
	}
	if len(messageList) == 0 {
		return messageListResponse, nil
	}

	useruuidMap := make(map[string]struct{}, 0)
	for _, message := range messageList {
		useruuidMap[message.SenderUuid] = struct{}{}
		messageListResponse.Messages = append(messageListResponse.Messages, &Message{
			MessageUuid: message.Uuid,
			SessionUuid: message.SessionUuid,
			SeqId:       message.SeqId,
			MessageType: message.MessageType,
			Content:     message.Content,
			SenderUuid:  message.SenderUuid,
			SendTime:    message.CreatedAt.Format("1月2日 15:04"),
		})
	}

	useruuidList := make([]string, 0, len(useruuidMap))
	for useruuid := range useruuidMap {
		useruuidList = append(useruuidList, useruuid)
	}
	userBaseList, err := s.UserBaseModel.FindByUuids(ctx, useruuidList)
	if err != nil {
		return nil, err
	}
	userBaseMap := make(map[string]*model.UserBase, len(userBaseList))
	for _, userBase := range userBaseList {
		userBaseMap[userBase.Uuid] = userBase
	}
	for _, message := range messageListResponse.Messages {
		userBase, ok := userBaseMap[message.SenderUuid]
		if !ok {
			continue
		}
		message.SenderName = userBase.Name
		message.SenderAvatar = userBase.Avatar
	}

	return messageListResponse, nil
}
