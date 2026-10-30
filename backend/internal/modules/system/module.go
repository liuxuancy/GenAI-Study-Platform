// system 模块的声明：用户管理、角色权限配置的权限码。只有管理员使用，不需要默认授权，也没有自己的表。

package system

import "caveclass/internal/framework/access"

// 权限码常量：模块声明和路由表都引用它们，避免两边拼写不一致。
const (
	PermissionUserView          = "system:user:view"
	PermissionUserCreate        = "system:user:create"
	PermissionUserUpdate        = "system:user:update"
	PermissionUserDelete        = "system:user:delete"
	PermissionUserResetPassword = "system:user:reset-password"
	PermissionRoleView          = "system:role:view"
	PermissionRoleAssign        = "system:role:assign-permission"
	PermissionCatalogList       = "system:permission:list"
	PermissionClassView         = "system:class:view"
	PermissionClassCreate       = "system:class:create"
	PermissionClassUpdate       = "system:class:update"
	PermissionCourseView        = "system:course:view"
	PermissionCourseUpdate      = "system:course:update"
	PermissionUsageView         = "system:usage:view"
)

// Module 实现 module.Module。
type Module struct{}

func (Module) GetName() string                       { return "system" }
func (Module) GetNamespaces() []string               { return []string{"system"} }
func (Module) GetDefaultGrants() map[string][]string { return nil } // 管理员在代码里直接放行
func (Module) GetModels() []any                      { return nil }

func (Module) GetPermissions() []access.PermissionDef {
	return []access.PermissionDef{
		{Code: PermissionUserView, Name: "用户管理", Type: access.PermissionTypeMenu, Description: "用户管理页面和用户列表"},
		{Code: PermissionUserCreate, Name: "新建用户", Type: access.PermissionTypeButton},
		{Code: PermissionUserUpdate, Name: "编辑用户", Type: access.PermissionTypeButton, Description: "修改昵称、邮箱，启用或禁用"},
		{Code: PermissionUserDelete, Name: "删除用户", Type: access.PermissionTypeButton},
		{Code: PermissionUserResetPassword, Name: "重置密码", Type: access.PermissionTypeButton},
		{Code: PermissionRoleView, Name: "角色权限", Type: access.PermissionTypeMenu, Description: "角色权限页面"},
		{Code: PermissionRoleAssign, Name: "配置角色权限", Type: access.PermissionTypeButton},
		{Code: PermissionCatalogList, Name: "权限码目录", Type: access.PermissionTypeAPI, Description: "角色权限页的勾选树数据"},
		{Code: PermissionClassView, Name: "班级管理", Type: access.PermissionTypeMenu, Description: "班级列表"},
		{Code: PermissionClassCreate, Name: "新建班级", Type: access.PermissionTypeButton},
		{Code: PermissionClassUpdate, Name: "编辑班级", Type: access.PermissionTypeButton, Description: "改班名、分配教师和学生"},
		{Code: PermissionCourseView, Name: "课程管理", Type: access.PermissionTypeMenu, Description: "全部课程和发布状态"},
		{Code: PermissionCourseUpdate, Name: "下架课程", Type: access.PermissionTypeButton, Description: "对学生隐藏或恢复，不删除课程"},
		{Code: PermissionUsageView, Name: "模型用量", Type: access.PermissionTypeMenu, Description: "账号数、全站 token，以及每位师生的调用明细"},
	}
}
