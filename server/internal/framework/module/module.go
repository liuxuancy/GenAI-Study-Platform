// Module 接口：每个业务模块向框架声明自己的权限码、默认授权和数据表。路由统一写在 internal/router/routes.go。

package module

import (
	"context"

	"gorm.io/gorm"

	"learnhub/internal/framework/access"
)

// Module 一个业务模块。不需要的部分返回 nil 即可。
type Module interface {
	// Name 模块名，全局唯一，用于日志和错误提示。
	Name() string
	// Namespaces 本模块拥有的使用方，即权限码第一段，例如 course 模块拥有 teacher 和 student。
	Namespaces() []string
	// Permissions 本模块声明的具体权限码，第一段必须属于 Namespaces。
	Permissions() []access.PermissionDef
	// DefaultGrants 角色 → 默认授予的权限码，只能授予本模块的权限码。
	DefaultGrants() map[string][]string
	// Models 本模块需要建表的结构体。
	Models() []any
}

// DemoProvider 可选接口：模块实现它，就能在重置演示数据时写入自己的演示数据。
type DemoProvider interface {
	LoadDemo(ctx context.Context, database *gorm.DB) error
}
