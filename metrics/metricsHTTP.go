package metrics

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

type HTTPMetrics struct {
	serviceName string

	// ОБЩИЕ RPC МЕТРИКИ (для всех сервисов)
	RPCRequestsTotal    *prometheus.CounterVec   // Счетчик запросов (для вычисления RPS)
	RPCRequestsDuration *prometheus.HistogramVec // Общее время
	RPCRequestsInFlight *prometheus.GaugeVec     // Активные запросы

	// ДЕТАЛИЗАЦИЯ ПО СТАТУСАМ
	StatusCodes *prometheus.CounterVec // 2xx, 3xx, 4xx, 5xx

	// HTTP-СПЕЦИФИЧНЫЕ МЕТРИКИ (по путям и методам)
	HTTPRequestsTotal    *prometheus.CounterVec   // Хиты по методам и путям
	HTTPRequestsDuration *prometheus.HistogramVec // Время по методам и путям
	HTTPRequestsErrors   *prometheus.CounterVec   // Ошибки по методам и путям
}

func NewHTTPMetrics(serviceName string) *HTTPMetrics {
	buckets := []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

	m := &HTTPMetrics{
		serviceName: serviceName,
	}

	// 1. ОБЩИЕ RPC МЕТРИКИ - счетчики для вычисления RPS
	m.RPCRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_requests_total", // Этот counter используется для rate() = RPS
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
			Name: "http_status_codes_total",
			Help: "Total HTTP responses by status code class",
		},
		[]string{"service", "status_class", "status_code"},
	)

	// 3. ДЕТАЛЬНЫЕ HTTP МЕТРИКИ
	m.HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total", // Этот counter используется для rate() = RPS по путям
			Help: "Total HTTP requests by method and path (use rate() for RPS)",
		},
		[]string{"method", "path", "service", "status_code"},
	)

	m.HTTPRequestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration by method and path",
			Buckets: buckets,
		},
		[]string{"method", "path", "service", "status_code"},
	)

	m.HTTPRequestsErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_errors_total",
			Help: "Total HTTP errors by method, path and error type",
		},
		[]string{"method", "path", "service", "error_type", "status_code"},
	)

	// Регистрируем все метрики
	metrics := []prometheus.Collector{
		m.RPCRequestsTotal, m.RPCRequestsDuration, m.RPCRequestsInFlight,
		m.StatusCodes, m.HTTPRequestsTotal, m.HTTPRequestsDuration,
		m.HTTPRequestsErrors,
	}

	for _, metric := range metrics {
		prometheus.MustRegister(metric)
	}

	return m
}

func (m *HTTPMetrics) recordAllMetrics(method, path string, statusCode int, duration time.Duration) {
	statusStr := strconv.Itoa(statusCode)
	statusClass := getStatusClass(statusCode)
	durationSeconds := duration.Seconds()

	// A. ОБЩИЕ RPC МЕТРИКИ - увеличиваем счетчики
	m.RPCRequestsTotal.WithLabelValues(m.serviceName, statusClass).Inc()
	m.RPCRequestsDuration.WithLabelValues(m.serviceName, statusClass).Observe(durationSeconds)

	// B. МЕТРИКИ СТАТУСОВ
	m.StatusCodes.WithLabelValues(m.serviceName, statusClass, statusStr).Inc()

	// C. ДЕТАЛЬНЫЕ HTTP МЕТРИКИ
	m.HTTPRequestsTotal.WithLabelValues(method, path, m.serviceName, statusStr).Inc()
	m.HTTPRequestsDuration.WithLabelValues(method, path, m.serviceName, statusStr).Observe(durationSeconds)

	// D. МЕТРИКИ ОШИБОК
	if statusCode >= 400 {
		errorType := getErrorType(statusCode)
		m.HTTPRequestsErrors.WithLabelValues(method, path, m.serviceName, errorType, statusStr).Inc()
	}
}

func (m *HTTPMetrics) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Увеличиваем счетчик активных запросов
		m.RPCRequestsInFlight.WithLabelValues(m.serviceName).Inc()

		// Перехватываем статус код
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		defer func() {
			if rec := recover(); rec != nil {
				if rw.statusCode < 400 {
					rw.statusCode = http.StatusInternalServerError
				}
				panic(rec)
			}
		}()

		// Вызываем следующий обработчик
		next.ServeHTTP(rw, r)

		// Вычисляем длительность
		duration := time.Since(start)

		// Уменьшаем счетчик активных запросов
		m.RPCRequestsInFlight.WithLabelValues(m.serviceName).Dec()

		// Получаем путь из роутера (уже нормализованный с плейсхолдерами)
		route := mux.CurrentRoute(r)
		var pathTemplate string
		if route != nil {
			pathTemplate, _ = route.GetPathTemplate()
		}

		// Если не удалось получить путь из роутера, используем raw path
		if pathTemplate == "" {
			pathTemplate = r.URL.Path
		}

		// Записываем все метрики
		m.recordAllMetrics(r.Method, pathTemplate, rw.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
	}
	return hj.Hijack()
}

func getStatusClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

func getErrorType(statusCode int) string {
	switch {
	case statusCode == 400:
		return "bad_request"
	case statusCode == 401:
		return "unauthorized"
	case statusCode == 403:
		return "forbidden"
	case statusCode == 404:
		return "not_found"
	case statusCode == 408:
		return "timeout"
	case statusCode == 409:
		return "conflict"
	case statusCode == 422:
		return "validation_error"
	case statusCode == 429:
		return "rate_limit"
	case statusCode >= 500 && statusCode < 600:
		return "server_error"
	default:
		return "client_error"
	}
}
