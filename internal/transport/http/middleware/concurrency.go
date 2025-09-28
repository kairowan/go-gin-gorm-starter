package middleware

import (
	"github.com/gin-gonic/gin"
	resp "go-gin-gorm-starter/internal/transport/http/response"
	"golang.org/x/sync/semaphore"
	"net/http"
)

// ConcurrencyLimit 限制同时在处理的请求数（保护 DB 下游）
func ConcurrencyLimit(max int64) gin.HandlerFunc {
	sem := semaphore.NewWeighted(max)
	return func(c *gin.Context) {
		if err := sem.Acquire(c, 1); err != nil {
			c.AbortWithStatusJSON(http.StatusOK, resp.Error(resp.CodeServerError, "server busy"))
			return
		}
		defer sem.Release(1)
		c.Next()
	}
}
