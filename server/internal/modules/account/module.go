// account 模块的声明：登录、退出、me、个人信息、改密码，只要求登录，不声明权限码，也没有自己的表。

package account

import "caveclass/internal/framework/access"

// Module 实现 module.Module。
type Module struct{}

func (Module) GetName() string                        { return "account" }
func (Module) GetNamespaces() []string                { return nil }
func (Module) GetPermissions() []access.PermissionDef { return nil }
func (Module) GetDefaultGrants() map[string][]string  { return nil }
func (Module) GetModels() []any                       { return nil }
