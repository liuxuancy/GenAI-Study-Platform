// 错误处理 ErrorHandler：统一渲染错误响应、兜住 panic。

package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/liuxuancy/Libs/logs"

	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/resp"
)

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
			// 只有 5xx 记错误日志，4xx 由 ginlogs.AccessLog 的访问日志记下即可
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
	// 记录错误日志，堆栈很长，单独作为字段输出，不拼进 msg
	slog.ErrorContext(ctx.Request.Context(), fmt.Sprintf("panic recovered: %s %s, panic: %v",
		ctx.Request.Method, ctx.Request.URL.Path, recovered),
		"stack", string(debug.Stack()))
	if !ctx.Writer.Written() {
		resp.Fail(ctx, errs.ErrInternal)
	}
	// 停止该请求后续handler处理
	ctx.Abort()
}
