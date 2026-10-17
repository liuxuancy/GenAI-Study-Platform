// 请求 ID 中间件：沿用或生成 X-Request-ID，写入 ctx 和响应头。

package middleware

import (
	"crypto/rand"

	"github.com/gin-gonic/gin"

	"learnhub/internal/platform/logs"
)

// RequestIDHeader 请求和响应里携带请求 ID 的头。
const RequestIDHeader = "X-Request-ID"

const maxRequestIDLen = 64

// RequestID 给每个请求分配 ID 并写入 ctx 和响应头，必须放在最外层。
// 客户端传了合法的 X-Request-ID 就沿用，便于前端或网关串联日志。
func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := ctx.GetHeader(RequestIDHeader)
		if !isValidRequestID(requestID) {
			requestID = rand.Text()
		}
		ctx.Header(RequestIDHeader, requestID)
		ctx.Request = ctx.Request.WithContext(logs.WithRequestID(ctx.Request.Context(), requestID))
		ctx.Next()
	}
}

// isValidRequestID 只接受字母、数字、- 和 _，防止日志注入。
func isValidRequestID(requestID string) bool {
	if requestID == "" || len(requestID) > maxRequestIDLen {
		return false
	}
	for _, char := range requestID {
		isAlnum := (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
		if !isAlnum && char != '-' && char != '_' {
			return false
		}
	}
	return true
}
