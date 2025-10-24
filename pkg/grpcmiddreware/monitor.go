package grpcmiddreware

import (
	"context"
	"log/slog"
	"os"
	"runtime/trace"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// MonitorUnaryInterceptor 创建一个用于监控一元拦截器
func MonitorUnaryInterceptor(ctx context.Context, fr *trace.FlightRecorder, logger *slog.Logger) grpc.UnaryServerInterceptor {
	var allCounter atomic.Int64
	var ingCounter atomic.Int64
	uuid := uuid.New().String()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				logger.Info("monitor info", "uuid", uuid, "allCounter", allCounter.Load(), "ingCounter", ingCounter.Load())
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	frOutPut, err := os.OpenFile("fr.trace", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Error("failed to open file", "error", err)
	}

	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		allCounter.Add(1)
		ingCounter.Add(1)
		defer ingCounter.Add(-1)

		start := time.Now()

		resp, err := handler(ctx, req)

		if time.Since(start) > 1*time.Millisecond {
			_, err := fr.WriteTo(frOutPut)
			if err != nil {
				logger.Error("failed to write to frOutPut", "error", err)
			}
		}
		return resp, err
	}
}
