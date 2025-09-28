package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	resp "go-gin-gorm-starter/internal/transport/http/response"
)

func Timeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusOK, resp.Error(504, "timeout"))
		}
	}
}
