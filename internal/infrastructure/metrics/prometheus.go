package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HandlerMetrics struct {
	RequestCount    *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
}

type ServiceMetrics struct {
	MethodCount    *prometheus.CounterVec
	MethodDuration *prometheus.HistogramVec
}

type RepositoryMetrics struct {
	QueryCount    *prometheus.CounterVec
	QueryDuration *prometheus.HistogramVec
}

func NewHandlerMetrics() *HandlerMetrics {
	requestCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "handler_requests_total",
			Help: "Total number of HTTP requests handled by the handler layer.",
		},
		[]string{"method", "endpoint", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "handler_request_duration_seconds",
			Help:    "Histogram of response latency for handler in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	prometheus.MustRegister(requestCount, requestDuration)

	return &HandlerMetrics{
		RequestCount:    requestCount,
		RequestDuration: requestDuration,
	}
}

func NewServiceMetrics() *ServiceMetrics {
	methodCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "service_methods_total",
			Help: "Total number of service methods executed.",
		},
		[]string{"method", "status"},
	)

	methodDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "service_method_duration_seconds",
			Help:    "Histogram of service method execution duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)

	prometheus.MustRegister(methodCount, methodDuration)

	return &ServiceMetrics{
		MethodCount:    methodCount,
		MethodDuration: methodDuration,
	}
}

func NewRepositoryMetrics() *RepositoryMetrics {
	queryCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "repository_queries_total",
			Help: "Total number of database queries executed.",
		},
		[]string{"query", "status"},
	)

	queryDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "repository_query_duration_seconds",
			Help:    "Histogram of database query execution duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query", "status"},
	)

	prometheus.MustRegister(queryCount, queryDuration)

	return &RepositoryMetrics{
		QueryCount:    queryCount,
		QueryDuration: queryDuration,
	}
}

func (hm *HandlerMetrics) HTTPHandler() http.Handler {
	return promhttp.Handler()
}
