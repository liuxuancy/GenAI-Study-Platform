package router

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"learnhub/internal/db"
	"learnhub/internal/resp"
)

func init() { gin.SetMode(gin.TestMode) }

func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gdb, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(gdb) })
	return New(Deps{DB: gdb})
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
