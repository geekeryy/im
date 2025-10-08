package service

import (
	context "context"
	"im/pkg/config"
	"log/slog"
)

type IMGatewayService struct {
	UnimplementedIMGatewayServer
	ctx    context.Context
	logger *slog.Logger
	conf   *config.IMGatewayConfig
}

func NewIMGatewayService(ctx context.Context, logger *slog.Logger, conf *config.IMGatewayConfig) *IMGatewayService {
	return &IMGatewayService{ctx: ctx, logger: logger, conf: conf}
}

func (s *IMGatewayService) DelConn(ctx context.Context, req *DelConnRequest) (*DelConnResponse, error) {
	return nil, nil
}
