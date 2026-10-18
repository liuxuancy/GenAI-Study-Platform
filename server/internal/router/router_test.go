package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"learnhub/internal/framework/module"
	"learnhub/internal/framework/testutil"
	"learnhub/internal/platform/resp"
)

func init() { gin.SetMode(gin.TestMode) }

func newTestHTTPHandler(t *testing.T) http.Handler {
	t.Helper()
	registry, err := module.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	return NewHTTPHandler(Deps{DB: testutil.NewDB(t), Catalog: registry.Catalog()})
}

func get(t *testing.T, handler http.Handler, path string) (int, resp.Result) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
	var result resp.Result
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("GET %s: body is not a Result: %q", path, recorder.Body.String())
	}
	return recorder.Code, result
}

func TestHealth(t *testing.T) {
	status, result := get(t, newTestHTTPHandler(t), "/api/health")
	if status != 200 || result.Code != 0 {
		t.Fatalf("health: status=%d result=%+v", status, result)
	}
	if data, _ := result.Data.(map[string]any); data["status"] != "ok" {
		t.Errorf("health data = %v", result.Data)
	}
}

func TestUnknownRouteIsJSON404(t *testing.T) {
	status, result := get(t, newTestHTTPHandler(t), "/api/nope")
	if status != 404 || result.Code != 404 {
		t.Errorf("unknown route: status=%d result=%+v", status, result)
	}
}
