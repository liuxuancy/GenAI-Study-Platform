// Package model 框架层的两张表：sys_user、sys_role_permission，两表通过role连接。业务表由各模块自己定义。
package model

import "time"

const (
	UserStatusDisabled = 0
	UserStatusActive   = 1
)

// User 对应 sys_user。
// Status 不加 gorm default 标签：否则写入 0（禁用）时会被当成"未赋值"而存成默认值。
type User struct {
	ID           int64  `gorm:"primaryKey"`
	Username     string `gorm:"size:64;not null;uniqueIndex"` // 写入前统一转成小写
	PasswordHash string `gorm:"size:100;not null" json:"-"`
	Nickname     string `gorm:"size:64"`
	Email        string `gorm:"size:128"`
	Role         string `gorm:"size:32;not null;index"` // 创建后不可修改
	Status       int    `gorm:"not null"`
	TokenVersion int    `gorm:"not null"` // 禁用、改密码、重置密码时加一，旧 token 立即失效
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TableName 方法名是 GORM 规定的，不能改成GetTableName
func (User) TableName() string { return "sys_user" }

// RolePermission 对应sys_role_permission，各角色权限配置，所有用户使用系统配置的角色，不针对特定用户单独修改权限
type RolePermission struct {
	Role           string `gorm:"primaryKey;size:32"`
	PermissionCode string `gorm:"primaryKey;size:128;index"`
}

// TableName 方法名是 GORM 规定的，不能改成GetTableName
func (RolePermission) TableName() string { return "sys_role_permission" }

// GetFrameworkModels 返回框架层需要建表的结构体。
func GetFrameworkModels() []any {
	return []any{&User{}, &RolePermission{}}
}
