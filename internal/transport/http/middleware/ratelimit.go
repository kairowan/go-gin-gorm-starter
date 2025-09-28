package middleware

import (
	"github.com/gin-gonic/gin"
	resp "go-gin-gorm-starter/internal/transport/http/response"
	"golang.org/x/time/rate"
	"net/http"
)

// RateLimit 全局令牌桶限速
func RateLimit(rps rate.Limit, burst int) gin.HandlerFunc {
	lim := rate.NewLimiter(rps, burst)
	return func(c *gin.Context) {
		if lim.Allow() {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusOK, resp.Error(resp.CodeServerError, "too many requests"))
	}
}

// RateLimitPerIP 每 IP 限速
func RateLimitPerIP(rps rate.Limit, burst int) gin.HandlerFunc {
	buckets := make(map[string]*rate.Limiter)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		lim, ok := buckets[ip]
		if !ok {
			lim = rate.NewLimiter(rps, burst)
			buckets[ip] = lim
		}
		if lim.Allow() {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusOK, resp.Error(resp.CodeServerError, "too many requests"))
	}
}
