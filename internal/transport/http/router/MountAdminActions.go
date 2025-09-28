package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"go-gin-gorm-starter/internal/feature/user"
	httpez "go-gin-gorm-starter/internal/transport/http/ez"
)

// 把管理端接口集中在这里注册
func MountAdminActions(admin *gin.RouterGroup, db *gorm.DB) {
	_ = db.AutoMigrate(&user.UserModel{})

	ez := httpez.New(admin)

	// --- GET /admin/v1/users  用户列表 ---
	type listQ struct {
		Offset      int    `form:"offset,default=0"`
		Limit       int    `form:"limit,default=20"`
		Q           string `form:"q"`            // 按 email/name 模糊搜
		WithDeleted bool   `form:"with_deleted"` // 是否包含软删
	}
	type row struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}
	type listOut struct {
		Total int64 `json:"total"`
		Items []row `json:"items"`
	}

	httpez.RegisterAction[listQ, listOut](ez, db, httpez.Action[listQ, listOut]{
		Method: http.MethodGet,
		Path:   "/users",
		Binder: httpez.BindQuery,
		Auth:   false, // 分组已走 AuthJWT("admin")，这里可不再重复校验
		Roles:  nil,   // 也可留空
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
					ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role,
				})
			}
			return out, nil
		},
	})

	// --- POST /admin/v1/users/:id/ban  封禁（软删） ---
	httpez.RegisterAction[struct{}, gin.H](ez, db, httpez.Action[struct{}, gin.H]{
		Method: http.MethodPost,
		Path:   "/users/:id/ban",
		Binder: httpez.BindNone,
		Auth:   false, // 分组已校验 admin
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
