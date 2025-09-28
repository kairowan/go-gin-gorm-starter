package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpReqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "Count of HTTP requests"},
		[]string{"path", "method", "status"},
	)
	httpLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests",
			Buckets: prometheus.DefBuckets,
		}, []string{"path", "method"},
	)
)

func init() { prometheus.MustRegister(httpReqTotal, httpLatency) }

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		httpReqTotal.WithLabelValues(path, c.Request.Method, strconv.Itoa(c.Writer.Status())).Inc()
		httpLatency.WithLabelValues(path, c.Request.Method).Observe(time.Since(start).Seconds())
	}
}
