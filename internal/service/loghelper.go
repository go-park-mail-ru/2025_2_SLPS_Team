package service

import (
	"context"
	"project/domain"

	"go.uber.org/zap"
)

func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(domain.LoggerKey).(*zap.Logger); ok {
		return l
	}
	return zap.NewNop()
}

func Info(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Info(msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Error(msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Warn(msg, fields...)
}
