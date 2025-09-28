package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "go.uber.org/automaxprocs"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"go-gin-gorm-starter/internal/core/auth"
	"go-gin-gorm-starter/internal/core/config"
	"go-gin-gorm-starter/internal/core/database"
	"go-gin-gorm-starter/internal/core/logger"
	"go-gin-gorm-starter/internal/core/server"
	"go-gin-gorm-starter/internal/repo"
	"go-gin-gorm-starter/internal/service"
	"go-gin-gorm-starter/internal/transport/http/handler"
	"go-gin-gorm-starter/internal/transport/http/router"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load(os.Getenv("CONFIG_PATH"))
	log, cleanup := logger.New(cfg.Log.Level, cfg.Log.JSON)
	defer cleanup()

	// DB 连接（失败直接 Fatal）
	db := mustOpenDB(cfg, log)
	log.Info("database connected", zap.String("driver", cfg.DB.Driver))

	// 依赖
	jwter := &auth.JWTer{
		Secret: []byte(cfg.JWT.Secret),
		Issuer: cfg.JWT.Issuer,
		TTL:    time.Duration(cfg.JWT.AccessTokenTTLMin) * time.Minute,
	}
	userRepo := repo.NewUserRepo(db)
	userSvc := service.NewUserService(userRepo)
	adminH := handler.NewAdminHandler(userSvc)

	// 路由（后台端）
	r := router.NewAdminEngine(log, adminH, jwter)

	// HTTP Server
	addr := server.Addr(cfg.App.Admin.Host, cfg.App.Admin.Port)
	srv := server.BuildServer(addr, r, 5*time.Second, 10*time.Second, 60*time.Second)

	// 启动前打印可点击地址
	host4human := cfg.App.Admin.Host
	if host4human == "" || host4human == "0.0.0.0" {
		host4human = "127.0.0.1"
	}
	baseURL := "http://" + host4human + ":" + fmt.Sprint(cfg.App.Admin.Port)
	log.Info("admin api starting",
		zap.String("addr", addr),
		zap.String("open", baseURL),
		zap.String("health", baseURL+"/health"),
		zap.String("admin_v1", baseURL+"/admin/v1"),
	)

	// 异步启动；失败立即标红退出
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("admin api start FAILED", zap.Error(err))
		}
	}()
	log.Info("admin api started SUCCESS")

	// 关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("admin api stopped gracefully")
}

type ginH map[string]any

func mustOpenDB(cfg *config.Config, l *zap.Logger) *gorm.DB {
	db, err := database.NewGorm(database.Opts{
		Driver:             cfg.DB.Driver,
		DSN:                cfg.DB.DSN,
		Username:           cfg.DB.Username, // 传入用户名
		Password:           cfg.DB.Password, // 传入密码
		MaxOpenConns:       cfg.DB.MaxOpenConns,
		MaxIdleConns:       cfg.DB.MaxIdleConns,
		ConnMaxLifetimeMin: cfg.DB.ConnMaxLifetimeMin,
		LogLevel:           cfg.DB.LogLevel,
	})
	if err != nil {
		l.Fatal("db open", zap.Error(err)) // 失败日志
	}
	return db
}
