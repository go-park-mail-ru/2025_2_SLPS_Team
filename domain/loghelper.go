package domain

import (
	"context"
	"reflect"

	"go.uber.org/zap"
)

func StructName(i interface{}) string {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(LoggerKey).(*zap.Logger); ok {
		return l
	}
	return zap.NewNop()
}

func Info(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Info(msg, fields...)
}

func Error(ctx context.Context, msg string, err error, fields ...zap.Field) {
	fields = append(fields, zap.Error(err))
	FromContext(ctx).Error(msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Warn(msg, fields...)
}

func DBLogger(ctx context.Context, repo string) *zap.Logger {
	base := FromContext(ctx)

	return base.With(
		zap.String("layer", "dbconn"),
		zap.String("repo", repo),
	)
}
