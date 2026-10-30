package seed_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"gorm.io/gorm"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/identity"
	"caveclass/internal/framework/model"
	"caveclass/internal/framework/module"
	"caveclass/internal/framework/seed"
	"caveclass/internal/framework/testutil"
)

// courseModule 带默认授权和演示数据的假模块。
type courseModule struct {
	demoCalls int
	demoErr   error
}

func (course *courseModule) GetName() string         { return "course" }
func (course *courseModule) GetNamespaces() []string { return []string{"teacher", "student"} }
func (course *courseModule) GetPermissions() []access.PermissionDef {
	return []access.PermissionDef{
		{Code: "teacher:course:view", Name: "查看课程", Type: access.PermissionTypeMenu},
		{Code: "student:catalog:view", Name: "课程广场", Type: access.PermissionTypeMenu},
		{Code: "student:my:view", Name: "我的学习", Type: access.PermissionTypeMenu},
	}
}
func (course *courseModule) GetDefaultGrants() map[string][]string {
	return map[string][]string{
		access.RoleTeacher: {"teacher:*"},
		access.RoleStudent: {"student:catalog:view", "student:my:view"},
	}
}
func (course *courseModule) GetModels() []any { return nil }
func (course *courseModule) LoadDemo(ctx context.Context, database *gorm.DB) error {
	course.demoCalls++
	return course.demoErr
}

var (
	_ module.Module       = (*courseModule)(nil)
	_ module.DemoProvider = (*courseModule)(nil)
)

func runSeed(t *testing.T, database *gorm.DB, course *courseModule) {
	t.Helper()
	if err := seed.Run(context.Background(), database, course.GetDefaultGrants()); err != nil {
		t.Fatal(err)
	}
}

func getUserByUsername(t *testing.T, database *gorm.DB, username string) model.User {
	t.Helper()
	var user model.User
	result := database.Raw("SELECT * FROM sys_user WHERE username = ?", username).Scan(&user)
	if result.Error != nil || result.RowsAffected == 0 {
		t.Fatalf("user %s not found: %v", username, result.Error)
	}
	return user
}

// getRowCount 统计表的行数，table 只传代码里写死的表名。
func getRowCount(t *testing.T, database *gorm.DB, table string) int64 {
	t.Helper()
	var count int64
	if err := database.Raw("SELECT COUNT(*) FROM " + table).Scan(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func getGrantsOfRole(t *testing.T, database *gorm.DB, role string) []string {
	t.Helper()
	var codes []string
	err := database.Raw("SELECT permission_code FROM sys_role_permission WHERE role = ? ORDER BY permission_code", role).
		Scan(&codes).Error
	if err != nil {
		t.Fatal(err)
	}
	return codes
}

func TestRunCreatesAccountsAndGrants(t *testing.T) {
	database := testutil.NewDB(t, &courseModule{})

	if count := getRowCount(t, database, "sys_user"); count != 4 {
		t.Errorf("users = %d, want 4", count)
	}
	for _, account := range seed.GetTestAccounts() {
		user := getUserByUsername(t, database, account.Username)
		if user.Role != account.Role || user.Status != model.UserStatusActive || user.TokenVersion != 0 {
			t.Errorf("%s: role=%s status=%d tokenVersion=%d", account.Username, user.Role, user.Status, user.TokenVersion)
		}
		if !identity.CheckPassword(user.PasswordHash, account.Password) {
			t.Errorf("%s: stored hash does not match password", account.Username)
		}
		wantMajor := ""
		if account.Username == "student1" {
			wantMajor = "Computer Science"
		}
		if user.Major != wantMajor {
			t.Errorf("%s major = %q, want %q", account.Username, user.Major, wantMajor)
		}
	}

	if got := getGrantsOfRole(t, database, access.RoleTeacher); !slices.Equal(got, []string{"teacher:*"}) {
		t.Errorf("teacher grants = %v", got)
	}
	if got := getGrantsOfRole(t, database, access.RoleStudent); !slices.Equal(got, []string{"student:catalog:view", "student:my:view"}) {
		t.Errorf("student grants = %v", got)
	}
	if got := getGrantsOfRole(t, database, access.RoleAdmin); len(got) != 0 {
		t.Errorf("admin must have no grant rows, got %v", got)
	}
}

func TestRunIsRepeatable(t *testing.T) {
	course := &courseModule{}
	database := testutil.NewDB(t, course)
	runSeed(t, database, course)
	runSeed(t, database, course)

	if count := getRowCount(t, database, "sys_user"); count != 4 {
		t.Errorf("users = %d after repeated seed, want 4", count)
	}
	if count := getRowCount(t, database, "sys_role_permission"); count != 3 {
		t.Errorf("grants = %d after repeated seed, want 3", count)
	}
}

// 管理员撤销过的授权，重启（再次执行种子）后不能被恢复。
func TestRunKeepsAdminChangesToGrants(t *testing.T) {
	course := &courseModule{}
	database := testutil.NewDB(t, course)
	revoked := testutil.MustExec(t, database, "DELETE FROM sys_role_permission WHERE role = ? AND permission_code = ?",
		access.RoleStudent, "student:my:view")
	if revoked != 1 {
		t.Fatalf("revoke: rows=%d, want 1", revoked)
	}

	runSeed(t, database, course)
	if got := getGrantsOfRole(t, database, access.RoleStudent); !slices.Equal(got, []string{"student:catalog:view"}) {
		t.Errorf("revoked grant came back: %v", got)
	}
}

func TestRunFillsEmptyMajorWithoutOverwrite(t *testing.T) {
	course := &courseModule{}
	database := testutil.NewDB(t, course)
	testutil.MustExec(t, database, "UPDATE sys_user SET major = ? WHERE username = ?", "Mathematics", "student1")

	runSeed(t, database, course)
	if got := getUserByUsername(t, database, "student1").Major; got != "Mathematics" {
		t.Errorf("major = %q, want the existing value", got)
	}
	if !identity.CheckPassword(getUserByUsername(t, database, "student1").PasswordHash, "student123") {
		t.Error("filling major reset the password")
	}

	testutil.MustExec(t, database, "UPDATE sys_user SET major = '' WHERE username = ?", "student1")
	runSeed(t, database, course)
	if got := getUserByUsername(t, database, "student1").Major; got != "Computer Science" {
		t.Errorf("empty major = %q, want Computer Science", got)
	}
}

func TestRunDoesNotResetExistingPassword(t *testing.T) {
	course := &courseModule{}
	database := testutil.NewDB(t, course)
	newHash, err := identity.HashPassword("changed-by-admin")
	if err != nil {
		t.Fatal(err)
	}
	testutil.MustExec(t, database, "UPDATE sys_user SET password_hash = ? WHERE username = ?", newHash, "admin")

	runSeed(t, database, course)
	if !identity.CheckPassword(getUserByUsername(t, database, "admin").PasswordHash, "changed-by-admin") {
		t.Error("seed reset an existing user's password")
	}
}

func TestLoadDemo(t *testing.T) {
	course := &courseModule{}
	database := testutil.NewDB(t, course)

	for range 2 { // 重复加载不报错、不重复写入
		if err := seed.LoadDemo(context.Background(), database, []module.Module{course}); err != nil {
			t.Fatal(err)
		}
	}
	for _, username := range []string{"student2", "student3"} {
		if user := getUserByUsername(t, database, username); user.Role != access.RoleStudent {
			t.Errorf("%s role = %s", username, user.Role)
		}
	}
	if count := getRowCount(t, database, "sys_user"); count != 6 {
		t.Errorf("users = %d, want 6", count)
	}
	if course.demoCalls != 2 {
		t.Errorf("module LoadDemo called %d times, want 2", course.demoCalls)
	}
}

// 模块写入演示数据失败时整个事务回滚，演示账号也不会留下。
func TestLoadDemoRollsBackOnModuleError(t *testing.T) {
	course := &courseModule{demoErr: errors.New("bad fixture")}
	database := testutil.NewDB(t, course)

	if err := seed.LoadDemo(context.Background(), database, []module.Module{course}); err == nil {
		t.Fatal("expected error")
	}
	if count := getRowCount(t, database, "sys_user"); count != 4 {
		t.Errorf("users = %d after failed demo load, want 4 (rolled back)", count)
	}
}
