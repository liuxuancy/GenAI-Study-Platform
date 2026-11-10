package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caveclass/internal/ai"
	"caveclass/internal/framework/access"
	"caveclass/internal/framework/identity"
	"caveclass/internal/framework/module"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/modules/account"
	"caveclass/internal/modules/system"
	"caveclass/internal/platform/resp"
)

func init() { gin.SetMode(gin.TestMode) }

// testServer 和 main.go 一样组装完整的服务，只是数据库换成临时库。
type testServer struct {
	database    *gorm.DB
	httpHandler http.Handler
}

// newTestServer 启用和 main.go 相同的模块；extraModules 用于在测试里额外挂上假模块。
func newTestServer(t *testing.T, extraModules ...module.Module) *testServer {
	t.Helper()
	modules := append([]module.Module{account.Module{}, system.Module{}}, extraModules...)
	registry, err := module.NewRegistry(modules...)
	if err != nil {
		t.Fatal(err)
	}
	database := testutil.NewDB(t, modules...)
	t.Setenv("AI_PROVIDER", "mock")
	aiService, err := ai.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	httpHandler := NewHTTPHandler(Deps{
		DB:          database,
		Catalog:     registry.GetCatalog(),
		Cache:       access.NewCache(time.Minute, access.NewDBLoader(database)),
		TokenIssuer: identity.NewTokenIssuer("test-secret", time.Hour),

		UserDeletionGuards: registry.GetUserDeletionGuards(),
		AI:                 aiService,
	})
	return &testServer{database: database, httpHandler: httpHandler}
}

// call 发一个请求；token 为空表示不登录，body 为 nil 表示没有请求体。
func (server *testServer) call(t *testing.T, method, path, token string, body any) (int, resp.Result) {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, &payload)
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	server.httpHandler.ServeHTTP(recorder, request)
	var result resp.Result
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("%s %s: body is not a Result: %q", method, path, recorder.Body.String())
	}
	return recorder.Code, result
}

func TestHealth(t *testing.T) {
	status, result := newTestServer(t).call(t, "GET", "/api/health", "", nil)
	if status != 200 || result.Code != 0 {
		t.Fatalf("health: status=%d result=%+v", status, result)
	}
	if data, _ := result.Data.(map[string]any); data["status"] != "ok" {
		t.Errorf("health data = %v", result.Data)
	}
}

func TestUnknownRouteIsJSON404(t *testing.T) {
	status, result := newTestServer(t).call(t, "GET", "/api/nope", "", nil)
	if status != 404 || result.Code != 404 {
		t.Errorf("unknown route: status=%d result=%+v", status, result)
	}
}
