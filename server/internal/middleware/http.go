// 请求日志 Logger 和错误处理 ErrorHandler。

package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"

	"learnhub/internal/errs"
	"learnhub/internal/logs"
	"learnhub/internal/resp"
)

// Logger 每个请求记一条日志。必须放在最外层，才能拿到最终状态码（包括 panic 后的 500）。
func Logger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		status := ctx.Writer.Status()
		slog.LogAttrs(ctx.Request.Context(), logLevel(status), "http",
			slog.String("method", ctx.Request.Method),
			slog.String("path", ctx.Request.URL.Path),
			slog.Int("status", status),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.String("ip", ctx.ClientIP()),
		)
	}
}

// ErrorHandler 统一渲染 ctx.Error 收集到的错误，并兜住 panic， 保证任何失败都以 {code, msg, data} 返回。
func ErrorHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer recoverPanic(ctx)
		ctx.Next()

		if len(ctx.Errors) == 0 || ctx.Writer.Written() {
			return
		}
		appError := errs.From(ctx.Errors.Last().Err)
		if appError.Code >= errs.SystemInternal {
			// 只有 5xx 记错误日志，4xx 由 Logger 记 WARN 即可
			logs.CtxError(ctx.Request.Context(), "request failed: %s %s, err: %v",
				ctx.Request.Method, ctx.Request.URL.Path, appError)
		}
		resp.Fail(ctx, appError)
	}
}

// recoverPanic 必须被直接 defer 调用，recover() 才能生效。
func recoverPanic(ctx *gin.Context) {
	recovered := recover()
	if recovered == nil {
		return
	}
	if recovered == http.ErrAbortHandler {
		panic(recovered) // 主动中断连接，交还给 net/http
	}
	// 堆栈很长，单独作为字段输出，不拼进 msg
	slog.ErrorContext(ctx.Request.Context(), fmt.Sprintf("panic recovered: %s %s, panic: %v",
		ctx.Request.Method, ctx.Request.URL.Path, recovered),
		"stack", string(debug.Stack()))
	if !ctx.Writer.Written() {
		resp.Fail(ctx, errs.ErrInternal)
	}
	ctx.Abort()
}

// logLevel 按状态码选日志级别：5xx ERROR，4xx WARN，其余 INFO。
func logLevel(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
