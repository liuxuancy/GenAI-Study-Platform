package identity_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/identity"
	"caveclass/internal/framework/model"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/middleware"
	"caveclass/internal/platform/resp"
)

func init() { gin.SetMode(gin.TestMode) }

type authenticateFixture struct {
	database *gorm.DB
	issuer   *identity.TokenIssuer
	cache    *access.Cache
	engine   *gin.Engine
}

// newAuthenticateFixture 搭一个最小的服务：ErrorHandler + Authenticate，
// /api/health 公开，/api/private 只要求登录，/api/my 要求 student:my:view。
func newAuthenticateFixture(t *testing.T) *authenticateFixture {
	t.Helper()
	database := testutil.NewDB(t)
	fixture := &authenticateFixture{
		database: database,
		issuer:   identity.NewTokenIssuer("test-secret", time.Hour),
		cache:    access.NewCache(time.Minute, access.NewDBLoader(database)),
		engine:   gin.New(),
	}
	testutil.MustExec(t, database, "INSERT INTO sys_role_permission (role, permission_code) VALUES (?, ?)",
		access.RoleStudent, "student:my:view")
	fixture.engine.Use(middleware.ErrorHandler())
	api := fixture.engine.Group("/api")
	api.Use(identity.Authenticate(fixture.issuer, fixture.cache))
	api.GET("/health", resp.ToGinHandler(okHandler))
	api.GET("/private", access.RequireLogin(), resp.ToGinHandler(currentUserHandler))
	api.GET("/my", access.RequirePermission("student:my:view"), resp.ToGinHandler(currentUserHandler))
	return fixture
}

func okHandler(*gin.Context) (any, error) { return "ok", nil }

func currentUserHandler(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return principal.Username, nil
}

func (fixture *authenticateFixture) issueTokenFor(t *testing.T, username string) string {
	t.Helper()
	token, _, err := fixture.issuer.Issue(testutil.GetUserID(t, fixture.database, username), 0)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func (fixture *authenticateFixture) get(t *testing.T, path, token string) (int, resp.Result) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	fixture.engine.ServeHTTP(recorder, request)
	var result resp.Result
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("GET %s: body is not a Result: %q", path, recorder.Body.String())
	}
	return recorder.Code, result
}

func TestPublicRouteNeedsNoToken(t *testing.T) {
	fixture := newAuthenticateFixture(t)
	if status, _ := fixture.get(t, "/api/health", ""); status != 200 {
		t.Errorf("public route: status = %d, want 200", status)
	}
}

func TestValidTokenSetsPrincipal(t *testing.T) {
	fixture := newAuthenticateFixture(t)
	status, result := fixture.get(t, "/api/private", fixture.issueTokenFor(t, "student1"))
	if status != 200 || result.Data != "student1" {
		t.Errorf("status=%d result=%+v", status, result)
	}
}

func TestRejectsMissingOrInvalidToken(t *testing.T) {
	fixture := newAuthenticateFixture(t)
	otherIssuer := identity.NewTokenIssuer("another-secret", time.Hour)
	forged, _, err := otherIssuer.Issue(testutil.GetUserID(t, fixture.database, "admin"), 0)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"no token":            "",
		"garbage token":       "not-a-jwt",
		"signed by other key": forged,
	}
	for name, token := range cases {
		if status, _ := fixture.get(t, "/api/private", token); status != 401 {
			t.Errorf("%s: status = %d, want 401", name, status)
		}
	}
}

func TestRejectsUnknownUser(t *testing.T) {
	fixture := newAuthenticateFixture(t)
	token, _, err := fixture.issuer.Issue(99999, 0)
	if err != nil {
		t.Fatal(err)
	}
	if status, _ := fixture.get(t, "/api/private", token); status != 401 {
		t.Errorf("status = %d, want 401", status)
	}
}

// 被禁用的用户，旧 token 立即失效（禁用操作会清空缓存）。
func TestRejectsDisabledUser(t *testing.T) {
	fixture := newAuthenticateFixture(t)
	token := fixture.issueTokenFor(t, "student1")
	if status, _ := fixture.get(t, "/api/private", token); status != 200 {
		t.Fatalf("before disable: status = %d", status)
	}
	testutil.MustExec(t, fixture.database, "UPDATE sys_user SET status = ? WHERE username = ?", model.UserStatusDisabled, "student1")
	fixture.cache.Reset()
	if status, result := fixture.get(t, "/api/private", token); status != 401 || result.Msg != "account is disabled" {
		t.Errorf("after disable: status=%d msg=%q", status, result.Msg)
	}
}

// token_version 加一（改密码、重置密码）后，旧 token 立即失效。
func TestRejectsOldTokenVersion(t *testing.T) {
	fixture := newAuthenticateFixture(t)
	token := fixture.issueTokenFor(t, "student1")
	testutil.MustExec(t, fixture.database, "UPDATE sys_user SET token_version = token_version + 1 WHERE username = ?", "student1")
	fixture.cache.Reset()
	if status, _ := fixture.get(t, "/api/private", token); status != 401 {
		t.Errorf("status = %d, want 401", status)
	}
}

// 管理员撤销权限并清空缓存后，用户不需要重新登录，下一个请求就会被拦下。
func TestRevokedPermissionTakesEffectWithoutRelogin(t *testing.T) {
	fixture := newAuthenticateFixture(t)
	token := fixture.issueTokenFor(t, "student1")
	if status, _ := fixture.get(t, "/api/my", token); status != 200 {
		t.Fatalf("before revoke: status = %d", status)
	}
	testutil.MustExec(t, fixture.database, "DELETE FROM sys_role_permission WHERE role = ? AND permission_code = ?",
		access.RoleStudent, "student:my:view")
	fixture.cache.Reset()
	if status, result := fixture.get(t, "/api/my", token); status != 403 {
		t.Errorf("after revoke: status=%d result=%+v", status, result)
	}
}

func TestTeacherCannotUseStudentPermission(t *testing.T) {
	fixture := newAuthenticateFixture(t)
	if status, _ := fixture.get(t, "/api/my", fixture.issueTokenFor(t, "teacher1")); status != 403 {
		t.Errorf("status = %d, want 403", status)
	}
	if status, _ := fixture.get(t, "/api/my", fixture.issueTokenFor(t, "admin")); status != 200 {
		t.Errorf("admin: status = %d, want 200", status)
	}
}

func TestIsPublicRoute(t *testing.T) {
	cases := map[[2]string]bool{
		{"GET", "/api/health"}:          true,
		{"POST", "/api/account/login"}:  true,
		{"GET", "/api/account/login"}:   false, // 方法不同
		{"GET", "/api/account/me"}:      false,
		{"POST", "/api/account/login/"}: false, // 路径必须完全一致
	}
	for request, want := range cases {
		if got := identity.IsPublicRoute(request[0], request[1]); got != want {
			t.Errorf("IsPublicRoute(%s %s) = %v, want %v", request[0], request[1], got, want)
		}
	}
}
