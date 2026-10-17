package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"learnhub/internal/platform/logs"
)

func echoRequestIDHandler(ctx *gin.Context) {
	ctx.String(http.StatusOK, logs.RequestID(ctx.Request.Context()))
}

func TestRequestID(t *testing.T) {
	engine := gin.New()
	engine.Use(RequestID())
	engine.GET("/id", echoRequestIDHandler)

	cases := []struct {
		name, header string
		keep         bool
	}{
		{"generated when missing", "", false},
		{"kept when valid", "abc-123_X", true},
		{"replaced when invalid", "bad id\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/id", nil)
			if tc.header != "" {
				request.Header.Set(RequestIDHeader, tc.header)
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)

			got := recorder.Header().Get(RequestIDHeader)
			if got == "" || got != recorder.Body.String() {
				t.Fatalf("header %q and ctx %q should match and be non-empty", got, recorder.Body.String())
			}
			if tc.keep && got != tc.header {
				t.Fatalf("want %q, got %q", tc.header, got)
			}
			if !tc.keep && got == tc.header {
				t.Fatalf("invalid header should be replaced, got %q", got)
			}
		})
	}
}
