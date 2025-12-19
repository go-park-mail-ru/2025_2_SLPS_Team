package metrics

import (
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// HTTPMiddleware создает middleware для трассировки HTTP запросов
func HTTPMiddleware(tracer opentracing.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Пытаемся извлечь span из заголовков (для распределенной трассировки)
			var span opentracing.Span
			wireContext, err := tracer.Extract(
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(r.Header),
			)

			if err != nil {
				// Если не удалось извлечь, создаем новый root span
				span = tracer.StartSpan(r.URL.Path)
			} else {
				// Продолжаем существующую трассировку
				span = tracer.StartSpan(
					r.URL.Path,
					opentracing.ChildOf(wireContext),
				)
			}
			defer span.Finish()

			// Устанавливаем теги для span
			ext.SpanKindRPCServer.Set(span)
			ext.HTTPMethod.Set(span, r.Method)
			ext.HTTPUrl.Set(span, r.URL.String())
			span.SetTag("http.user_agent", r.UserAgent())

			// Добавляем span в контекст запроса
			ctx := opentracing.ContextWithSpan(r.Context(), span)

			// Создаем обертку для ResponseWriter для отслеживания статуса
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Передаем запрос дальше
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Устанавливаем теги после обработки
			ext.HTTPStatusCode.Set(span, uint16(rw.statusCode))
			if rw.statusCode >= 400 {
				span.SetTag("error", true)
			}
		})
	}
}
