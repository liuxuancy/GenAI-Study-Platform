// Package db 负责打开 SQLite、建表和删除本地数据库文件。
package db

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 每个连接都会应用的参数
const dsnParams = "" +
	"?_pragma=journal_mode(WAL)" + // 读写互不阻塞
	"&_pragma=busy_timeout(5000)" + // 拿不到写锁时最多等 5 秒，而不是直接报 database is locked
	"&_txlock=immediate" // 事务开始即拿写锁；默认的 deferred 在"先读后写"时会跳过 busy_timeout 直接报错

// Open 打开 path 处的 SQLite 文件，不存在则创建。
// 全局只初始化一次：启动时由 main.go 调用，返回的连接池由所有请求共用。
func Open(path string) (*gorm.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	database, err := gorm.Open(sqlite.Open(path+dsnParams), &gorm.Config{
		NowFunc: func() time.Time { return time.Now().UTC() }, // 时间统一存 UTC
		Logger:  logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	return database, nil
}

// Close 关闭底层连接池。
func Close(database *gorm.DB) error {
	sqlDB, err := database.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Migrate 根据结构体自动建表、加字段。不会删字段或改字段名。
func Migrate(database *gorm.DB, models ...any) error {
	if err := database.AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

// RemoveFiles 删除数据库文件以及 WAL 模式附带的 -wal、-shm 文件，文件不存在不算错误。
func RemoveFiles(path string) error {
	for _, file := range []string{path, path + "-wal", path + "-shm"} {
		if err := os.Remove(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", file, err)
		}
	}
	return nil
}
