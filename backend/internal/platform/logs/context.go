// 在 ctx 中存取日志字段和请求 ID。

package logs

import (
	"context"
	"log/slog"
)

// RequestIDKey 日志中请求 ID 的字段名。
const RequestIDKey = "request_id"

type attrsKey struct{}

type requestIDKey struct{}

// WithAttrs 往 ctx 追加日志字段，之后用该 ctx 打的日志都会带上。
// 例如鉴权通过后：ctx = logs.WithAttrs(ctx, slog.Int64("user_id", uid))
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	existing := getContextAttrs(ctx)
	// 复制一份再追加，避免和父 ctx 共享底层数组
	merged := make([]slog.Attr, 0, len(existing)+len(attrs))
	merged = append(merged, existing...)
	merged = append(merged, attrs...)
	return context.WithValue(ctx, attrsKey{}, merged)
}

// WithRequestID 写入请求 ID，同时作为日志字段。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	ctx = context.WithValue(ctx, requestIDKey{}, requestID)
	return WithAttrs(ctx, slog.String(RequestIDKey, requestID))
}

// GetRequestID 取出请求 ID，没有则返回空串。
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

func getContextAttrs(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	attrs, _ := ctx.Value(attrsKey{}).([]slog.Attr)
	return attrs
}
