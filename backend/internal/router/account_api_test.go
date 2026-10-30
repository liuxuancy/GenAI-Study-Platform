package router

import (
	"testing"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/model"
	"caveclass/internal/framework/seed"
	"caveclass/internal/framework/testutil"
)

func (server *testServer) login(t *testing.T, username, password string) string {
	t.Helper()
	status, result := server.call(t, "POST", "/api/account/login", "",
		map[string]string{"username": username, "password": password})
	if status != 200 {
		t.Fatalf("login %s: status=%d msg=%q", username, status, result.Msg)
	}
	data, _ := result.Data.(map[string]any)
	token, _ := data["token"].(string)
	if token == "" {
		t.Fatalf("login %s: no token in %+v", username, result.Data)
	}
	return token
}

func TestLoginWithEveryTestAccount(t *testing.T) {
	server := newTestServer(t)
	for _, testAccount := range seed.GetTestAccounts() {
		server.login(t, testAccount.Username, testAccount.Password)
	}
	server.login(t, "  ADMIN ", "admin123") // 用户名不区分大小写，前后空格会去掉
}

// 用户不存在和密码错误返回完全相同的结果，不泄露用户名是否存在。
func TestLoginFailuresLookTheSame(t *testing.T) {
	server := newTestServer(t)
	wrongPasswordStatus, wrongPassword := server.call(t, "POST", "/api/account/login", "",
		map[string]string{"username": "admin", "password": "wrong"})
	unknownUserStatus, unknownUser := server.call(t, "POST", "/api/account/login", "",
		map[string]string{"username": "nobody", "password": "wrong"})
	if wrongPasswordStatus != 401 || unknownUserStatus != 401 || wrongPassword.Msg != unknownUser.Msg {
		t.Errorf("wrong password: %d %q; unknown user: %d %q",
			wrongPasswordStatus, wrongPassword.Msg, unknownUserStatus, unknownUser.Msg)
	}
	if status, _ := server.call(t, "POST", "/api/account/login", "", map[string]string{"username": "admin"}); status != 400 {
		t.Errorf("missing password: status = %d, want 400", status)
	}
}

func TestDisabledUserCannotLogin(t *testing.T) {
	server := newTestServer(t)
	testutil.MustExec(t, server.database, "UPDATE sys_user SET status = ? WHERE username = ?", model.UserStatusDisabled, "student1")
	status, result := server.call(t, "POST", "/api/account/login", "",
		map[string]string{"username": "student1", "password": "student123"})
	if status != 401 || result.Msg != "account is disabled" {
		t.Errorf("status=%d msg=%q", status, result.Msg)
	}
}

func TestMe(t *testing.T) {
	server := newTestServer(t)
	testutil.MustExec(t, server.database, "INSERT INTO sys_role_permission (role, permission_code) VALUES (?, ?)",
		access.RoleStudent, "student:my:view")

	_, student := server.call(t, "GET", "/api/account/me", server.login(t, "student1", "student123"), nil)
	studentData, _ := student.Data.(map[string]any)
	if studentData["username"] != "student1" || studentData["role"] != access.RoleStudent ||
		studentData["roleName"] != "学生" || studentData["isAdmin"] != false ||
		studentData["major"] != "Computer Science" {
		t.Errorf("student me = %+v", studentData)
	}
	if permissions, _ := studentData["permissions"].([]any); len(permissions) != 1 || permissions[0] != "student:my:view" {
		t.Errorf("student permissions = %v", studentData["permissions"])
	}
	if _, exists := studentData["passwordHash"]; exists {
		t.Error("me must not expose the password hash")
	}

	_, admin := server.call(t, "GET", "/api/account/me", server.login(t, "admin", "admin123"), nil)
	adminData, _ := admin.Data.(map[string]any)
	if adminData["isAdmin"] != true {
		t.Errorf("admin me = %+v", adminData)
	}
	if permissions, ok := adminData["permissions"].([]any); !ok || len(permissions) != 0 {
		t.Errorf("admin permissions = %v, want an empty array (not null)", adminData["permissions"])
	}
}

func TestAccountRoutesRequireLogin(t *testing.T) {
	server := newTestServer(t)
	routes := [][2]string{
		{"GET", "/api/account/me"},
		{"POST", "/api/account/logout"},
		{"PUT", "/api/account/profile"},
		{"PUT", "/api/account/password"},
	}
	for _, route := range routes {
		if status, _ := server.call(t, route[0], route[1], "", nil); status != 401 {
			t.Errorf("%s %s without token: status = %d, want 401", route[0], route[1], status)
		}
	}
}

func TestUpdateProfile(t *testing.T) {
	server := newTestServer(t)
	token := server.login(t, "teacher1", "teacher123")

	status, _ := server.call(t, "PUT", "/api/account/profile", token,
		map[string]string{"nickname": " 新昵称 ", "email": "teacher1@example.com"})
	if status != 200 {
		t.Fatalf("update profile: status = %d", status)
	}
	_, me := server.call(t, "GET", "/api/account/me", token, nil)
	data, _ := me.Data.(map[string]any)
	if data["nickname"] != "新昵称" || data["email"] != "teacher1@example.com" {
		t.Errorf("me after update = %+v", data)
	}

	if status, _ := server.call(t, "PUT", "/api/account/profile", token, map[string]string{"email": "not-an-email"}); status != 400 {
		t.Errorf("invalid email: status = %d, want 400", status)
	}
}

// 改密码后，包括当前设备在内的旧 token 立即失效，用新密码可以重新登录。
func TestChangePassword(t *testing.T) {
	server := newTestServer(t)
	token := server.login(t, "student1", "student123")

	cases := []struct {
		name string
		body map[string]string
	}{
		{"wrong old password", map[string]string{"oldPassword": "wrong", "newPassword": "newpass123"}},
		{"same as old", map[string]string{"oldPassword": "student123", "newPassword": "student123"}},
		{"too short", map[string]string{"oldPassword": "student123", "newPassword": "12345"}},
	}
	for _, testCase := range cases {
		// 都是 400 而不是 401：前端收到 401 会退出登录
		if status, _ := server.call(t, "PUT", "/api/account/password", token, testCase.body); status != 400 {
			t.Errorf("%s: status = %d, want 400", testCase.name, status)
		}
	}

	status, _ := server.call(t, "PUT", "/api/account/password", token,
		map[string]string{"oldPassword": "student123", "newPassword": "newpass123"})
	if status != 200 {
		t.Fatalf("change password: status = %d", status)
	}
	if status, _ := server.call(t, "GET", "/api/account/me", token, nil); status != 401 {
		t.Errorf("old token after password change: status = %d, want 401", status)
	}
	if status, _ := server.call(t, "POST", "/api/account/login", "",
		map[string]string{"username": "student1", "password": "student123"}); status != 401 {
		t.Errorf("login with old password: status = %d, want 401", status)
	}
	newToken := server.login(t, "student1", "newpass123")
	if status, _ := server.call(t, "GET", "/api/account/me", newToken, nil); status != 200 {
		t.Errorf("new token: status = %d, want 200", status)
	}
}

func TestLogout(t *testing.T) {
	server := newTestServer(t)
	if status, _ := server.call(t, "POST", "/api/account/logout", server.login(t, "admin", "admin123"), nil); status != 200 {
		t.Errorf("logout: status = %d, want 200", status)
	}
}
