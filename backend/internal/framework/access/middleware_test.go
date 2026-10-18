package access

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"caveclass/internal/platform/middleware"
	"caveclass/internal/platform/resp"
)

func init() { gin.SetMode(gin.TestMode) }

// principalSetter 模拟认证中间件：把指定的用户写入请求；principal 为 nil 表示未登录。
type principalSetter struct {
	principal *Principal
}

func (setter principalSetter) handle(ctx *gin.Context) {
	if setter.principal != nil {
		SetPrincipal(ctx, setter.principal)
	}
	ctx.Next()
}

func okHandler(*gin.Context) (any, error) { return "ok", nil }

func requestStatus(principal *Principal, guard gin.HandlerFunc) int {
	engine := gin.New()
	engine.Use(middleware.ErrorHandler(), principalSetter{principal: principal}.handle)
	engine.GET("/target", guard, resp.ToGinHandler(okHandler))
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest("GET", "/target", nil))
	return recorder.Code
}

func TestRequirePermission(t *testing.T) {
	guard := RequirePermission("teacher:course:create")
	cases := []struct {
		name      string
		principal *Principal
		status    int
	}{
		{"not logged in", nil, 401},
		{"missing permission", &Principal{Role: RoleStudent, Permissions: []string{"student:my:view"}}, 403},
		{"exact grant", &Principal{Role: RoleTeacher, Permissions: []string{"teacher:course:create"}}, 200},
		{"wildcard grant", &Principal{Role: RoleTeacher, Permissions: []string{"teacher:*"}}, 200},
		{"admin bypass", &Principal{Role: RoleAdmin}, 200},
	}
	for _, testCase := range cases {
		if status := requestStatus(testCase.principal, guard); status != testCase.status {
			t.Errorf("%s: status = %d, want %d", testCase.name, status, testCase.status)
		}
	}
}

func TestRequireLogin(t *testing.T) {
	if status := requestStatus(nil, RequireLogin()); status != 401 {
		t.Errorf("not logged in: status = %d, want 401", status)
	}
	if status := requestStatus(&Principal{Role: RoleStudent}, RequireLogin()); status != 200 {
		t.Errorf("logged in without any permission: status = %d, want 200", status)
	}
}

// 路由表里把权限码写错，应在启动注册路由时就失败。
func TestRequirePermissionPanicsOnInvalidCode(t *testing.T) {
	for _, code := range []string{"", "teacher::view", "teacher:*:view"} {
		if !panics(code) {
			t.Errorf("RequirePermission(%q) should panic", code)
		}
	}
}

func panics(code string) (didPanic bool) {
	defer recoverAsFlag(&didPanic)
	RequirePermission(code)
	return false
}

func recoverAsFlag(didPanic *bool) {
	if recover() != nil {
		*didPanic = true
	}
}
