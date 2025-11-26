package grpc

import (
	"context"
	"fmt"
	"project/cmd/logger"
	"project/config"
	"project/domain"
	"strconv"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func ClientUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Получаем логгер из контекста
		logger := domain.FromContext(ctx)
		logger.Info("gRPC client request", zap.String("method", method))

		// Получаем userID, если есть
		userID, _ := ctx.Value(domain.UserIDKey).(int32)
		reqID, _ := ctx.Value("requestID").(string)
		// Создаем metadata с нужными полями
		md := metadata.Pairs(
			"request-id", reqID,
			"user-id", fmt.Sprintf("%d", userID),
		)

		// Добавляем к контексту
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Вызываем настоящий gRPC метод
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func ServerUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, _ := metadata.FromIncomingContext(ctx)

		reqID := ""
		if vals := md.Get("request-id"); len(vals) > 0 {
			reqID = vals[0]
		}
		userID := int32(0)
		if vals := md.Get("user-id"); len(vals) > 0 {
			parsed, _ := strconv.Atoi(vals[0])
			userID = int32(parsed)
		}
		logger := logger.NewLogger(config.NewConfig())
		logger = logger.With(zap.String("requestID", reqID), zap.Int32("selfUserID", userID))
		ctx = context.WithValue(ctx, domain.LoggerKey, logger)
		ctx = context.WithValue(ctx, domain.UserIDKey, userID)

		return handler(ctx, req)
	}
}
