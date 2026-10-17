package router

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"learnhub/internal/framework/access"
	"learnhub/internal/framework/module"
	"learnhub/internal/framework/testutil"
	"learnhub/internal/platform/resp"
)

func init() { gin.SetMode(gin.TestMode) }

// pingModule 注册一个 /api/ping 路由，用来确认模块路由被挂上了。
type pingModule struct{}

func (pingModule) Name() string                        { return "ping" }
func (pingModule) Namespaces() []string                { return nil }
func (pingModule) Permissions() []access.PermissionDef { return nil }
func (pingModule) DefaultGrants() map[string][]string  { return nil }
func (pingModule) Models() []any                       { return nil }
func (pingModule) Register(api *gin.RouterGroup, deps module.Deps) {
	api.GET("/ping", resp.H(pongHandler))
}

func pongHandler(*gin.Context) (any, error) { return "pong", nil }

func newTestEngine(t *testing.T, modules ...module.Module) *gin.Engine {
	t.Helper()
	registry, err := module.NewRegistry(modules...)
	if err != nil {
		t.Fatal(err)
	}
	return New(Deps{DB: testutil.NewDB(t, modules...), Registry: registry})
}

func get(t *testing.T, engine *gin.Engine, path string) (int, resp.Result) {
	t.Helper()
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
	var result resp.Result
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("GET %s: body is not a Result: %q", path, recorder.Body.String())
	}
	return recorder.Code, result
}

func TestHealth(t *testing.T) {
	status, result := get(t, newTestEngine(t), "/api/health")
	if status != 200 || result.Code != 0 {
		t.Fatalf("health: status=%d result=%+v", status, result)
	}
	if data, _ := result.Data.(map[string]any); data["status"] != "ok" {
		t.Errorf("health data = %v", result.Data)
	}
}

func TestUnknownRouteIsJSON404(t *testing.T) {
	status, result := get(t, newTestEngine(t), "/api/nope")
	if status != 404 || result.Code != 404 {
		t.Errorf("unknown route: status=%d result=%+v", status, result)
	}
}

func TestModuleRoutesAreRegistered(t *testing.T) {
	status, result := get(t, newTestEngine(t, pingModule{}), "/api/ping")
	if status != 200 || result.Data != "pong" {
		t.Errorf("module route: status=%d result=%+v", status, result)
	}
}
