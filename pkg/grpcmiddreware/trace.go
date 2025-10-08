package grpcmiddreware

import (
	"context"
	"im/pkg/xcontext"

	"google.golang.org/grpc"
)

// TraceUnaryInterceptor 创建一个用于链路追踪的 gRPC 一元拦截器
func TraceUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(xcontext.WithTraceID(ctx, xcontext.GetOrGenerateTraceID(ctx)), req)
	}
}
