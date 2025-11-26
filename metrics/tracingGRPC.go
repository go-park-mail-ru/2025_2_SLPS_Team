package metrics

import (
	"context"
	"log"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor(tracer opentracing.Tracer) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Извлекаем метаданные
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Извлекаем span из метаданных
		var span opentracing.Span
		wireContext, err := tracer.Extract(
			opentracing.HTTPHeaders,
			metadataTextMap(md),
		)

		if err != nil {
			span = tracer.StartSpan(info.FullMethod)
		} else {
			span = tracer.StartSpan(
				info.FullMethod,
				opentracing.ChildOf(wireContext),
			)
		}
		defer span.Finish()

		// Устанавливаем теги
		ext.SpanKindRPCServer.Set(span)
		span.SetTag("grpc.method", info.FullMethod)

		// Добавляем span в контекст
		ctx = opentracing.ContextWithSpan(ctx, span)

		// Обрабатываем запрос
		resp, err := handler(ctx, req)

		// Устанавливаем теги результата
		if err != nil {
			s, _ := status.FromError(err)
			span.SetTag("grpc.status_code", s.Code().String())
			span.SetTag("error", true)
			span.SetTag("error.message", err.Error())
		} else {
			span.SetTag("grpc.status_code", "OK")
		}

		return resp, err
	}
}

// metadataTextMap адаптер для работы с gRPC metadata
type metadataTextMap metadata.MD

func (m metadataTextMap) Set(key, val string) {
	m[key] = append(m[key], val)
}

func (m metadataTextMap) ForeachKey(callback func(key, val string) error) error {
	for k, vals := range m {
		for _, v := range vals {
			if err := callback(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// UnaryClientInterceptor создает interceptor для трассировки gRPC клиентов
func UnaryClientInterceptor(tracer opentracing.Tracer) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var span opentracing.Span

		// Проверяем есть ли родительский span в контексте
		if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
			span = tracer.StartSpan(
				method,
				opentracing.ChildOf(parentSpan.Context()),
			)
		} else {
			span = tracer.StartSpan(method)
		}
		defer span.Finish()

		// Устанавливаем теги
		ext.SpanKindRPCClient.Set(span)
		span.SetTag("grpc.method", method)
		span.SetTag("grpc.target", cc.Target())

		// Инжектим span в метаданные для передачи на сервер
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		carrier := metadataTextMap(md)
		if err := tracer.Inject(span.Context(), opentracing.HTTPHeaders, carrier); err != nil {
			span.SetTag("injection.error", err.Error())
		}

		// Создаем новый контекст с метаданными
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Вызываем RPC метод
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Устанавливаем теги результата
		if err != nil {
			span.SetTag("error", true)
			span.SetTag("error.message", err.Error())
		}

		return err
	}
}

type tracedClientStream struct {
	grpc.ClientStream
	span opentracing.Span
}

func (tcs *tracedClientStream) CloseSend() error {
	err := tcs.ClientStream.CloseSend()
	if err != nil {
		tcs.span.SetTag("error", true)
		tcs.span.SetTag("error.message", err.Error())
	}
	return err
}

func (tcs *tracedClientStream) RecvMsg(m interface{}) error {
	err := tcs.ClientStream.RecvMsg(m)
	if err != nil {
		tcs.span.SetTag("error", true)
		tcs.span.SetTag("error.message", err.Error())
		tcs.span.Finish()
	}
	return err
}

func (tcs *tracedClientStream) SendMsg(m interface{}) error {
	err := tcs.ClientStream.SendMsg(m)
	if err != nil {
		tcs.span.SetTag("error", true)
		tcs.span.SetTag("error.message", err.Error())
	}
	return err
}

func initTracer() func(context.Context) error {
	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("my-go-service"),
		)),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown
}
