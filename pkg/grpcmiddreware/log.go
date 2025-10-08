package grpcmiddreware

import (
	"context"
	"im/pkg/xcontext"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

func LogUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return logUnaryInterceptor(ctx, logger, req, info, handler)
	}
}

func logUnaryInterceptor(ctx context.Context, logger *slog.Logger, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	startTime := time.Now()
	logger = logger.With("full_method", info.FullMethod, "start_time", startTime.Format(time.RFC3339Nano), "trace_id", xcontext.GetTraceID(ctx))
	resp, err := handler(ctx, req)
	if err != nil {
		logger.Error("gRPC call failed", "error", err, "duration", time.Since(startTime).String())
	}
	logger.Debug("log Interceptor debug info", "req", req, "resp", resp,"duration", time.Since(startTime).String())

	return resp, err
}
