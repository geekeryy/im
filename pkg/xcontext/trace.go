package xcontext

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
)

type traceid struct{}

const traceIDKey = "trace-id"

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, &traceid{}, traceID)
}

func GetTraceID(ctx context.Context) string {
	return ctx.Value(&traceid{}).(string)
}

// GetOrGenerateTraceID 从上下文中获取或生成新的 trace ID
func GetOrGenerateTraceID(ctx context.Context) string {
	// 尝试从 gRPC metadata 中获取 trace ID
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if traceIDs := md.Get(traceIDKey); len(traceIDs) > 0 {
			return traceIDs[0]
		}
	}

	// 如果没有找到，生成新的 trace ID
	return generateTraceID()
}

// GenerateTraceID 生成新的 trace ID
func generateTraceID() string {
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}
