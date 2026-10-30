package router

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"gorm.io/gorm"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/model"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/errs"
)

// adminToken 登录管理员，返回 token。
func (server *testServer) adminToken(t *testing.T) string {
	t.Helper()
	return server.login(t, "admin", "admin123")
}

func dataAsMap(result any) map[string]any {
	data, _ := result.(map[string]any)
	return data
}

func dataAsList(result any) []any {
	list, _ := result.([]any)
	return list
}

func stringList(values []any) []string {
	list := []string{}
	for _, value := range values {
		text, _ := value.(string)
		list = append(list, text)
	}
	return list
}

// 讲师、学生都没有 system 权限码，调用管理接口一律 403；没登录 401。
func TestSystemRoutesRejectNonAdmins(t *testing.T) {
	server := newTestServer(t)
	routes := [][2]string{
		{"GET", "/api/system/users"},
		{"POST", "/api/system/users"},
		{"PUT", "/api/system/users/1"},
		{"PUT", "/api/system/users/1/password"},
		{"DELETE", "/api/system/users/1"},
		{"GET", "/api/system/roles"},
		{"PUT", "/api/system/roles/ROLE_STUDENT/permissions"},
		{"GET", "/api/system/permissions"},
		{"GET", "/api/system/classes"},
		{"POST", "/api/system/classes"},
		{"PUT", "/api/system/classes/1"},
		{"PUT", "/api/system/classes/1/members"},
	}
	teacherToken := server.login(t, "teacher1", "teacher123")
	studentToken := server.login(t, "student1", "student123")
	for _, route := range routes {
		for _, token := range []string{teacherToken, studentToken} {
			if status, _ := server.call(t, route[0], route[1], token, map[string]any{}); status != 403 {
				t.Errorf("%s %s as non-admin: status = %d, want 403", route[0], route[1], status)
			}
		}
		if status, _ := server.call(t, route[0], route[1], "", nil); status != 401 {
			t.Errorf("%s %s without token: status = %d, want 401", route[0], route[1], status)
		}
	}
}

func TestGetUserList(t *testing.T) {
	server := newTestServer(t)
	token := server.adminToken(t)

	cases := []struct {
		query     string
		total     float64
		usernames []string
	}{
		{"", 4, []string{"admin", "teacher1", "teacher2", "student1"}},
		{"?role=ROLE_TEACHER", 2, []string{"teacher1", "teacher2"}},
		{"?keyword=teach", 2, []string{"teacher1", "teacher2"}},
		{"?keyword=" + "%25", 0, []string{}},     // %25 是 "%"，必须按普通字符匹配，而不是通配所有
		{"?keyword=学生", 1, []string{"student1"}}, // 按昵称匹配
		{"?page=2&pageSize=3", 4, []string{"student1"}},
		{"?status=0", 0, []string{}},
	}
	for _, testCase := range cases {
		status, result := server.call(t, "GET", "/api/system/users"+testCase.query, token, nil)
		if status != 200 {
			t.Errorf("%q: status = %d msg=%q", testCase.query, status, result.Msg)
			continue
		}
		data := dataAsMap(result.Data)
		var usernames []string
		for _, item := range dataAsList(data["items"]) {
			usernames = append(usernames, dataAsMap(item)["username"].(string))
		}
		if data["total"] != testCase.total || !slices.Equal(usernames, testCase.usernames) {
			t.Errorf("%q: total=%v usernames=%v, want %v %v", testCase.query, data["total"], usernames, testCase.total, testCase.usernames)
		}
	}

	if status, _ := server.call(t, "GET", "/api/system/users?status=2", token, nil); status != 400 {
		t.Errorf("invalid status filter: status = %d, want 400", status)
	}
}

func TestCreateUser(t *testing.T) {
	server := newTestServer(t)
	token := server.adminToken(t)

	status, result := server.call(t, "POST", "/api/system/users", token, map[string]string{
		"username": "Teacher3", "password": "teacher123", "nickname": "讲师三号", "role": access.RoleTeacher,
	})
	if status != 200 {
		t.Fatalf("create: status=%d msg=%q", status, result.Msg)
	}
	created := dataAsMap(result.Data)
	if created["username"] != "teacher3" || created["roleName"] != "讲师" || created["status"] != float64(model.UserStatusActive) {
		t.Errorf("created = %+v", created)
	}
	if _, exists := created["passwordHash"]; exists {
		t.Error("created user must not expose password hash")
	}
	server.login(t, "teacher3", "teacher123") // 新账号可以登录

	cases := []struct {
		name   string
		body   map[string]string
		status int
	}{
		{"duplicate username", map[string]string{"username": "TEACHER3", "password": "123456", "role": access.RoleStudent}, 409},
		{"unknown role", map[string]string{"username": "someone", "password": "123456", "role": "ROLE_GUEST"}, 400},
		{"bad username", map[string]string{"username": "a b", "password": "123456", "role": access.RoleStudent}, 400},
		{"short username", map[string]string{"username": "ab", "password": "123456", "role": access.RoleStudent}, 400},
		{"short password", map[string]string{"username": "someone", "password": "12345", "role": access.RoleStudent}, 400},
		{"invalid email", map[string]string{"username": "someone", "password": "123456", "role": access.RoleStudent, "email": "x"}, 400},
	}
	for _, testCase := range cases {
		if status, result := server.call(t, "POST", "/api/system/users", token, testCase.body); status != testCase.status {
			t.Errorf("%s: status=%d msg=%q, want %d", testCase.name, status, result.Msg, testCase.status)
		}
	}
}

// 禁用后该用户的旧 token 立即失效、也不能再登录；重新启用后可以登录，但旧 token 仍然失效。
func TestDisableAndEnableUser(t *testing.T) {
	server := newTestServer(t)
	token := server.adminToken(t)
	studentToken := server.login(t, "student1", "student123")
	studentPath := fmt.Sprintf("/api/system/users/%d", testutil.GetUserID(t, server.database, "student1"))

	status, result := server.call(t, "PUT", studentPath, token, map[string]any{"nickname": "改过的昵称", "email": "", "status": 0})
	if status != 200 {
		t.Fatalf("disable: status=%d msg=%q", status, result.Msg)
	}
	if status, _ := server.call(t, "GET", "/api/account/me", studentToken, nil); status != 401 {
		t.Errorf("old token after disable: status = %d, want 401", status)
	}
	if status, _ := server.call(t, "POST", "/api/account/login", "", map[string]string{"username": "student1", "password": "student123"}); status != 401 {
		t.Errorf("login while disabled: status = %d, want 401", status)
	}

	server.call(t, "PUT", studentPath, token, map[string]any{"nickname": "改过的昵称", "status": 1})
	newStudentToken := server.login(t, "student1", "student123")
	_, me := server.call(t, "GET", "/api/account/me", newStudentToken, nil)
	if dataAsMap(me.Data)["nickname"] != "改过的昵称" {
		t.Errorf("nickname not updated: %+v", me.Data)
	}
	if status, _ := server.call(t, "GET", "/api/account/me", studentToken, nil); status != 401 {
		t.Errorf("token issued before disable must stay invalid after re-enable: status = %d", status)
	}
}

func TestUpdateUserRules(t *testing.T) {
	server := newTestServer(t)
	token := server.adminToken(t)
	adminPath := fmt.Sprintf("/api/system/users/%d", testutil.GetUserID(t, server.database, "admin"))

	cases := []struct {
		name   string
		path   string
		body   map[string]any
		status int
	}{
		{"disable self", adminPath, map[string]any{"status": 0}, 400},
		{"missing status", adminPath, map[string]any{"nickname": "x"}, 400},
		{"invalid status", adminPath, map[string]any{"status": 5}, 400},
		{"unknown user", "/api/system/users/99999", map[string]any{"status": 1}, 404},
		{"bad id", "/api/system/users/abc", map[string]any{"status": 1}, 400},
		{"update self nickname", adminPath, map[string]any{"nickname": "超级管理员", "status": 1}, 200},
	}
	for _, testCase := range cases {
		if status, result := server.call(t, "PUT", testCase.path, token, testCase.body); status != testCase.status {
			t.Errorf("%s: status=%d msg=%q, want %d", testCase.name, status, result.Msg, testCase.status)
		}
	}
}

// 管理员重置密码后，该用户旧 token 立即失效，用新密码可以登录。
func TestResetUserPassword(t *testing.T) {
	server := newTestServer(t)
	token := server.adminToken(t)
	teacherToken := server.login(t, "teacher1", "teacher123")
	path := fmt.Sprintf("/api/system/users/%d/password", testutil.GetUserID(t, server.database, "teacher1"))

	if status, _ := server.call(t, "PUT", path, token, map[string]string{"password": "123"}); status != 400 {
		t.Errorf("short password: status = %d, want 400", status)
	}
	if status, result := server.call(t, "PUT", path, token, map[string]string{"password": "reset-123"}); status != 200 {
		t.Fatalf("reset: status=%d msg=%q", status, result.Msg)
	}
	if status, _ := server.call(t, "GET", "/api/account/me", teacherToken, nil); status != 401 {
		t.Errorf("old token after reset: status = %d, want 401", status)
	}
	server.login(t, "teacher1", "reset-123")
}

func TestDeleteUser(t *testing.T) {
	server := newTestServer(t)
	token := server.adminToken(t)
	studentToken := server.login(t, "student1", "student123")

	adminPath := fmt.Sprintf("/api/system/users/%d", testutil.GetUserID(t, server.database, "admin"))
	if status, _ := server.call(t, "DELETE", adminPath, token, nil); status != 400 {
		t.Errorf("delete self: status = %d, want 400", status)
	}
	if status, _ := server.call(t, "DELETE", "/api/system/users/99999", token, nil); status != 404 {
		t.Errorf("delete unknown: status = %d, want 404", status)
	}

	studentPath := fmt.Sprintf("/api/system/users/%d", testutil.GetUserID(t, server.database, "student1"))
	if status, result := server.call(t, "DELETE", studentPath, token, nil); status != 200 {
		t.Fatalf("delete: status=%d msg=%q", status, result.Msg)
	}
	if status, _ := server.call(t, "GET", "/api/account/me", studentToken, nil); status != 401 {
		t.Errorf("deleted user's token: status = %d, want 401", status)
	}
}

// ownedCourseGuard 假模块：模拟"讲师名下有课程时不能删除"。
type ownedCourseGuard struct {
	ownerUsername string
}

func (ownedCourseGuard) GetName() string                        { return "fake-course" }
func (ownedCourseGuard) GetNamespaces() []string                { return nil }
func (ownedCourseGuard) GetPermissions() []access.PermissionDef { return nil }
func (ownedCourseGuard) GetDefaultGrants() map[string][]string  { return nil }
func (ownedCourseGuard) GetModels() []any                       { return nil }

func (guard ownedCourseGuard) CheckUserDeletion(ctx context.Context, database *gorm.DB, userID int64) error {
	var username string
	if err := database.Raw("SELECT username FROM sys_user WHERE id = ?", userID).Scan(&username).Error; err != nil {
		return err
	}
	if username == guard.ownerUsername {
		return errs.Conflict("teacher still owns courses and cannot be deleted")
	}
	return nil
}

func TestDeleteUserBlockedByOtherModule(t *testing.T) {
	server := newTestServer(t, ownedCourseGuard{ownerUsername: "teacher1"})
	token := server.adminToken(t)

	teacherPath := fmt.Sprintf("/api/system/users/%d", testutil.GetUserID(t, server.database, "teacher1"))
	if status, result := server.call(t, "DELETE", teacherPath, token, nil); status != 409 || result.Msg != "teacher still owns courses and cannot be deleted" {
		t.Errorf("guarded delete: status=%d msg=%q, want 409", status, result.Msg)
	}
	otherPath := fmt.Sprintf("/api/system/users/%d", testutil.GetUserID(t, server.database, "teacher2"))
	if status, _ := server.call(t, "DELETE", otherPath, token, nil); status != 200 {
		t.Errorf("unguarded delete: status = %d, want 200", status)
	}
}

func TestGetRoleList(t *testing.T) {
	server := newTestServer(t)
	testutil.MustExec(t, server.database, "INSERT INTO sys_role_permission (role, permission_code) VALUES (?, ?), (?, ?)",
		access.RoleTeacher, "system:user:view",
		access.RoleTeacher, "system:removed:code") // 模拟代码里已经删掉的权限码

	status, result := server.call(t, "GET", "/api/system/roles", server.adminToken(t), nil)
	if status != 200 {
		t.Fatalf("status=%d msg=%q", status, result.Msg)
	}
	roles := dataAsList(result.Data)
	if len(roles) != 3 {
		t.Fatalf("roles = %v", roles)
	}
	admin, teacher := dataAsMap(roles[0]), dataAsMap(roles[1])
	if admin["code"] != access.RoleAdmin || admin["editable"] != false {
		t.Errorf("admin role = %+v", admin)
	}
	if teacher["editable"] != true ||
		!slices.Equal(stringList(dataAsList(teacher["permissions"])), []string{"system:user:view"}) ||
		!slices.Equal(stringList(dataAsList(teacher["stalePermissions"])), []string{"system:removed:code"}) {
		t.Errorf("teacher role = %+v", teacher)
	}
}

// 给学生授权后，学生不需要重新登录，下一个请求就拥有新权限；撤销同理。
func TestAssignRolePermissions(t *testing.T) {
	server := newTestServer(t)
	token := server.adminToken(t)
	studentToken := server.login(t, "student1", "student123")
	path := "/api/system/roles/ROLE_STUDENT/permissions"

	if status, _ := server.call(t, "GET", "/api/system/users", studentToken, nil); status != 403 {
		t.Fatalf("before grant: status = %d, want 403", status)
	}
	status, result := server.call(t, "PUT", path, token, map[string]any{"permissions": []string{"system:user:view", "system:user:view"}})
	if status != 200 || !slices.Equal(stringList(dataAsList(result.Data)), []string{"system:user:view"}) {
		t.Fatalf("grant: status=%d data=%v msg=%q", status, result.Data, result.Msg)
	}
	if status, _ := server.call(t, "GET", "/api/system/users", studentToken, nil); status != 200 {
		t.Errorf("after grant: status = %d, want 200", status)
	}

	server.call(t, "PUT", path, token, map[string]any{"permissions": []string{}})
	if status, _ := server.call(t, "GET", "/api/system/users", studentToken, nil); status != 403 {
		t.Errorf("after revoke: status = %d, want 403", status)
	}

	cases := []struct {
		name   string
		path   string
		body   map[string]any
		status int
	}{
		{"admin role", "/api/system/roles/ROLE_ADMIN/permissions", map[string]any{"permissions": []string{}}, 400},
		{"unknown role", "/api/system/roles/ROLE_GUEST/permissions", map[string]any{"permissions": []string{}}, 404},
		{"unknown code", path, map[string]any{"permissions": []string{"system:user:fly"}}, 400},
		{"global wildcard", path, map[string]any{"permissions": []string{"*"}}, 400},
		{"missing field", path, map[string]any{}, 400},
		{"namespace wildcard", path, map[string]any{"permissions": []string{"system:*"}}, 200},
	}
	for _, testCase := range cases {
		if status, result := server.call(t, "PUT", testCase.path, token, testCase.body); status != testCase.status {
			t.Errorf("%s: status=%d msg=%q, want %d", testCase.name, status, result.Msg, testCase.status)
		}
	}
}

func TestGetPermissionCatalog(t *testing.T) {
	server := newTestServer(t)
	status, result := server.call(t, "GET", "/api/system/permissions", server.adminToken(t), nil)
	if status != 200 {
		t.Fatalf("status=%d msg=%q", status, result.Msg)
	}
	var codes []string
	for _, definition := range dataAsList(result.Data) {
		codes = append(codes, dataAsMap(definition)["code"].(string))
	}
	if len(codes) != 11 || codes[0] != "system:user:view" || !slices.Contains(codes, "system:class:view") {
		t.Errorf("catalog codes = %v", codes)
	}
}
