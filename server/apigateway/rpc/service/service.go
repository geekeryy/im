package service

import (
	context "context"
	"errors"
	"fmt"
	"im/model"
	"im/pkg/config"
	"im/pkg/jwt"
	"im/pkg/password"
	"im/pkg/xcontext"
	"im/pkg/xstrings"
	"log"
	"log/slog"

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
	UserInfoModel       model.UserInfoModel
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
		UserInfoModel:       model.NewUserInfoModel(mysqlClient),
	}
}

func (s *APIGatewayService) SessionList(ctx context.Context, req *SessionListRequest) (*SessionListResponse, error) {
	sessionListResponse := &SessionListResponse{
		Sessions: make([]*Session, 0),
	}
	userUUID := xcontext.GetUserUUID(ctx)
	sessionList, err := s.SessionMembersModel.FindSessionsByUserUuid(ctx, userUUID)
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
		sessionItem := &Session{
			Uuid:        session.Uuid,
			Name:        session.Name,
			Avatar:      session.Avatar,
			UnreadCount: 0,
		}

		switch session.SessionType {
		case model.SessionTypeSingle:
			sessionMember, err := s.SessionMembersModel.FindAllMembersBySessionUuid(ctx, sessionUuid)
			if err != nil {
				return nil, err
			}
			otherMember := ""
			for _, member := range sessionMember {
				if member == userUUID {
					continue
				}
				otherMember = member
			}
			otherMemberBase, err := s.UserBaseModel.FindByUuid(ctx, otherMember)
			if err != nil {
				return nil, err
			}
			if otherMemberBase == nil {
				return nil, fmt.Errorf("other member not found by uuid %s", otherMember)
			}
			sessionItem.Name = otherMemberBase.Name
			sessionItem.Avatar = otherMemberBase.Avatar
		}

		if latestMessage != nil {
			sessionItem.LastMessage = latestMessage.Content
			sessionItem.LastTime = latestMessage.CreatedAt.Format("1月2日 15:04")
		}
		sessionListResponse.Sessions = append(sessionListResponse.Sessions, sessionItem)
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
	token, _, err := jwt.GenerateToken(userIdentity.UserUuid, 3600*24*365, nil)
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
	userName := xstrings.NewRandomUserName()
	avatar := xstrings.NewRandomAvatar()
	userIdentityNew := &model.UserIdentity{
		UserUuid:     userUuid,
		Identifier:   req.Identifier,
		IdentityType: req.IdentityType,
		Credential:   credential,
	}
	userBase := &model.UserBase{
		Uuid:   userUuid,
		Name:   userName,
		Avatar: avatar,
		Status: model.UserStatusActive,
	}
	if err := s.MysqlClient.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err = s.UserBaseModel.RegisterUserBase(ctx, session, userBase); err != nil {
			return err
		}
		if err = s.UserIdentityModel.RegisterUserIdentity(ctx, session, userIdentityNew); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if err := s.initSession(ctx, userUuid); err != nil {
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

// 为当前用户与所有好友建立单聊会话
func (s *APIGatewayService) initSession(ctx context.Context, userUUID string) error {
	userBases, err := s.UserBaseModel.FindAll(ctx)
	if err != nil {
		return err
	}
	if err := s.MysqlClient.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		for _, userBase := range userBases {
			if userBase.Uuid == userUUID {
				continue
			}
			sessionUUID := uuid.New().String()

			err := s.SessionsModel.CreateSession(ctx, session, &model.Sessions{
				Uuid:        sessionUUID,
				Name:        "",
				Avatar:      "",
				SessionType: model.SessionTypeSingle,
				Status:      model.SessionStatusActive,
			})
			if err != nil {
				return err
			}
			err = s.SessionMembersModel.JoinSession(ctx, session, sessionUUID, userUUID)
			if err != nil {
				return err
			}
			err = s.SessionMembersModel.JoinSession(ctx, session, sessionUUID, userBase.Uuid)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (s *APIGatewayService) GetSessionUserList(ctx context.Context, req *GetSessionUserListRequest) (*GetSessionUserListResponse, error) {
	sessionUserListResponse := &GetSessionUserListResponse{
		Users: make([]*SessionUserListItem, 0),
	}
	sessionUserList, err := s.SessionMembersModel.FindAllMembersBySessionUuid(ctx, req.SessionUuid)
	if err != nil {
		return nil, err
	}
	for _, userUuid := range sessionUserList {
		userBase, err := s.UserBaseModel.FindByUuid(ctx, userUuid)
		if err != nil {
			return nil, err
		}
		if userBase == nil {
			continue
		}
		sessionUserListResponse.Users = append(sessionUserListResponse.Users, &SessionUserListItem{
			UserUuid:   userUuid,
			UserName:   userBase.Name,
			UserAvatar: userBase.Avatar,
		})
	}
	return sessionUserListResponse, nil
}

func (s *APIGatewayService) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	message := &model.Messages{
		Uuid:        uuid.New().String(),
		SessionUuid: req.SessionUuid,
		SenderUuid:  req.SenderUuid,
		MessageType: req.MessageType,
		Status:      model.MessageStatusSent,
		SeqId:       req.SeqId,
		Content:     req.Payload,
	}
	if _, err := s.MessagesModel.Insert(ctx, message); err != nil {
		return nil, err
	}
	return &SendMessageResponse{
		MessageUuid: message.Uuid,
	}, nil
}

func (s *APIGatewayService) GetUserInfo(ctx context.Context, req *GetUserInfoRequest) (*GetUserInfoResponse, error) {
	userbase, err := s.UserBaseModel.FindByUuid(ctx, xcontext.GetUserUUID(ctx))
	if err != nil {
		return nil, err
	}
	if userbase == nil {
		return nil, errors.New("用户不存在")
	}
	resp := &GetUserInfoResponse{
		Uuid:   userbase.Uuid,
		Name:   userbase.Name,
		Avatar: userbase.Avatar,
	}
	userinfo, err := s.UserInfoModel.FindByUuid(ctx, userbase.Uuid)
	if err != nil {
		return nil, err
	}
	if userinfo != nil {
		resp.Email = userinfo.Email
		resp.Mobile = userinfo.Mobile
	}
	return resp, nil
}
