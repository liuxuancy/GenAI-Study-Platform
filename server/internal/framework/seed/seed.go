// 种子数据：测试账号，以及授权表为空时写入的各模块默认授权。可重复执行，不覆盖已有数据。

package seed

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"gorm.io/gorm"

	"learnhub/internal/framework/access"
	"learnhub/internal/framework/identity"
	"learnhub/internal/framework/model"
	"learnhub/internal/platform/logs"
)

// Account 一个要写入的账号。演示数据的 users.json 也是这个格式。
type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// 测试账号，密码只用于本地开发
var testAccounts = []Account{
	{Username: "admin", Password: "admin123", Nickname: "管理员", Role: access.RoleAdmin},
	{Username: "teacher1", Password: "teacher123", Nickname: "讲师一号", Role: access.RoleTeacher},
	{Username: "teacher2", Password: "teacher123", Nickname: "讲师二号", Role: access.RoleTeacher},
	{Username: "student1", Password: "student123", Nickname: "学生一号", Role: access.RoleStudent},
}

// TestAccounts 返回测试账号（副本），测试里用来登录。
func TestAccounts() []Account {
	return slices.Clone(testAccounts)
}

// Run 写入种子数据，整个过程在一个事务里。
func Run(ctx context.Context, database *gorm.DB, defaultGrants map[string][]string) error {
	seeder := &seeder{ctx: ctx, defaultGrants: defaultGrants}
	return database.WithContext(ctx).Transaction(seeder.run)
}

type seeder struct {
	ctx           context.Context
	defaultGrants map[string][]string
}

func (seeder *seeder) run(tx *gorm.DB) error {
	if err := ensureAccounts(seeder.ctx, tx, "test accounts", testAccounts); err != nil {
		return err
	}
	return ensureDefaultGrants(seeder.ctx, tx, seeder.defaultGrants)
}

// ensureAccounts 逐个创建不存在的账号。已存在的账号保持原样，不会重置密码。
func ensureAccounts(ctx context.Context, tx *gorm.DB, label string, accounts []Account) error {
	created := 0
	for _, account := range accounts {
		isNew, err := ensureAccount(tx, account)
		if err != nil {
			return err
		}
		if isNew {
			created++
		}
	}
	logs.CtxInfo(ctx, "%s: %d created, %d already existed", label, created, len(accounts)-created)
	return nil
}

func ensureAccount(tx *gorm.DB, account Account) (bool, error) {
	username := strings.ToLower(account.Username)
	if !access.IsValidRole(account.Role) {
		return false, fmt.Errorf("account %s: unknown role %q", username, account.Role)
	}

	var count int64
	if err := tx.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}

	hash, err := identity.HashPassword(account.Password)
	if err != nil {
		return false, fmt.Errorf("account %s: %w", username, err)
	}
	user := model.User{
		Username:     username,
		PasswordHash: hash,
		Nickname:     account.Nickname,
		Role:         account.Role,
		Status:       model.UserStatusActive,
	}
	return true, tx.Create(&user).Error
}

// ensureDefaultGrants 只在授权表为空（首次启动）时写入默认授权，
// 之后管理员的调整都保留，重启不会被覆盖。
func ensureDefaultGrants(ctx context.Context, tx *gorm.DB, defaultGrants map[string][]string) error {
	var count int64
	if err := tx.Model(&model.RolePermission{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		logs.CtxInfo(ctx, "seed grants: skipped, %d grants already exist", count)
		return nil
	}

	var rows []model.RolePermission
	for _, role := range slices.Sorted(maps.Keys(defaultGrants)) {
		for _, code := range defaultGrants[role] {
			rows = append(rows, model.RolePermission{Role: role, PermissionCode: code})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	if err := tx.Create(&rows).Error; err != nil {
		return err
	}
	logs.CtxInfo(ctx, "seed grants: %d default grants written", len(rows))
	return nil
}
