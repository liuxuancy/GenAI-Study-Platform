package model_test

import (
	"testing"

	"learnhub/internal/framework/access"
	"learnhub/internal/framework/model"
	"learnhub/internal/framework/testutil"
)

// TableName 被误改名时，GORM 会悄悄改用默认表名 users、role_permissions，这里能及时发现。
func TestFrameworkTableNames(t *testing.T) {
	database := testutil.NewDB(t)
	for _, table := range []string{"sys_user", "sys_role_permission"} {
		if !database.Migrator().HasTable(table) {
			t.Errorf("table %s not created", table)
		}
	}
}

// 禁用状态是 0，必须能原样存进去，而不是被当成"未赋值"换成别的值。
func TestDisabledStatusIsStored(t *testing.T) {
	database := testutil.NewDB(t)
	user := model.User{Username: "disabled", PasswordHash: "x", Role: access.RoleStudent, Status: model.UserStatusDisabled}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	var stored model.User
	if err := database.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.UserStatusDisabled {
		t.Errorf("status = %d, want %d", stored.Status, model.UserStatusDisabled)
	}
}

func TestUsernameIsUnique(t *testing.T) {
	database := testutil.NewDB(t)
	duplicate := model.User{Username: "admin", PasswordHash: "x", Role: access.RoleStudent, Status: model.UserStatusActive}
	if err := database.Create(&duplicate).Error; err == nil {
		t.Error("expected unique constraint error for duplicate username")
	}
}
