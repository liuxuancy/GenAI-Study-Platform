// Package testutil 测试脚手架：建一个临时 SQLite，建表并写入种子数据，供各包的测试使用。
package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"learnhub/internal/framework/identity"
	"learnhub/internal/framework/model"
	"learnhub/internal/framework/module"
	"learnhub/internal/framework/seed"
	"learnhub/internal/platform/db"
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
	models := append(model.FrameworkModels(), registry.Models()...)
	if err := db.Migrate(database, models...); err != nil {
		t.Fatal(err)
	}
	if err := seed.Run(context.Background(), database, registry.DefaultGrants()); err != nil {
		t.Fatal(err)
	}
	return database
}
