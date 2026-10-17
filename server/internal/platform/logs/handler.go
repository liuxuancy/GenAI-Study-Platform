// ContextHandler：输出日志前把 ctx 里的字段加到记录上。

package logs

import (
	"context"
	"log/slog"
)

// ContextHandler 在每条日志输出前，把 ctx 里通过 WithAttrs 存的字段加到记录上。
type ContextHandler struct {
	inner slog.Handler
}

func NewContextHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{inner: inner}
}

func (handler *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return handler.inner.Enabled(ctx, level)
}

func (handler *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if attrs := attrsFrom(ctx); len(attrs) > 0 {
		record = record.Clone() // Record 可能被多个 Handler 共享，修改前先复制
		record.AddAttrs(attrs...)
	}
	return handler.inner.Handle(ctx, record)
}

func (handler *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{inner: handler.inner.WithAttrs(attrs)}
}

func (handler *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{inner: handler.inner.WithGroup(name)}
}
