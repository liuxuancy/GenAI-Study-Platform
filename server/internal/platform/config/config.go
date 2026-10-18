// Package config 从环境变量读取运行配置。
package config

import (
	"fmt"
	"os"
	"time"
)

// 未设置 JWT_SECRET 时使用的开发密钥，prod 环境下拒绝启动。
const DevJWTSecret = "learnhub-dev-secret-do-not-use-in-prod"

type Config struct {
	Env                string        // APP_ENV：dev（默认）| prod
	HTTPAddr           string        // HTTP_ADDR
	DBPath             string        // DB_PATH：SQLite 文件路径
	JWTSecret          string        // JWT_SECRET
	JWTTTL             time.Duration // JWT_TTL：token 有效期
	PermissionCacheTTL time.Duration // PERMISSION_CACHE_TTL：权限缓存时长
}

// Load 从环境变量读取配置。
// 全局只初始化一次：启动时由 main.go 调用。
func Load() (Config, error) {
	cfg := Config{
		Env:       getConfigStr("APP_ENV", "dev"),
		HTTPAddr:  getConfigStr("HTTP_ADDR", ":8080"),
		DBPath:    getConfigStr("DB_PATH", "data/learnhub.db"),
		JWTSecret: getConfigStr("JWT_SECRET", DevJWTSecret),
	}
	if cfg.Env != "dev" && cfg.Env != "prod" {
		return Config{}, fmt.Errorf("APP_ENV: must be dev or prod, got %q", cfg.Env)
	}

	var err error
	if cfg.JWTTTL, err = getConfigDuration("JWT_TTL", 24*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.PermissionCacheTTL, err = getConfigDuration("PERMISSION_CACHE_TTL", 30*time.Second); err != nil {
		return Config{}, err
	}
	// 开发密钥是公开的，用它签的 token 谁都能伪造
	if cfg.UsesDevSecret() && !cfg.IsDev() {
		return Config{}, fmt.Errorf("JWT_SECRET must be set when APP_ENV=prod")
	}
	return cfg, nil
}

func (cfg Config) IsDev() bool { return cfg.Env == "dev" }

func (cfg Config) UsesDevSecret() bool { return cfg.JWTSecret == DevJWTSecret }

func getConfigStr(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getConfigDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s: invalid duration %q", key, value)
	}
	return duration, nil
}
