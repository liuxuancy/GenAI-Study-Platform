package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/resp"
)

func init() { gin.SetMode(gin.TestMode) }

func newEngine() *gin.Engine {
	engine := gin.New()
	engine.Use(Logger(), ErrorHandler())
	engine.GET("/ok", resp.ToGinHandler(okHandler))
	engine.GET("/forbidden", resp.ToGinHandler(forbiddenHandler))
	engine.GET("/plain-error", resp.ToGinHandler(plainErrorHandler))
	engine.GET("/panic", resp.ToGinHandler(panicHandler))
	engine.POST("/bind", resp.ToGinHandler(bindHandler))
	return engine
}

func okHandler(*gin.Context) (any, error) {
	return gin.H{"hello": "world"}, nil
}

func forbiddenHandler(*gin.Context) (any, error) {
	return nil, errs.Forbidden("not your course")
}

func plainErrorHandler(*gin.Context) (any, error) {
	return nil, errors.New("db connection reset: secret-host:3306")
}

func panicHandler(*gin.Context) (any, error) {
	panic("boom")
}

func bindHandler(ctx *gin.Context) (any, error) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return input.Name, nil
}

func doRequest(t *testing.T, handler http.Handler, method, path, body string) (int, resp.Result) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	var result resp.Result
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("%s %s: body is not a Result: %q", method, path, recorder.Body.String())
	}
	return recorder.Code, result
}

func TestErrorHandler(t *testing.T) {
	engine := newEngine()
	cases := []struct {
		name, method, path, body string
		status, code             int
		msg                      string
	}{
		{"success", "GET", "/ok", "", 200, 0, "ok"},
		{"business error", "GET", "/forbidden", "", 403, 403, "not your course"},
		{"unknown error hides details", "GET", "/plain-error", "", 500, 500, errs.ErrInternal.Msg},
		{"panic", "GET", "/panic", "", 500, 500, errs.ErrInternal.Msg},
		{"bind failure", "POST", "/bind", `{}`, 400, 400, ""},
		{"bind success", "POST", "/bind", `{"name":"a"}`, 200, 0, "ok"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, result := doRequest(t, engine, testCase.method, testCase.path, testCase.body)
			if status != testCase.status || result.Code != testCase.code {
				t.Errorf("got status=%d code=%d, want %d/%d (msg=%q)", status, result.Code, testCase.status, testCase.code, result.Msg)
			}
			if testCase.msg != "" && result.Msg != testCase.msg {
				t.Errorf("msg = %q, want %q", result.Msg, testCase.msg)
			}
			if strings.Contains(result.Msg, "secret-host") {
				t.Errorf("internal details leaked: %q", result.Msg)
			}
		})
	}
}
