package apigateway

import (
	"context"
	"im/pkg/config"
	"im/pkg/grpcmiddreware"
	"log"
	"log/slog"
	"net"
	"os"

	"im/server/apigateway/rpc/service"

	"google.golang.org/grpc"
)

//go:generate protoc --go_out=rpc/service --go-grpc_out=rpc/service rpc/service/apigateway.proto

func Run() {
	ctx := context.Background()
	conf := config.NewConf().GetAPIGatewayConfig()

	level := slog.LevelInfo
	if conf.Mode == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpcmiddreware.TraceUnaryInterceptor(), grpcmiddreware.LogUnaryInterceptor(logger)),
	)

	service.RegisterAPIGatewayServer(server, service.NewAPIGatewayService(ctx, logger, conf))
	listener, err := net.Listen("tcp", conf.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("api gateway server listening", "address", conf.Addr)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
