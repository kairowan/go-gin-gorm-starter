package router

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	resp "go-gin-gorm-starter/internal/transport/http/response"
)

/* ================== 你原有的轻封装 ================== */

type EZ struct{ g *gin.RouterGroup }

func New(g *gin.RouterGroup) EZ { return EZ{g: g} }

func (e EZ) GET(path string, h func(c *gin.Context) (any, error)) {
	e.g.GET(path, func(c *gin.Context) {
		data, err := h(c)
		if err != nil {
			c.JSON(http.StatusOK, resp.Error(500, err.Error()))
			return
		}
		c.JSON(http.StatusOK, resp.OK(data))
	})
}

func POST[T any](e EZ, path string, h func(c *gin.Context, in T) (any, error)) {
	e.g.POST(path, func(c *gin.Context) {
		var in T
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusOK, resp.Error(400, err.Error()))
			return
		}
		data, err := h(c, in)
		if err != nil {
			c.JSON(http.StatusOK, resp.Error(500, err.Error()))
			return
		}
		c.JSON(http.StatusOK, resp.OK(data))
	})
}

// 处理 multipart/form-data 多文件上传
func POSTFILES(e EZ, path string, fieldName string, h func(c *gin.Context, files []*multipart.FileHeader) (any, error)) {
	e.g.POST(path, func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusOK, resp.Error(400, "invalid multipart form: "+err.Error()))
			return
		}
		files := form.File[fieldName]
		if len(files) == 0 {
			c.JSON(http.StatusOK, resp.Error(400, "no files uploaded"))
			return
		}

		data, err := h(c, files)
		if err != nil {
			c.JSON(http.StatusOK, resp.Error(500, err.Error()))
			return
		}
		c.JSON(http.StatusOK, resp.OK(data))
	})
}

/* ================== 新增：Action（非 CRUD 一行注册） ================== */

// 绑定方式
type Binder string

const (
	BindJSON  Binder = "json"  // 从 JSON 绑定
	BindQuery Binder = "query" // 从 URL ?a=b 绑定
	BindNone  Binder = "none"  // 不绑定，自己从 c.Param / c.PostForm 取
)

// 统一错误对象（配合 resp.Error(int, msg)）
type AErr struct {
	Code int
	Msg  string
	Err  error
}

func (e *AErr) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "action error"
}
func BadRequest(msg string) error   { return &AErr{Code: 400, Msg: msg} }
func Unauthorized(msg string) error { return &AErr{Code: 401, Msg: msg} }
func Forbidden(msg string) error    { return &AErr{Code: 403, Msg: msg} }
func NotFound(msg string) error     { return &AErr{Code: 404, Msg: msg} }
func Internal(msg string, err error) error {
	// 如果你项目里 500 常量名不是 500，可改成 resp.CodeInternal
	return &AErr{Code: 500, Msg: msg, Err: err}
}

// 动作定义：I 入参，O 出参
type Action[I any, O any] struct {
	Method  string   // "GET" | "POST" | "PUT" | "DELETE"
	Path    string   // 例："/auth/login"、"/orders/:id/pay"
	Binder  Binder   // 绑定方式
	Auth    bool     // 是否要求登录（检查 userId）
	Roles   []string // 限定角色（可选）
	UseTx   bool     // 是否包事务（gorm.Transaction）
	Handler func(c *gin.Context, db *gorm.DB, in *I) (O, error)
}

// 在当前 EZ 下注册动作接口（传入 *gorm.DB）
func RegisterAction[I any, O any](e EZ, db *gorm.DB, a Action[I, O]) {
	h := func(c *gin.Context) {
		// 1) 鉴权/角色
		if a.Auth {
			uid := c.GetString("userId")
			if uid == "" {
				c.JSON(http.StatusOK, resp.Error(401, "unauthorized"))
				return
			}
			if len(a.Roles) > 0 {
				role := c.GetString("role")
				ok := false
				for _, r := range a.Roles {
					if role == r {
						ok = true
						break
					}
				}
				if !ok {
					c.JSON(http.StatusOK, resp.Error(403, "forbidden"))
					return
				}
			}
		}

		// 2) 绑定入参
		var in I
		var bindErr error
		switch a.Binder {
		case BindJSON:
			bindErr = c.ShouldBindJSON(&in)
		case BindQuery:
			bindErr = c.ShouldBindQuery(&in)
		default: // BindNone: 不绑定
		}
		if bindErr != nil {
			c.JSON(http.StatusOK, resp.Error(400, bindErr.Error()))
			return
		}

		// 3) 执行（可选事务）
		run := func(tx *gorm.DB) (O, error) { return a.Handler(c, tx, &in) }
		var out O
		var err error
		if a.UseTx {
			err = db.WithContext(c).Transaction(func(tx *gorm.DB) error {
				o, e := run(tx)
				out = o
				return e
			})
		} else {
			out, err = run(db.WithContext(c))
		}

		// 4) 统一错误映射
		if err != nil {
			var ae *AErr
			if errors.As(err, &ae) {
				c.JSON(http.StatusOK, resp.Error(ae.Code, ae.Error()))
				return
			}
			c.JSON(http.StatusOK, resp.Error(500, err.Error()))
			return
		}
		c.JSON(http.StatusOK, resp.OK(out))
	}

	switch strings.ToUpper(a.Method) {
	case http.MethodGet:
		e.g.GET(a.Path, h)
	case http.MethodPut:
		e.g.PUT(a.Path, h)
	case http.MethodDelete:
		e.g.DELETE(a.Path, h)
	default: // 默认 POST
		e.g.POST(a.Path, h)
	}
}
