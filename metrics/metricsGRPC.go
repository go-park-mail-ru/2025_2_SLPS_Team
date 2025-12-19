package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCMetrics struct {
	serviceName string

	// ОБЩИЕ RPC МЕТРИКИ (совместимы с HTTP метриками)
	RPCRequestsTotal    *prometheus.CounterVec
	RPCRequestsDuration *prometheus.HistogramVec
	RPCRequestsInFlight *prometheus.GaugeVec

	// ДЕТАЛИЗАЦИЯ ПО СТАТУСАМ
	StatusCodes *prometheus.CounterVec

	// gRPC-СПЕЦИФИЧНЫЕ МЕТРИКИ
	GRPCRequestsTotal    *prometheus.CounterVec   // Хиты по сервисам и методам
	GRPCRequestsDuration *prometheus.HistogramVec // Время по сервисам и методам
	GRPCRequestsErrors   *prometheus.CounterVec   // Ошибки по сервисам и методам
}

func NewGRPCMetrics(serviceName string) *GRPCMetrics {
	buckets := []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

	m := &GRPCMetrics{
		serviceName: serviceName,
	}

	// 1. ОБЩИЕ RPC МЕТРИКИ (совместимы с HTTP)
	m.RPCRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_requests_total",
			Help: "Total number of RPC requests (use rate() for RPS)",
		},
		[]string{"service", "status_class"}, // auth-service, 2xx/4xx/5xx
	)

	m.RPCRequestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rpc_request_duration_seconds",
			Help:    "RPC request duration across all services",
			Buckets: buckets,
		},
		[]string{"service", "status_class"},
	)

	m.RPCRequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rpc_requests_in_flight",
			Help: "Current number of RPC requests being processed across all services",
		},
		[]string{"service"},
	)

	// 2. МЕТРИКИ СТАТУСОВ
	m.StatusCodes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_status_codes_total",
			Help: "Total gRPC responses by status code class",
		},
		[]string{"service", "status_class", "grpc_code"},
	)

	// 3. ДЕТАЛЬНЫЕ gRPC МЕТРИКИ
	m.GRPCRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total gRPC requests by service and method",
		},
		[]string{"grpc_service", "grpc_method", "service", "grpc_code"},
	)

	m.GRPCRequestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "gRPC request duration by service and method",
			Buckets: buckets,
		},
		[]string{"grpc_service", "grpc_method", "service", "grpc_code"},
	)

	m.GRPCRequestsErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_errors_total",
			Help: "Total gRPC errors by service, method and error type",
		},
		[]string{"grpc_service", "grpc_method", "service", "error_type", "grpc_code"},
	)

	// Регистрируем все метрики
	metrics := []prometheus.Collector{
		m.RPCRequestsTotal, m.RPCRequestsDuration, m.RPCRequestsInFlight,
		m.StatusCodes, m.GRPCRequestsTotal, m.GRPCRequestsDuration,
		m.GRPCRequestsErrors,
	}

	for _, metric := range metrics {
		prometheus.MustRegister(metric)
	}

	return m
}

// UnaryServerInterceptor для серверной стороны
func (m *GRPCMetrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Увеличиваем счетчик активных запросов
		m.RPCRequestsInFlight.WithLabelValues(m.serviceName).Inc()

		// Парсим информацию о методе
		grpcService, grpcMethod := parseGRPCMethod(info.FullMethod)

		// Вызываем обработчик
		resp, err := handler(ctx, req)

		// Получаем gRPC статус код
		grpcCode := status.Code(err)
		grpcCodeStr := grpcCode.String()
		statusClass := grpcCodeToStatusClass(grpcCode)

		// Уменьшаем счетчик активных запросов
		m.RPCRequestsInFlight.WithLabelValues(m.serviceName).Dec()

		// Вычисляем длительность
		duration := time.Since(start)

		// Записываем все метрики
		m.recordAllMetrics(grpcService, grpcMethod, grpcCodeStr, statusClass, duration, err)

		return resp, err
	}
}

// StreamServerInterceptor для streaming сервера
func (m *GRPCMetrics) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Увеличиваем счетчик активных запросов
		m.RPCRequestsInFlight.WithLabelValues(m.serviceName).Inc()

		// Парсим информацию о методе
		grpcService, grpcMethod := parseGRPCMethod(info.FullMethod)

		// Вызываем обработчик
		err := handler(srv, ss)

		// Получаем gRPC статус код
		grpcCode := status.Code(err)
		grpcCodeStr := grpcCode.String()
		statusClass := grpcCodeToStatusClass(grpcCode)

		// Уменьшаем счетчик активных запросов
		m.RPCRequestsInFlight.WithLabelValues(m.serviceName).Dec()

		// Вычисляем длительность
		duration := time.Since(start)

		// Записываем все метрики
		m.recordAllMetrics(grpcService, grpcMethod, grpcCodeStr, statusClass, duration, err)

		return err
	}
}

// recordAllMetrics записывает все виды метрик для gRPC запроса
func (m *GRPCMetrics) recordAllMetrics(grpcService, grpcMethod, grpcCode, statusClass string, duration time.Duration, err error) {
	durationSeconds := duration.Seconds()

	// A. ОБЩИЕ RPC МЕТРИКИ
	m.RPCRequestsTotal.WithLabelValues(m.serviceName, statusClass).Inc()
	m.RPCRequestsDuration.WithLabelValues(m.serviceName, statusClass).Observe(durationSeconds)

	// B. МЕТРИКИ СТАТУСОВ
	m.StatusCodes.WithLabelValues(m.serviceName, statusClass, grpcCode).Inc()

	// C. ДЕТАЛЬНЫЕ gRPC МЕТРИКИ
	m.GRPCRequestsTotal.WithLabelValues(grpcService, grpcMethod, m.serviceName, grpcCode).Inc()
	m.GRPCRequestsDuration.WithLabelValues(grpcService, grpcMethod, m.serviceName, grpcCode).Observe(durationSeconds)

	// D. МЕТРИКИ ОШИБОК
	if err != nil {
		errorType := grpcCodeToErrorType(status.Code(err))
		m.GRPCRequestsErrors.WithLabelValues(grpcService, grpcMethod, m.serviceName, errorType, grpcCode).Inc()
	}
}

// Дополнительные методы для ручной записи ошибок
func (m *GRPCMetrics) RecordError(grpcService, grpcMethod, errorType string, grpcCode codes.Code) {
	grpcCodeStr := grpcCode.String()
	statusClass := grpcCodeToStatusClass(grpcCode)
	m.GRPCRequestsErrors.WithLabelValues(grpcService, grpcMethod, m.serviceName, errorType, grpcCodeStr).Inc()

	// Также записываем в общие метрики
	m.RPCRequestsTotal.WithLabelValues(m.serviceName, statusClass).Inc()
}

func (m *GRPCMetrics) RecordBusinessError(grpcService, grpcMethod, errorCode string) {
	m.GRPCRequestsErrors.WithLabelValues(grpcService, grpcMethod, m.serviceName, "business_error", errorCode).Inc()
}

func (m *GRPCMetrics) RecordTimeout(grpcService, grpcMethod string) {
	m.GRPCRequestsErrors.WithLabelValues(grpcService, grpcMethod, m.serviceName, "timeout", codes.DeadlineExceeded.String()).Inc()
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ====================

// parseGRPCMethod парсит полное имя gRPC метода
func parseGRPCMethod(fullMethod string) (string, string) {
	// Формат: "/package.Service/Method"
	// Пример: "/auth.AuthService/Login"

	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}

	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			service := fullMethod[:i]  // "auth.AuthService"
			method := fullMethod[i+1:] // "Login"
			return service, method
		}
	}

	return "unknown", "unknown"
}

// grpcCodeToStatusClass преобразует gRPC код в класс статуса
func grpcCodeToStatusClass(code codes.Code) string {
	switch code {
	case codes.OK:
		return "2xx"
	case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition,
		codes.OutOfRange, codes.Unimplemented, codes.Aborted:
		return "4xx"
	case codes.Unknown, codes.DeadlineExceeded, codes.ResourceExhausted,
		codes.Internal, codes.Unavailable, codes.DataLoss:
		return "5xx"
	default:
		return "unknown"
	}
}

// grpcCodeToErrorType преобразует gRPC код в тип ошибки
func grpcCodeToErrorType(code codes.Code) string {
	switch code {
	case codes.Canceled:
		return "canceled"
	case codes.Unknown:
		return "unknown"
	case codes.InvalidArgument:
		return "invalid_argument"
	case codes.DeadlineExceeded:
		return "deadline_exceeded"
	case codes.NotFound:
		return "not_found"
	case codes.AlreadyExists:
		return "already_exists"
	case codes.PermissionDenied:
		return "permission_denied"
	case codes.ResourceExhausted:
		return "resource_exhausted"
	case codes.FailedPrecondition:
		return "failed_precondition"
	case codes.Aborted:
		return "aborted"
	case codes.OutOfRange:
		return "out_of_range"
	case codes.Unimplemented:
		return "unimplemented"
	case codes.Internal:
		return "internal"
	case codes.Unavailable:
		return "unavailable"
	case codes.DataLoss:
		return "data_loss"
	case codes.Unauthenticated:
		return "unauthenticated"
	default:
		return "unknown_error"
	}
}
