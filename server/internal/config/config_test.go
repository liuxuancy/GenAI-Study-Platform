package config

import (
	"testing"
	"time"
)

var allKeys = []string{"APP_ENV", "HTTP_ADDR", "DB_PATH", "JWT_SECRET", "JWT_TTL", "PERMISSION_CACHE_TTL"}

// setEnv 先清空所有配置项，避免本机环境变量干扰，再设置 values 中的值。
func setEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for _, key := range allKeys {
		t.Setenv(key, values[key])
	}
}

func TestLoadDefaults(t *testing.T) {
	setEnv(t, nil)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "dev" || cfg.HTTPAddr != ":8080" || cfg.DBPath != "data/learnhub.db" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if cfg.JWTTTL != 24*time.Hour || cfg.PermissionCacheTTL != 30*time.Second {
		t.Errorf("unexpected default durations: %+v", cfg)
	}
	if !cfg.UsesDevSecret() {
		t.Error("expected dev secret when JWT_SECRET is unset")
	}
}

func TestLoadOverrides(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":              "prod",
		"HTTP_ADDR":            ":9000",
		"DB_PATH":              "/tmp/x.db",
		"JWT_SECRET":           "s3cret",
		"JWT_TTL":              "2h",
		"PERMISSION_CACHE_TTL": "5s",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IsDev() || cfg.HTTPAddr != ":9000" || cfg.DBPath != "/tmp/x.db" || cfg.UsesDevSecret() {
		t.Errorf("overrides not applied: %+v", cfg)
	}
	if cfg.JWTTTL != 2*time.Hour || cfg.PermissionCacheTTL != 5*time.Second {
		t.Errorf("durations not applied: %+v", cfg)
	}
}

func TestLoadRejectsBadInput(t *testing.T) {
	cases := map[string]map[string]string{
		"bad duration":         {"JWT_TTL": "tomorrow"},
		"negative duration":    {"PERMISSION_CACHE_TTL": "-1s"},
		"unknown env":          {"APP_ENV": "staging"},
		"prod with dev secret": {"APP_ENV": "prod"},
	}
	for name, values := range cases {
		t.Run(name, func(t *testing.T) {
			setEnv(t, values)
			if _, err := Load(); err == nil {
				t.Error("expected error")
			}
		})
	}
}
