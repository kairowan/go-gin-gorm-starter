package router

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"go-gin-gorm-starter/internal/core/auth"
	"go-gin-gorm-starter/internal/feature/user"
	httpez "go-gin-gorm-starter/internal/transport/http/ez"
	mdw "go-gin-gorm-starter/internal/transport/http/middleware"
	"go-gin-gorm-starter/pkg/utils"
)

func NewAPIEngine(l *zap.Logger, db *gorm.DB, jwter *auth.JWTer) *gin.Engine {
	r := gin.New()

	// 中间件
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

	// 前缀
	api := r.Group("/api/v1")

	// 统一注册器（保留你原来的）
	MountAllAPI(api)

	// 鉴权分组（⚠️ /me 必须挂这里，才能拿到 userId）
	authUser := api.Group("")
	authUser.Use(mdw.AuthJWT(jwter, ""))

	// 用 Action 方式挂载：/auth/login（公共） 和 /me（鉴权）
	mountAuthActions(api, authUser, db, jwter)

	return r
}

// ---------- 动作注册：/auth/login + /me ----------

func mountAuthActions(api, authUser *gin.RouterGroup, db *gorm.DB, jwter *auth.JWTer) {
	// 确保用户表
	_ = db.AutoMigrate(&user.UserModel{})

	// 公共分组（无需登录）
	ezPublic := httpez.New(api)

	// /auth/login：查不到就自动注册 + 发 JWT
	type loginIn struct {
		Email    string `json:"email"    binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Name     string `json:"name"     binding:"omitempty,max=64"` // 首次注册可用
	}
	type loginOut struct {
		Token string      `json:"token"`
		IsNew bool        `json:"isNew"`
		User  interface{} `json:"user"`
	}
	httpez.RegisterAction[loginIn, loginOut](ezPublic, db, httpez.Action[loginIn, loginOut]{
		Method: http.MethodPost,
		Path:   "/auth/login",
		Binder: httpez.BindJSON,
		Auth:   false,
		Handler: func(c *gin.Context, tx *gorm.DB, in *loginIn) (loginOut, error) {
			email := strings.TrimSpace(in.Email)
			name := strings.TrimSpace(in.Name)

			var u user.UserModel
			err := tx.Where("email = ?", email).First(&u).Error

			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				// 自动注册
				if name == "" {
					if at := strings.IndexByte(email, '@'); at > 0 {
						name = email[:at]
					} else {
						name = "user"
					}
				}
				u = user.UserModel{
					ID:           utils.NewID(),
					Email:        email,
					Name:         name,
					PasswordHash: utils.HashPassword(in.Password),
					Role:         "user",
				}
				if e := tx.Create(&u).Error; e != nil {
					// 并发兜底：唯一冲突 → 再查一次
					if isDupKey(e) {
						if e2 := tx.Where("email = ?", email).First(&u).Error; e2 != nil {
							return loginOut{}, httpez.Internal("login failed", e2)
						}
					} else {
						return loginOut{}, httpez.BadRequest(e.Error())
					}
				}
				tok, e := jwter.Issue(u.ID, u.Role)
				if e != nil || tok == "" {
					return loginOut{}, httpez.Internal("issue token failed", e)
				}
				return loginOut{
					Token: tok, IsNew: true,
					User: gin.H{"id": u.ID, "email": u.Email, "name": u.Name, "role": u.Role},
				}, nil

			case err != nil:
				return loginOut{}, httpez.Internal("db error", err)

			default:
				// 已存在 → 校验密码
				if !utils.CheckPassword(in.Password, u.PasswordHash) {
					return loginOut{}, httpez.Unauthorized("invalid credentials")
				}
				tok, e := jwter.Issue(u.ID, u.Role)
				if e != nil || tok == "" {
					return loginOut{}, httpez.Internal("issue token failed", e)
				}
				return loginOut{
					Token: tok, IsNew: false,
					User: gin.H{"id": u.ID, "email": u.Email, "name": u.Name, "role": u.Role},
				}, nil
			}
		},
	})

	// 鉴权分组（需要登录）—— /me 必须挂在带中间件的分组
	ezAuth := httpez.New(authUser)

	type meOut struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}
	httpez.RegisterAction[struct{}, meOut](ezAuth, db, httpez.Action[struct{}, meOut]{
		Method: http.MethodGet,
		Path:   "/me",
		Binder: httpez.BindNone,
		Auth:   true, // 这里可以保留 true（双保险），也可以设为 false 因为分组已走中间件
		Handler: func(c *gin.Context, tx *gorm.DB, _ *struct{}) (meOut, error) {
			uid := c.GetString("userId")
			if uid == "" {
				return meOut{}, httpez.Unauthorized("unauthorized")
			}
			var u user.UserModel
			if err := tx.Where("id = ?", uid).First(&u).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return meOut{}, httpez.NotFound("user not found")
				}
				return meOut{}, httpez.Internal("db error", err)
			}
			return meOut{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role}, nil
		},
	})
}

func isDupKey(err error) bool {
	// 不依赖 gorm.ErrDuplicatedKey，避免版本差异导致“undefined”
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique violation") ||
		strings.Contains(msg, "duplicate key")
}
