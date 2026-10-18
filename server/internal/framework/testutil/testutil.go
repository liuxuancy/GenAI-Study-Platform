// Package testutil 测试脚手架：建一个临时 SQLite，建表并写入种子数据，供各包的测试使用。
package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"caveclass/internal/framework/identity"
	"caveclass/internal/framework/model"
	"caveclass/internal/framework/module"
	"caveclass/internal/framework/seed"
	"caveclass/internal/platform/db"
)

// NewDB 返回一个已建表、已写入测试账号和 modules 默认授权的数据库，测试结束后自动删除。
func NewDB(t *testing.T, modules ...module.Module) *gorm.DB {
	t.Helper()
	identity.UseFastHashForTests()

	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(database) })

	registry, err := module.NewRegistry(modules...)
	if err != nil {
		t.Fatal(err)
	}
	models := append(model.GetFrameworkModels(), registry.GetModels()...)
	if err := db.Migrate(database, models...); err != nil {
		t.Fatal(err)
	}
	if err := seed.Run(context.Background(), database, registry.GetDefaultGrants()); err != nil {
		t.Fatal(err)
	}
	return database
}

// MustExec 执行一条写入 SQL（INSERT / UPDATE / DELETE），出错直接让测试失败，返回影响的行数。
func MustExec(t *testing.T, database *gorm.DB, sql string, args ...any) int64 {
	t.Helper()
	result := database.Exec(sql, args...)
	if result.Error != nil {
		t.Fatalf("exec %q: %v", sql, result.Error)
	}
	return result.RowsAffected
}

// GetUserID 按用户名查出用户 ID，查不到直接让测试失败。
func GetUserID(t *testing.T, database *gorm.DB, username string) int64 {
	t.Helper()
	var userID int64
	result := database.Raw("SELECT id FROM sys_user WHERE username = ?", username).Scan(&userID)
	if result.Error != nil || result.RowsAffected == 0 {
		t.Fatalf("user %s not found: %v", username, result.Error)
	}
	return userID
}
