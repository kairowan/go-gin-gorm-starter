// internal/transport/http/router/admin.go
package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"go-gin-gorm-starter/internal/core/auth"
	mdw "go-gin-gorm-starter/internal/transport/http/middleware"
)

func NewAdminEngine(l *zap.Logger, db *gorm.DB, jwter *auth.JWTer) *gin.Engine {
	r := gin.New()

	r.Use(
		mdw.RequestID(),
		mdw.RateLimit(200, 400),
		mdw.ConcurrencyLimit(300),
		mdw.MaxBodyBytes(16<<20),
		mdw.Timeout(10*time.Second),
		mdw.SimpleRecovery(),
		mdw.Metrics(),
		mdw.AccessLog(l),
	)

	// 健康检查
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": 1}) })

	// 管理端 v1（统一要求 admin 角色）
	admin := r.Group("/admin/v1")
	admin.Use(mdw.AuthJWT(jwter, "admin"))

	// ① 自动发现（如有）
	MountAllAdmin(admin)

	// ② 用 Action 挂载管理端接口（用户列表/封禁等）
	MountAdminActions(admin, db)

	return r
}
