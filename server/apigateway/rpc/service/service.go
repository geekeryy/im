package service

import (
	context "context"
	"errors"
	"fmt"
	"im/model"
	"im/pkg/config"
	"im/pkg/jwt"
	"im/pkg/password"
	"log"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"
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
	UserIdentityModel   model.UserIdentityModel
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
		UserIdentityModel:   model.NewUserIdentityModel(mysqlClient),
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

func (s *APIGatewayService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	userIdentity, err := s.UserIdentityModel.FindByIdentifierAndIdentityType(ctx, req.Identifier, req.IdentityType)
	if err != nil {
		return nil, err
	}
	if userIdentity == nil {
		return nil, nil
	}
	switch req.IdentityType {
	case model.IdentityTypePassword:
		if !password.Check(req.Credential, userIdentity.Credential) {
			return nil, errors.New("密码错误")
		}
	default:
		return nil, errors.New("不支持的身份类型")
	}
	token, _, err := jwt.GenerateToken(userIdentity.UserUuid, 3600, nil)
	if err != nil {
		return nil, err
	}
	refreshToken, _, err := jwt.GenerateToken(userIdentity.UserUuid, 86400, map[string]interface{}{"type": "refresh"})
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
	}, nil
}

func (s *APIGatewayService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	userIdentity, err := s.UserIdentityModel.FindByIdentifierAndIdentityType(ctx, req.Identifier, req.IdentityType)
	if err != nil {
		return nil, err
	}
	if userIdentity != nil {
		return nil, errors.New("用户已存在")
	}

	var credential string
	switch req.IdentityType {
	case model.IdentityTypePassword:
		credential, err = password.HashEncrypt(req.Credential)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("不支持的身份类型")
	}

	userUuid := uuid.New().String()
	userName := NewRandomUserName()
	userIdentityNew := &model.UserIdentity{
		UserUuid:     userUuid,
		Identifier:   req.Identifier,
		IdentityType: req.IdentityType,
		Credential:   credential,
	}
	userBase := &model.UserBase{
		Uuid:   userUuid,
		Name:   userName,
		Avatar: "https://avatar.com/user.png",
		Status: model.UserStatusActive,
	}
	if err:=s.MysqlClient.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err = s.UserBaseModel.RegisterUserBase(ctx, session, userBase); err != nil {
			return err
		}
		if err = s.UserIdentityModel.RegisterUserIdentity(ctx, session, userIdentityNew); err != nil {
			return err
		}
		return nil
	});err!=nil{
		return nil, err
	}

	token, _, err := jwt.GenerateToken(userIdentityNew.UserUuid, 3600, nil)
	if err != nil {
		return nil, err
	}
	refreshToken, _, err := jwt.GenerateToken(userIdentityNew.UserUuid, 86400, map[string]interface{}{"type": "refresh"})
	if err != nil {
		return nil, err
	}
	return &RegisterResponse{
		Token:        token,
		RefreshToken: refreshToken,
	}, nil
}

// 生成随机昵称
func NewRandomUserName() string {
	adjectives := []string{"快乐的", "聪明的", "勇敢的", "温柔的", "活泼的", "可爱的", "优雅的", "神秘的", "阳光的", "梦幻的"}
	nouns := []string{"小猫", "小狗", "小鸟", "小熊", "小兔", "小鱼", "小鹿", "小狐", "小象", "小龙"}

	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	adjective := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	number := rand.Intn(999999)
	return fmt.Sprintf("%s%s%06d", adjective, noun, number)
}
