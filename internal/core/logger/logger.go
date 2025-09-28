package logger

import (
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type FileRotate struct {
	Enable     bool   // 是否启用文件写入 + 切割
	Filename   string // 日志文件路径，如 logs/app.log
	MaxSizeMB  int    // 单个文件最大 MB
	MaxBackups int    // 保留旧文件个数
	MaxAgeDays int    // 保留天数
	Compress   bool   // 是否压缩旧日志
}

type Options struct {
	Level       string     // 日志级别：debug / info / warn / error
	JSON        bool       // 是否 JSON 格式输出
	AddCaller   bool       // 是否输出调用者文件行号
	Development bool       // 开发模式（影响编码器细节）
	Rotate      FileRotate // 文件切割配置（可选）
}

func New(level string, json bool) (*zap.Logger, func()) {
	l, closer := buildLogger(Options{
		Level:       level,
		JSON:        json,
		AddCaller:   true,
		Development: !json, // 控制台更适合开发格式
	})
	return l, closer
}

func NewWithRotate(level string, json bool, filename string, maxSizeMB, maxBackups, maxAgeDays int, compress bool) (*zap.Logger, func()) {
	l, closer := buildLogger(Options{
		Level:       level,
		JSON:        json,
		AddCaller:   true,
		Development: !json,
		Rotate: FileRotate{
			Enable:     true,
			Filename:   filename,
			MaxSizeMB:  maxSizeMB,
			MaxBackups: maxBackups,
			MaxAgeDays: maxAgeDays,
			Compress:   compress,
		},
	})
	return l, closer
}

func buildLogger(opt Options) (*zap.Logger, func()) {
	// 1) 日志级别
	var lvl zapcore.Level
	if err := lvl.Set(opt.Level); err != nil {
		lvl = zapcore.InfoLevel
	}

	var enc zapcore.Encoder
	if opt.JSON {
		cfg := zap.NewProductionEncoderConfig()
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.TimeKey = "ts"
		cfg.EncodeCaller = zapcore.ShortCallerEncoder
		enc = zapcore.NewJSONEncoder(cfg)
	} else {
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncodeCaller = zapcore.ShortCallerEncoder
		enc = zapcore.NewConsoleEncoder(cfg)
	}

	var sinks []zapcore.Core

	stdCore := zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), lvl)
	sinks = append(sinks, stdCore)

	if opt.Rotate.Enable {
		rotator := &lumberjack.Logger{
			Filename:   opt.Rotate.Filename,
			MaxSize:    max(1, opt.Rotate.MaxSizeMB),  // MB
			MaxBackups: max(0, opt.Rotate.MaxBackups), // 个数
			MaxAge:     max(0, opt.Rotate.MaxAgeDays), // 天
			Compress:   opt.Rotate.Compress,
		}
		fileWS := zapcore.AddSync(rotWriter{rotator})
		fileCore := zapcore.NewCore(enc, fileWS, lvl)
		sinks = append(sinks, fileCore)
	}

	core := zapcore.NewTee(sinks...)

	sampled := zapcore.NewSamplerWithOptions(core, time.Second, 100, 100)

	opts := []zap.Option{}
	if opt.AddCaller {
		opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(1))
	}
	if opt.Development {
		opts = append(opts, zap.Development())
	}
	l := zap.New(sampled, opts...)
	cleanup := func() { _ = l.Sync() }
	return l, cleanup
}

type rotWriter struct{ *lumberjack.Logger }

func (w rotWriter) Write(p []byte) (n int, err error) { return w.Logger.Write(p) }
func (w rotWriter) Sync() error                       { return nil }

// max 辅助
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Middleware(l *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next() // 先执行后续处理

		latency := time.Since(start)
		if raw != "" {
			path = path + "?" + raw
		}
		rid, _ := c.Get("X-Request-ID")

		fields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}
		if ridStr, ok := rid.(string); ok && ridStr != "" {
			fields = append(fields, zap.String("request_id", ridStr))
		}
		if len(c.Errors) > 0 {
			// 将 gin 的错误栈输出
			l.Error("HTTP", append(fields, zap.String("errors", c.Errors.String()))...)
		} else {
			l.Info("HTTP", fields...)
		}
	}
}

type zapIOWriter struct {
	l     *zap.Logger
	level zapcore.Level
}

func (w *zapIOWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\r\n")
	if ce := w.l.Check(w.level, msg); ce != nil {
		ce.Write()
	}
	return len(p), nil
}

func ToWriter(l *zap.Logger, level zapcore.Level) io.Writer {
	return &zapIOWriter{l: l, level: level}
}

func ToStdLogger(l *zap.Logger, level zapcore.Level) (*log.Logger, error) {
	return zap.NewStdLogAt(l, level)
}

func RedirectStdLog(l *zap.Logger, level zapcore.Level) func() {
	undo, _ := zap.RedirectStdLogAt(l, level)
	return func() { undo() }
}
