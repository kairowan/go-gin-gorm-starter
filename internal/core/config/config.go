package config

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type HTTP struct {
	Host            string
	Port            int
	ReadTimeoutSec  int
	WriteTimeoutSec int
	IdleTimeoutSec  int
}
type AdminHTTP struct {
	Host string
	Port int
}

type App struct {
	Name  string
	Env   string
	HTTP  HTTP
	Admin AdminHTTP
}

type Log struct {
	Level string
	JSON  bool
}

type JWT struct {
	Secret            string
	Issuer            string
	AccessTokenTTLMin int
}

type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type DB struct {
	Driver             string
	DSN                string
	Username           string
	Password           string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeMin int
	AutoMigrate        bool
	LogLevel           string
}

type Config struct {
	App   App
	Log   Log
	JWT   JWT
	DB    DB
	Redis Redis `mapstructure:"redis"`
}

func Load(path string) *Config {
	v := viper.New()
	if path == "" {
		path = os.Getenv("CONFIG_PATH")
		if path == "" {
			path = "./configs/config.local.yaml"
		}
	}
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("APP")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("read config: %v", err)
	}
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		log.Fatalf("unmarshal config: %v", err)
	}
	_ = time.Now()
	return &c
}
