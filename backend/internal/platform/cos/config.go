// Package cos 封装腾讯云 COS（课程资料私有存储，密钥仅服务端环境变量）。
package cos

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBucket = "caveclass-resources-1385365202"
	defaultRegion = "ap-guangzhou"
)

// Config 从 COS_* 环境变量读取；Secret 为空时 IsConfigured 为 false。
type Config struct {
	Bucket    string
	Region    string
	SecretID  string
	SecretKey string
}

// LoadConfig 读取 COS_BUCKET、COS_REGION、COS_SECRET_ID、COS_SECRET_KEY。
func LoadConfig() Config {
	return Config{
		Bucket:    getenv("COS_BUCKET", defaultBucket),
		Region:    getenv("COS_REGION", defaultRegion),
		SecretID:  strings.TrimSpace(os.Getenv("COS_SECRET_ID")),
		SecretKey: strings.TrimSpace(os.Getenv("COS_SECRET_KEY")),
	}
}

// IsConfigured 是否具备调用 COS API 的最小凭证（不校验 Bucket 是否存在）。
func (c Config) IsConfigured() bool {
	return c.Bucket != "" && c.Region != "" && c.SecretID != "" && c.SecretKey != ""
}

// Validate 返回可读的配置错误（不含密钥内容）。
func (c Config) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("COS_BUCKET is required")
	}
	if c.Region == "" {
		return fmt.Errorf("COS_REGION is required")
	}
	if c.SecretID == "" {
		return fmt.Errorf("COS_SECRET_ID is required")
	}
	if c.SecretKey == "" {
		return fmt.Errorf("COS_SECRET_KEY is required")
	}
	return nil
}

// RedactedSummary 供日志/CLI 输出，不含 Secret。
func (c Config) RedactedSummary() string {
	return fmt.Sprintf("bucket=%q region=%q secret_id_set=%t secret_key_set=%t",
		c.Bucket, c.Region, c.SecretID != "", c.SecretKey != "")
}

func getenv(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

// Limits 上传大小与预签名 TTL（环境变量可覆盖）。
type Limits struct {
	DocMaxBytes         int64
	ArchiveMaxBytes     int64
	PresignTTL          time.Duration
	DocPreviewEnabled   bool // COS 数据万象 doc-preview；Bucket 需开通且 COS_DOC_PREVIEW_ENABLED=true
}

const (
	defaultDocMaxBytes     = 50 * 1024 * 1024
	defaultArchiveMaxBytes = 500 * 1024 * 1024
	defaultPresignSec      = 900
)

// LoadLimits 读取 COS_DOC_MAX_BYTES、COS_ARCHIVE_MAX_BYTES、COS_PRESIGN_TTL_SEC。
func LoadLimits() Limits {
	return Limits{
		DocMaxBytes:       getenvInt64("COS_DOC_MAX_BYTES", defaultDocMaxBytes),
		ArchiveMaxBytes:   getenvInt64("COS_ARCHIVE_MAX_BYTES", defaultArchiveMaxBytes),
		PresignTTL:        time.Duration(getenvInt("COS_PRESIGN_TTL_SEC", defaultPresignSec)) * time.Second,
		DocPreviewEnabled: getenvBool("COS_DOC_PREVIEW_ENABLED", false),
	}
}

func getenvBool(key string, defaultValue bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return defaultValue
	}
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

func getenvInt(key string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultValue
	}
	return n
}

func getenvInt64(key string, defaultValue int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return defaultValue
	}
	return n
}
