package router

import (
	"sort"
	"sync"

	"github.com/gin-gonic/gin"
)

// APIModule 模块可选择实现其中一个或两个接口
type APIModule interface{ MountAPI(*gin.RouterGroup) }
type AdminModule interface{ MountAdmin(*gin.RouterGroup) }

// 可选：实现该接口可控制挂载顺序（数值越小越先挂）
// 不实现则默认 100
type prioritizer interface{ Priority() int }

var (
	mu        sync.RWMutex
	apiMods   []APIModule
	adminMods []AdminModule
)

// Register 统一注册入口：根据类型断言分发到 API/Admin 列表
func Register(mod any) {
	mu.Lock()
	defer mu.Unlock()
	if m, ok := mod.(APIModule); ok {
		apiMods = append(apiMods, m)
	}
	if m, ok := mod.(AdminModule); ok {
		adminMods = append(adminMods, m)
	}
}

// MountAllAPI 在 /api/v1 上挂载所有已注册的 API 模块
func MountAllAPI(api *gin.RouterGroup) {
	mu.RLock()
	mods := append([]APIModule(nil), apiMods...)
	mu.RUnlock()

	sort.SliceStable(mods, func(i, j int) bool {
		return priorityOf(mods[i]) < priorityOf(mods[j])
	})
	for _, m := range mods {
		m.MountAPI(api)
	}
}

// MountAllAdmin 在 /admin/v1 上挂载所有已注册的 Admin 模块
func MountAllAdmin(admin *gin.RouterGroup) {
	mu.RLock()
	mods := append([]AdminModule(nil), adminMods...)
	mu.RUnlock()

	sort.SliceStable(mods, func(i, j int) bool {
		return priorityOf(mods[i]) < priorityOf(mods[j])
	})
	for _, m := range mods {
		m.MountAdmin(admin)
	}
}

func priorityOf(v any) int {
	if p, ok := v.(prioritizer); ok {
		return p.Priority()
	}
	return 100
}
