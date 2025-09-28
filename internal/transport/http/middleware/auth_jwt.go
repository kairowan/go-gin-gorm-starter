package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-gin-gorm-starter/internal/core/auth"
	resp "go-gin-gorm-starter/internal/transport/http/response"
)

func AuthJWT(j *auth.JWTer, requireRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ah := c.GetHeader("Authorization")
		if !strings.HasPrefix(ah, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusOK, resp.Error(resp.CodeUnauthorized, "missing token"))
			return
		}
		claims, err := j.Parse(strings.TrimPrefix(ah, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusOK, resp.Error(resp.CodeUnauthorized, "invalid token"))
			return
		}
		if requireRole != "" && claims.Role != requireRole {
			c.AbortWithStatusJSON(http.StatusOK, resp.Error(resp.CodeForbidden, "forbidden"))
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}
