package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const KeyRequestID = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.Request.Header.Get(KeyRequestID)
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Writer.Header().Set(KeyRequestID, rid)
		c.Set(KeyRequestID, rid)
		c.Next()
	}
}
