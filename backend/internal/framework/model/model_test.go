package model_test

import (
	"testing"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/model"
	"caveclass/internal/framework/testutil"
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

// 禁用状态是 0，存进去、读出来都必须还是 0。
func TestDisabledStatusIsStored(t *testing.T) {
	database := testutil.NewDB(t)
	testutil.MustExec(t, database,
		"INSERT INTO sys_user (username, password_hash, role, status, token_version) VALUES (?, ?, ?, ?, 0)",
		"disabled", "x", access.RoleStudent, model.UserStatusDisabled)
	var stored model.User
	if err := database.Raw("SELECT * FROM sys_user WHERE username = ?", "disabled").Scan(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.UserStatusDisabled {
		t.Errorf("status = %d, want %d", stored.Status, model.UserStatusDisabled)
	}
}

func TestUsernameIsUnique(t *testing.T) {
	database := testutil.NewDB(t)
	err := database.Exec("INSERT INTO sys_user (username, password_hash, role, status, token_version) VALUES (?, ?, ?, ?, 0)",
		"admin", "x", access.RoleStudent, model.UserStatusActive).Error
	if err == nil {
		t.Error("expected unique constraint error for duplicate username")
	}
}
