package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type respWriter struct {
	gin.ResponseWriter
	status int
	size   int
}

func (w *respWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func (w *respWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

func AccessLog(l *zap.Logger) gin.HandlerFunc {
	// 敏感字段 key（query/form/body 中统一按 key）
	sensitiveKeys := map[string]struct{}{
		"password": {}, "pwd": {}, "token": {}, "authorization": {},
		"secret": {}, "client_secret": {}, "access_token": {},
	}

	mask := func(kv map[string][]string) map[string][]string {
		out := map[string][]string{}
		for k, v := range kv {
			lk := strings.ToLower(k)
			if _, ok := sensitiveKeys[lk]; ok {
				out[k] = []string{"****"}
			} else {
				out[k] = v
			}
		}
		return out
	}

	return func(c *gin.Context) {
		start := time.Now()
		w := &respWriter{ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		q := mask(c.Request.URL.Query())
		// 打印摘要：method/path/status/latency/ip/ua/query/size
		l.Info("HTTP",
			zap.String("rid", c.GetString("rid")),
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.Int("status", w.status),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
			zap.String("ua", c.Request.UserAgent()),
			zap.Any("query", q),
			zap.Int("size", w.size),
		)
	}
}
