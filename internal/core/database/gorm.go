package database

import (
	"fmt"
	"log"
	_ "log"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
)

type Opts struct {
	Driver             string
	DSN                string
	Username           string
	Password           string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeMin int
	LogLevel           string
}

func NewGorm(o Opts) (*gorm.DB, error) {
	var dial gorm.Dialector
	switch o.Driver {
	case "postgres":
		dial = postgres.Open(o.DSN)
	case "mysql":
		dsn := normalizeMySQLDSN(o.DSN, o.Username, o.Password)
		masked := dsn
		if at := strings.Index(masked, "@"); at > 0 {
			if colon := strings.Index(masked[:at], ":"); colon > 0 {
				masked = masked[:colon+1] + "****" + masked[at:]
			}
		}
		log.Println("[db] final mysql dsn =", masked)

		dial = mysql.Open(dsn)
	default:
		return nil, ErrUnsupportedDriver
	}
	lvl := logger.Warn
	switch o.LogLevel {
	case "silent":
		lvl = logger.Silent
	case "error":
		lvl = logger.Error
	case "info":
		lvl = logger.Info
	}
	db, err := gorm.Open(dial, &gorm.Config{
		Logger: logger.Default.LogMode(lvl),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(o.MaxOpenConns)
	sqlDB.SetMaxIdleConns(o.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(o.ConnMaxLifetimeMin) * time.Minute)
	db = db.
		Session(&gorm.Session{
			PrepareStmt:            true, // 预编译缓存，提高 QPS
			CreateBatchSize:        200,  // 批量写
			SkipDefaultTransaction: true, // 只在需要时手动开 Tx
		})
	return db, nil
}
func normalizeMySQLDSN(input, userOverride, passOverride string) string {
	in := strings.TrimSpace(input)
	if in == "" {
		return in
	}

	// jdbc:mysql://... → mysql://...
	if strings.HasPrefix(in, "jdbc:mysql://") {
		in = strings.TrimPrefix(in, "jdbc:")
	}
	// 如果本身就是 go-sql-driver 的 DSN（user:pass@tcp(...)），不做改写
	if !strings.HasPrefix(in, "mysql://") {
		// 仍可在这里按需注入 user/pass，但容易误伤已有 DSN；保持原样更稳
		return in
	}

	u, err := url.Parse(in)
	if err != nil {
		return in // 解析失败则交给驱动报错
	}

	// 基础信息
	hostport := u.Host
	dbname := strings.TrimPrefix(u.Path, "/")

	// 用户名/密码：URL 中的（或 query 里的）→ 最后用 override 覆盖
	var user, pass string
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}
	q := u.Query()
	if q.Get("user") != "" {
		user = q.Get("user")
		q.Del("user")
	}
	if q.Get("password") != "" {
		pass = q.Get("password")
		q.Del("password")
	}
	if userOverride != "" {
		user = userOverride
	}
	if passOverride != "" {
		pass = passOverride
	}

	// Navicat/JDBC 常见参数适配
	// characterEncoding → charset（若未显式设置 charset）
	if q.Get("characterEncoding") != "" && q.Get("charset") == "" {
		q.Set("charset", q.Get("characterEncoding"))
	}
	q.Del("characterEncoding")

	// useUnicode 无需；删除避免噪音
	q.Del("useUnicode")

	// zeroDateTimeBehavior JDBC 专用；go-sql-driver 不支持，删除避免 DSN 不识别
	q.Del("zeroDateTimeBehavior")

	// useSSL → tls（go-sql-driver 的参数）
	if v := strings.ToLower(q.Get("useSSL")); v != "" {
		switch v {
		case "true", "1":
			q.Set("tls", "true") // 校验证书（需可信CA + 域名匹配）
		case "skip-verify":
			q.Set("tls", "skip-verify") // 跳过校验（开发期可用）
		case "preferred":
			q.Set("tls", "preferred") // 尝试TLS，不行则退回明文
		default: // "false" / 其它
			q.Set("tls", "false")
		}
		q.Del("useSSL")
	}

	// serverTimezone → loc
	if tz := q.Get("serverTimezone"); tz != "" {
		q.Set("loc", tz) // 传入的已经 URL 编码，如 GMT%2B8、Asia%2FShanghai
		q.Del("serverTimezone")
	}

	// 推荐默认项：parseTime/charset
	if q.Get("parseTime") == "" {
		q.Set("parseTime", "true")
	}
	if q.Get("charset") == "" {
		q.Set("charset", "utf8mb4")
	}

	// 拼成 go-sql-driver 语法：user:pass@tcp(host:port)/db?...
	cred := user
	if pass != "" {
		cred += ":" + pass
	}
	if cred != "" {
		cred += "@"
	}

	dsn := fmt.Sprintf("%stcp(%s)/%s", cred, hostport, dbname)
	if enc := q.Encode(); enc != "" {
		dsn += "?" + enc
	}
	return dsn
}

var ErrUnsupportedDriver = gorm.ErrInvalidDB
