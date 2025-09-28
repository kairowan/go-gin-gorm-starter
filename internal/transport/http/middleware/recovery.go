package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	resp "go-gin-gorm-starter/internal/transport/http/response"
)

func SimpleRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.AbortWithStatusJSON(http.StatusOK, resp.Error(resp.CodeServerError, "internal error"))
			}
		}()
		c.Next()
	}
}
