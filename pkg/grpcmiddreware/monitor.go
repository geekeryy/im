package grpcmiddreware

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// MonitorUnaryInterceptor 创建一个用于链路追踪的 gRPC 一元拦截器
func MonitorUnaryInterceptor(ctx context.Context) grpc.UnaryServerInterceptor {
	var allCounter atomic.Int64
	var ingCounter atomic.Int64
	uuid := uuid.New().String()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Println(uuid,"allCounter", allCounter.Load(), "ingCounter", ingCounter.Load())
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		allCounter.Add(1)
		ingCounter.Add(1)
		defer ingCounter.Add(-1)
		resp, err := handler(ctx, req)
		return resp, err
	}
}
