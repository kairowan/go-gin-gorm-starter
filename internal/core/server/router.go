package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Options struct {
	Name string
	Mode string
}

func NewRouter(l *zap.Logger) *gin.Engine {
	r := gin.New()
	r.Use(ginzap.Ginzap(l, time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(l, true))
	r.Use(cors.Default())
	// 也可接入你自定义的 RequestID/Timeout 中间件
	return r
}

func StartHTTP(srv *http.Server, l *zap.Logger) error {
	l.Info("http starting", zap.String("addr", srv.Addr))
	return srv.ListenAndServe()
}

func BuildServer(addr string, handler http.Handler, rt, wt, it time.Duration) *http.Server {
	return &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    rt,
		WriteTimeout:   wt,
		IdleTimeout:    it,
		MaxHeaderBytes: 1 << 20, // 1MB
	}
}

func Addr(host string, port int) string { return fmt.Sprintf("%s:%d", host, port) }
