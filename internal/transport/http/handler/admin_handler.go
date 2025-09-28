package router

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"go-gin-gorm-starter/internal/feature/user"
	httpez "go-gin-gorm-starter/internal/transport/http/ez"
)

func MountAdminActions(admin *gin.RouterGroup, db *gorm.DB) {
	_ = db.AutoMigrate(&user.UserModel{})

	ezAdmin := httpez.New(admin)

	// --- 用户列表 ---
	type listQ struct {
		Offset      int    `form:"offset,default=0"`
		Limit       int    `form:"limit,default=20"`
		Q           string `form:"q"`            // 可选：按 email/name 模糊搜
		WithDeleted bool   `form:"with_deleted"` // 是否包含软删
	}
	type row struct {
		ID        string    `json:"id"`
		Email     string    `json:"email"`
		Name      string    `json:"name"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"createdAt"`
	}
	type listOut struct {
		Total int64 `json:"total"`
		Items []row `json:"items"`
	}

	httpez.RegisterAction[listQ, listOut](ezAdmin, db, httpez.Action[listQ, listOut]{
		Method: http.MethodGet,
		Path:   "/users",
		Binder: httpez.BindQuery,
		Auth:   true,
		Roles:  []string{"admin"}, // 需要 admin 角色
		Handler: func(c *gin.Context, tx *gorm.DB, in *listQ) (listOut, error) {
			if in.Limit <= 0 || in.Limit > 100 {
				in.Limit = 20
			}
			q := tx.WithContext(c).Model(&user.UserModel{})
			if in.WithDeleted {
				q = q.Unscoped()
			}
			if s := strings.TrimSpace(in.Q); s != "" {
				like := "%" + s + "%"
				q = q.Where("email LIKE ? OR name LIKE ?", like, like)
			}

			var total int64
			if err := q.Count(&total).Error; err != nil {
				return listOut{}, httpez.Internal("count users failed", err)
			}

			var us []user.UserModel
			if err := q.Order("created_at DESC").Limit(in.Limit).Offset(in.Offset).Find(&us).Error; err != nil {
				return listOut{}, httpez.Internal("list users failed", err)
			}

			out := listOut{Total: total, Items: make([]row, 0, len(us))}
			for _, u := range us {
				out.Items = append(out.Items, row{
					ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, CreatedAt: u.CreatedAt,
				})
			}
			return out, nil
		},
	})

	// --- 封禁（软删） ---
	httpez.RegisterAction[struct{}, gin.H](ezAdmin, db, httpez.Action[struct{}, gin.H]{
		Method: http.MethodPost,
		Path:   "/users/:id/ban",
		Binder: httpez.BindNone,
		Auth:   true,
		Roles:  []string{"admin"},
		Handler: func(c *gin.Context, tx *gorm.DB, _ *struct{}) (gin.H, error) {
			id := c.Param("id")
			if id == "" {
				return nil, httpez.BadRequest("missing id")
			}
			res := tx.WithContext(c).Where("id = ?", id).Delete(&user.UserModel{})
			if res.Error != nil {
				return nil, httpez.Internal("ban user failed", res.Error)
			}
			if res.RowsAffected == 0 {
				return nil, httpez.NotFound("user not found")
			}
			return gin.H{"id": id}, nil
		},
	})
}
