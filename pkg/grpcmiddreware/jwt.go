package grpcmiddreware

import (
	"context"
	"im/pkg/jwt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func JwtUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return jwtUnaryInterceptor(ctx, logger, req, info, handler)
	}
}

var isIgnoreMethods = []string{
	"/apigateway.APIGateway/Login",
	"/apigateway.APIGateway/Register",
	"/apigateway.APIGateway/GetSessionUserList",
	"/apigateway.APIGateway/SendMessage",
}

func jwtUnaryInterceptor(ctx context.Context, logger *slog.Logger, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	for _, method := range isIgnoreMethods {
		if method == info.FullMethod {
			return handler(ctx, req)
		}
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "metadata is required")
	}
	token := md.Get("token")
	if len(token) == 0 {
		logger.Error("token is required")
		return nil, status.Errorf(codes.Unauthenticated, "token is required")
	}

	claims, err := jwt.ValidateToken(token[0])
	if err != nil {
		logger.Error("validate token error", "error", err, "token", token)
		return nil, status.Errorf(codes.Unauthenticated, "validate token error: %v", err)
	}

	userUUID, err := claims.GetSubject()
	if err != nil || len(userUUID) == 0 {
		logger.Error("validate claims error", "error", err)
		return nil, status.Errorf(codes.Unauthenticated, "validate claims error: %v", err)
	}
	logger.Debug("jwtUnaryInterceptor", "user_uuid", userUUID)
	ctx = context.WithValue(metadata.AppendToOutgoingContext(ctx, "user_uuid", userUUID), "user_uuid", userUUID)
	return handler(ctx, req)

}
