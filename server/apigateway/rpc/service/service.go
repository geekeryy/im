package service

import (
	context "context"
	"im/pkg/config"
	"log/slog"
)

type APIGatewayService struct {
	UnimplementedAPIGatewayServer
	ctx    context.Context
	logger *slog.Logger
	conf   *config.APIGatewayConfig
}

func NewAPIGatewayService(ctx context.Context, logger *slog.Logger, conf *config.APIGatewayConfig) *APIGatewayService {
	return &APIGatewayService{ctx: ctx, logger: logger, conf: conf}
}


func (s *APIGatewayService) SessionList(ctx context.Context, req *SessionListRequest) (*SessionListResponse, error) {
	return nil, nil
}

func (s *APIGatewayService) HistoryMessage(ctx context.Context, req *HistoryMessageRequest) (*HistoryMessageResponse, error) {
	return nil, nil
}