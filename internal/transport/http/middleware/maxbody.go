package middleware

import (
	"github.com/gin-gonic/gin"
	resp "go-gin-gorm-starter/internal/transport/http/response"
	"net/http"
)

// MaxBodyBytes 限制请求体大小（16MB）
func MaxBodyBytes(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
		if c.Err() != nil && !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusOK, resp.Error(resp.CodeBadRequest, "request body too large"))
		}
	}
}
