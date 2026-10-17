package seed_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"learnhub/internal/framework/access"
	"learnhub/internal/framework/identity"
	"learnhub/internal/framework/model"
	"learnhub/internal/framework/module"
	"learnhub/internal/framework/seed"
	"learnhub/internal/framework/testutil"
)

// courseModule 带默认授权和演示数据的假模块。
type courseModule struct {
	demoCalls int
	demoErr   error
}

func (course *courseModule) Name() string         { return "course" }
func (course *courseModule) Namespaces() []string { return []string{"teacher", "student"} }
func (course *courseModule) Permissions() []access.PermissionDef {
	return []access.PermissionDef{
		{Code: "teacher:course:view", Name: "查看课程", Type: access.PermissionTypeMenu},
		{Code: "student:catalog:view", Name: "课程广场", Type: access.PermissionTypeMenu},
		{Code: "student:my:view", Name: "我的学习", Type: access.PermissionTypeMenu},
	}
}
func (course *courseModule) DefaultGrants() map[string][]string {
	return map[string][]string{
		access.RoleTeacher: {"teacher:*"},
		access.RoleStudent: {"student:catalog:view", "student:my:view"},
	}
}
func (course *courseModule) Models() []any                                   { return nil }
func (course *courseModule) Register(api *gin.RouterGroup, deps module.Deps) {}
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
	if err := seed.Run(context.Background(), database, course.DefaultGrants()); err != nil {
		t.Fatal(err)
	}
}

func findUser(t *testing.T, database *gorm.DB, username string) model.User {
	t.Helper()
	var user model.User
	if err := database.Where("username = ?", username).First(&user).Error; err != nil {
		t.Fatalf("user %s: %v", username, err)
	}
	return user
}

func countRows(t *testing.T, database *gorm.DB, table any) int64 {
	t.Helper()
	var count int64
	if err := database.Model(table).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func grantsOf(t *testing.T, database *gorm.DB, role string) []string {
	t.Helper()
	var codes []string
	if err := database.Model(&model.RolePermission{}).Where("role = ?", role).
		Order("permission_code").Pluck("permission_code", &codes).Error; err != nil {
		t.Fatal(err)
	}
	return codes
}

func TestRunCreatesAccountsAndGrants(t *testing.T) {
	database := testutil.NewDB(t, &courseModule{})

	if count := countRows(t, database, &model.User{}); count != 4 {
		t.Errorf("users = %d, want 4", count)
	}
	for _, account := range seed.TestAccounts() {
		user := findUser(t, database, account.Username)
		if user.Role != account.Role || user.Status != model.UserStatusActive || user.TokenVersion != 0 {
			t.Errorf("%s: role=%s status=%d tokenVersion=%d", account.Username, user.Role, user.Status, user.TokenVersion)
		}
		if !identity.CheckPassword(user.PasswordHash, account.Password) {
			t.Errorf("%s: stored hash does not match password", account.Username)
		}
	}

	if got := grantsOf(t, database, access.RoleTeacher); !slices.Equal(got, []string{"teacher:*"}) {
		t.Errorf("teacher grants = %v", got)
	}
	if got := grantsOf(t, database, access.RoleStudent); !slices.Equal(got, []string{"student:catalog:view", "student:my:view"}) {
		t.Errorf("student grants = %v", got)
	}
	if got := grantsOf(t, database, access.RoleAdmin); len(got) != 0 {
		t.Errorf("admin must have no grant rows, got %v", got)
	}
}

func TestRunIsRepeatable(t *testing.T) {
	course := &courseModule{}
	database := testutil.NewDB(t, course)
	runSeed(t, database, course)
	runSeed(t, database, course)

	if count := countRows(t, database, &model.User{}); count != 4 {
		t.Errorf("users = %d after repeated seed, want 4", count)
	}
	if count := countRows(t, database, &model.RolePermission{}); count != 3 {
		t.Errorf("grants = %d after repeated seed, want 3", count)
	}
}

// 管理员撤销过的授权，重启（再次执行种子）后不能被恢复。
func TestRunKeepsAdminChangesToGrants(t *testing.T) {
	course := &courseModule{}
	database := testutil.NewDB(t, course)
	revoke := database.Where("role = ? AND permission_code = ?", access.RoleStudent, "student:my:view").
		Delete(&model.RolePermission{})
	if revoke.Error != nil || revoke.RowsAffected != 1 {
		t.Fatalf("revoke: %v, rows=%d", revoke.Error, revoke.RowsAffected)
	}

	runSeed(t, database, course)
	if got := grantsOf(t, database, access.RoleStudent); !slices.Equal(got, []string{"student:catalog:view"}) {
		t.Errorf("revoked grant came back: %v", got)
	}
}

func TestRunDoesNotResetExistingPassword(t *testing.T) {
	course := &courseModule{}
	database := testutil.NewDB(t, course)
	newHash, err := identity.HashPassword("changed-by-admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&model.User{}).Where("username = ?", "admin").
		Update("password_hash", newHash).Error; err != nil {
		t.Fatal(err)
	}

	runSeed(t, database, course)
	if !identity.CheckPassword(findUser(t, database, "admin").PasswordHash, "changed-by-admin") {
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
		if user := findUser(t, database, username); user.Role != access.RoleStudent {
			t.Errorf("%s role = %s", username, user.Role)
		}
	}
	if count := countRows(t, database, &model.User{}); count != 6 {
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
	if count := countRows(t, database, &model.User{}); count != 4 {
		t.Errorf("users = %d after failed demo load, want 4 (rolled back)", count)
	}
}
