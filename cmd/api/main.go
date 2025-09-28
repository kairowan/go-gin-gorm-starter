package main

import (
	"context"
	"errors"
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
	"go-gin-gorm-starter/internal/feature/user"
	"go-gin-gorm-starter/internal/transport/http/router"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load(os.Getenv("CONFIG_PATH"))
	log, cleanup := logger.New(cfg.Log.Level, cfg.Log.JSON)
	defer cleanup()

	// 数据库（失败会直接 Fatal）
	db := mustOpenDB(cfg, log)
	log.Info("database connected", zap.String("driver", cfg.DB.Driver))

	// 自动迁移（使用新模型）
	if cfg.DB.AutoMigrate {
		if err := db.AutoMigrate(&user.UserModel{}); err != nil {
			log.Fatal("automigrate failed", zap.Error(err))
		}
		log.Info("automigrate done")
	}

	// JWT
	jwter := &auth.JWTer{
		Secret: []byte(cfg.JWT.Secret),
		Issuer: cfg.JWT.Issuer,
		TTL:    time.Duration(cfg.JWT.AccessTokenTTLMin) * time.Minute,
	}

	// 路由（用户端）
	r := router.NewAPIEngine(log, db, jwter)

	// HTTP Server
	addr := server.Addr(cfg.App.HTTP.Host, cfg.App.HTTP.Port)
	srv := server.BuildServer(
		addr, r,
		time.Duration(cfg.App.HTTP.ReadTimeoutSec)*time.Second,
		time.Duration(cfg.App.HTTP.WriteTimeoutSec)*time.Second,
		time.Duration(cfg.App.HTTP.IdleTimeoutSec)*time.Second,
	)

	// 启动日志
	host4human := cfg.App.HTTP.Host
	if host4human == "" || host4human == "0.0.0.0" {
		host4human = "127.0.0.1"
	}
	baseURL := "http://" + host4human + ":" + fmt.Sprint(cfg.App.HTTP.Port)
	log.Info("user api starting",
		zap.String("addr", addr),
		zap.String("open", baseURL),
		zap.String("health", baseURL+"/health"),
		zap.String("api_v1", baseURL+"/api/v1"),
	)

	// 异步启动
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("user api start FAILED", zap.Error(err))
		}
	}()
	log.Info("user api started SUCCESS")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("user api stopped gracefully")
}

func mustOpenDB(cfg *config.Config, l *zap.Logger) *gorm.DB {
	db, err := database.NewGorm(database.Opts{
		Driver:             cfg.DB.Driver,
		DSN:                cfg.DB.DSN,
		Username:           cfg.DB.Username,
		Password:           cfg.DB.Password,
		MaxOpenConns:       cfg.DB.MaxOpenConns,
		MaxIdleConns:       cfg.DB.MaxIdleConns,
		ConnMaxLifetimeMin: cfg.DB.ConnMaxLifetimeMin,
		LogLevel:           cfg.DB.LogLevel,
	})
	if err != nil {
		l.Fatal("db open", zap.Error(err))
	}
	return db
}
