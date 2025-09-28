package ez

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	resp "go-gin-gorm-starter/internal/transport/http/response"
	"go-gin-gorm-starter/pkg/utils"
)

// Hook
type CrudHooks[T any] struct {
	BeforeCreate func(c *gin.Context, m *T) error
	BeforeUpdate func(c *gin.Context, m *T) error
	ScopeList    func(c *gin.Context, q *gorm.DB) *gorm.DB // 自定义筛选/排序
	AfterGet     func(c *gin.Context, m *T)
}

type CrudConfig[T any] struct {
	DB    *gorm.DB
	Group *gin.RouterGroup // 已鉴权分组（能拿 userId）
	Path  string
	New   func() *T

	Hooks CrudHooks[T]

	AllowCreate bool
	AllowList   bool
	AllowGet    bool
	AllowUpdate bool
	AllowDelete bool

	IDField    string // 默认 "ID"
	OwnerField string // 默认优先 "OwnerID"，其次 "UserID"/"UID"

	AutoID bool          // 默认 true
	IDGen  func() string // 默认 utils.NewID

	// 列表排序（列名按模型字段自动转 snake_case），为空则按 ID DESC
	OrderBy string // 例如 "CreatedAt DESC"
}

// 反射 & 工具
func (c *CrudConfig[T]) idFieldCandidates() []string {
	if c.IDField != "" {
		return []string{c.IDField, "ID", "Id"}
	}
	return []string{"ID", "Id"}
}

func (c *CrudConfig[T]) ownerFieldCandidates() []string {
	if c.OwnerField != "" {
		return []string{c.OwnerField, "OwnerID", "UserID", "UID"}
	}
	return []string{"OwnerID", "UserID", "UID"}
}

func getStringFieldPtr(obj any, candidates []string) (*string, bool) {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		return nil, false
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return nil, false
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// 未导出字段跳过
		if f.PkgPath != "" {
			continue
		}
		for _, cand := range candidates {
			if f.Name == cand {
				fv := v.Field(i)
				if fv.Kind() == reflect.String && fv.CanSet() {
					p := fv.Addr().Interface().(*string)
					return p, true
				}
			}
		}
	}
	return nil, false
}

func readStringField(obj any, candidates []string) (string, bool) {
	p, ok := getStringFieldPtr(obj, candidates)
	if !ok {
		return "", false
	}
	return *p, true
}

func writeStringField(obj any, candidates []string, val string) bool {
	p, ok := getStringFieldPtr(obj, candidates)
	if !ok {
		return false
	}
	*p = val
	return true
}

func atoiDefault(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil && v > 0 {
		return v
	}
	return def
}

func toSnake(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// CRUD 注册（无需模型实现任何接口）
func Crud[T any](cfg CrudConfig[T]) {
	// 默认放开所有操作
	if !cfg.AllowCreate && !cfg.AllowGet && !cfg.AllowList && !cfg.AllowUpdate && !cfg.AllowDelete {
		cfg.AllowCreate, cfg.AllowList, cfg.AllowGet, cfg.AllowUpdate, cfg.AllowDelete = true, true, true, true, true
	}
	// 默认 AutoID/IDGen
	if cfg.AutoID == false && cfg.IDGen == nil {
		// 显式设置了 AutoID=false 则不生成；否则默认生成
		cfg.AutoID = true
	}
	if cfg.IDGen == nil {
		cfg.IDGen = utils.NewID
	}
	// 自动迁移
	_ = cfg.DB.AutoMigrate(cfg.New())

	idFieldNames := cfg.idFieldCandidates()
	ownerFieldNames := cfg.ownerFieldCandidates()

	// Create
	if cfg.AllowCreate {
		cfg.Group.POST(cfg.Path, func(c *gin.Context) {
			m := cfg.New()
			if err := c.ShouldBindJSON(m); err != nil {
				c.JSON(http.StatusOK, resp.Error(resp.CodeBadRequest, err.Error()))
				return
			}
			uid := c.GetString("userId")
			if uid == "" {
				c.JSON(http.StatusOK, resp.Error(resp.CodeUnauthorized, "unauthorized"))
				return
			}
			// 自动生成 ID（若开启且为空）
			if cfg.AutoID {
				if id, ok := readStringField(m, idFieldNames); !ok {
					c.JSON(http.StatusOK, resp.Error(resp.CodeBadRequest, "id field not found"))
					return
				} else if strings.TrimSpace(id) == "" {
					_ = writeStringField(m, idFieldNames, cfg.IDGen())
				}
			}
			// 写 Owner
			if !writeStringField(m, ownerFieldNames, uid) {
				c.JSON(http.StatusOK, resp.Error(resp.CodeBadRequest, "owner field not found"))
				return
			}

			if cfg.Hooks.BeforeCreate != nil {
				if err := cfg.Hooks.BeforeCreate(c, m); err != nil {
					c.JSON(http.StatusOK, resp.Error(resp.CodeBadRequest, err.Error()))
					return
				}
			}
			if err := cfg.DB.WithContext(c).Create(m).Error; err != nil {
				c.JSON(http.StatusOK, resp.Error(resp.CodeBadRequest, err.Error()))
				return
			}
			if cfg.Hooks.AfterGet != nil {
				cfg.Hooks.AfterGet(c, m)
			}
			c.JSON(http.StatusOK, resp.OK(m))
		})
	}

	// List（我的）
	if cfg.AllowList {
		cfg.Group.GET(cfg.Path, func(c *gin.Context) {
			uid := c.GetString("userId")
			if uid == "" {
				c.JSON(http.StatusOK, resp.Error(resp.CodeUnauthorized, "unauthorized"))
				return
			}
			page := atoiDefault(c.Query("page"), 1)
			size := atoiDefault(c.Query("size"), 20)
			if size <= 0 || size > 100 {
				size = 20
			}
			offset := (page - 1) * size

			// 用结构体 Where 自动映射列名，避免手写 owner_id
			ownerFilter := cfg.New()
			if !writeStringField(ownerFilter, ownerFieldNames, uid) {
				c.JSON(http.StatusOK, resp.Error(resp.CodeBadRequest, "owner field not found"))
				return
			}

			q := cfg.DB.WithContext(c).Model(cfg.New()).Where(ownerFilter)
			if cfg.Hooks.ScopeList != nil {
				q = cfg.Hooks.ScopeList(c, q)
			}

			var total int64
			if err := q.Count(&total).Error; err != nil {
				c.JSON(http.StatusOK, resp.Error(resp.CodeServerError, err.Error()))
				return
			}

			var items []T
			// 动态排序：优先按配置 OrderBy，否则按 ID DESC
			if cfg.OrderBy != "" {
				q = q.Order(cfg.OrderBy)
			} else {
				// 用字段名转 snake_case（例如 ID -> id）
				idCol := toSnake(idFieldNames[0])
				if idCol == "" {
					idCol = "id"
				}
				q = q.Order(clause.OrderByColumn{Column: clause.Column{Name: idCol}, Desc: true})
			}
			if err := q.Limit(size).Offset(offset).Find(&items).Error; err != nil {
				c.JSON(http.StatusOK, resp.Error(resp.CodeServerError, err.Error()))
				return
			}
			if cfg.Hooks.AfterGet != nil {
				for i := range items {
					cfg.Hooks.AfterGet(c, &items[i])
				}
			}
			c.JSON(http.StatusOK, resp.OK(gin.H{
				"list": items, "total": total, "page": page, "size": size,
			}))
		})
	}

	// Get
	if cfg.AllowGet {
		cfg.Group.GET(cfg.Path+"/:id", func(c *gin.Context) {
			uid := c.GetString("userId")
			if uid == "" {
				c.JSON(http.StatusOK, resp.Error(resp.CodeUnauthorized, "unauthorized"))
				return
			}
			id := c.Param("id")

			filter := cfg.New()
			_ = writeStringField(filter, idFieldNames, id)
			_ = writeStringField(filter, ownerFieldNames, uid)

			m := cfg.New()
			if err := cfg.DB.WithContext(c).Where(filter).First(m).Error; err != nil {
				c.JSON(http.StatusOK, resp.Error(resp.CodeNotFound, "not found"))
				return
			}
			if cfg.Hooks.AfterGet != nil {
				cfg.Hooks.AfterGet(c, m)
			}
			c.JSON(http.StatusOK, resp.OK(m))
		})
	}

	// Update
	if cfg.AllowUpdate {
		cfg.Group.PUT(cfg.Path+"/:id", func(c *gin.Context) {
			uid := c.GetString("userId")
			if uid == "" {
				c.JSON(http.StatusOK, resp.Error(resp.CodeUnauthorized, "unauthorized"))
				return
			}
			id := c.Param("id")

			// 先确认归属
			check := cfg.New()
			_ = writeStringField(check, idFieldNames, id)
			_ = writeStringField(check, ownerFieldNames, uid)
			if err := cfg.DB.WithContext(c).Where(check).First(check).Error; err != nil {
				c.JSON(http.StatusOK, resp.Error(resp.CodeNotFound, "not found"))
				return
			}

			in := cfg.New()
			if err := c.ShouldBindJSON(in); err != nil {
				c.JSON(http.StatusOK, resp.Error(resp.CodeBadRequest, err.Error()))
				return
			}
			// 强制保持 ID/Owner
			_ = writeStringField(in, idFieldNames, id)
			_ = writeStringField(in, ownerFieldNames, uid)

			if cfg.Hooks.BeforeUpdate != nil {
				if err := cfg.Hooks.BeforeUpdate(c, in); err != nil {
					c.JSON(http.StatusOK, resp.Error(resp.CodeBadRequest, err.Error()))
					return
				}
			}
			if err := cfg.DB.WithContext(c).Model(cfg.New()).Where(check).Updates(in).Error; err != nil {
				c.JSON(http.StatusOK, resp.Error(resp.CodeBadRequest, err.Error()))
				return
			}
			if cfg.Hooks.AfterGet != nil {
				cfg.Hooks.AfterGet(c, in)
			}
			c.JSON(http.StatusOK, resp.OK(gin.H{"id": id}))
		})
	}

	// Delete
	if cfg.AllowDelete {
		cfg.Group.DELETE(cfg.Path+"/:id", func(c *gin.Context) {
			uid := c.GetString("userId")
			if uid == "" {
				c.JSON(http.StatusOK, resp.Error(resp.CodeUnauthorized, "unauthorized"))
				return
			}
			id := c.Param("id")

			filter := cfg.New()
			_ = writeStringField(filter, idFieldNames, id)
			_ = writeStringField(filter, ownerFieldNames, uid)

			res := cfg.DB.WithContext(c).Where(filter).Delete(cfg.New())
			if res.Error != nil {
				c.JSON(http.StatusOK, resp.Error(resp.CodeServerError, res.Error.Error()))
				return
			}
			if res.RowsAffected == 0 {
				c.JSON(http.StatusOK, resp.Error(resp.CodeNotFound, "not found"))
				return
			}
			c.JSON(http.StatusOK, resp.OK(gin.H{"id": id}))
		})
	}
}
