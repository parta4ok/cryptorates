package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	},
	[]string{"method", "path", "status"},
)

var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "path", "status"},
)

func MustRegister() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
	)
}

func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		labels := []string{r.Method, r.URL.Path, strconv.Itoa(ww.status)}

		httpRequestsTotal.WithLabelValues(labels...).Inc()
		httpRequestDuration.WithLabelValues(labels...).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
