package discovery

import (
	"context"
	"im/pkg/config"
	"im/pkg/grpcmiddreware"
	"im/server/discovery/rpc/service"
	"log"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"
)

//go:generate protoc --go_out=rpc/service --go-grpc_out=rpc/service rpc/service/discovery.proto

func Run() {
	ctx := context.Background()
	conf := config.NewConf().GetDiscoveryConfig()

	level := slog.LevelInfo
	if conf.Mode == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpcmiddreware.MonitorUnaryInterceptor(ctx), grpcmiddreware.TraceUnaryInterceptor(), grpcmiddreware.LogUnaryInterceptor(logger)),
	)

	service.RegisterDiscoveryServer(server, service.NewDiscoveryService(ctx, logger, conf))

	listener, err := net.Listen("tcp", conf.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	logger.Info("discovery server listening", "address", conf.Addr)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	// TODO	处理信号

}
