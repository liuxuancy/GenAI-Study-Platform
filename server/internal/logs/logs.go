// Package logs 基于 slog 的日志封装：从 ctx 自动带出 request_id 等请求级字段。
//
// 用法：logs.CtxInfo(ctx, "输入的 id：%d", inputID)，占位符同 fmt（%s %v %t %d 等）。
package logs

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// Setup 用 ContextHandler 包装 inner 并设为全局默认 logger。
// 之后直接调用 slog.InfoContext(ctx, ...) 也会带上 ctx 字段。
func Setup(inner slog.Handler) {
	slog.SetDefault(slog.New(NewContextHandler(inner)))
}

func CtxDebug(ctx context.Context, format string, args ...any) {
	write(ctx, slog.LevelDebug, format, args...)
}

func CtxInfo(ctx context.Context, format string, args ...any) {
	write(ctx, slog.LevelInfo, format, args...)
}

func CtxWarn(ctx context.Context, format string, args ...any) {
	write(ctx, slog.LevelWarn, format, args...)
}

func CtxError(ctx context.Context, format string, args ...any) {
	write(ctx, slog.LevelError, format, args...)
}

// write 自己构造 Record，让 source 指向业务调用处而不是本包。
// 只能被 CtxXxx 直接调用，调用层级变了要同步调整 skip。
func write(ctx context.Context, level slog.Level, format string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	handler := slog.Default().Handler()
	if !handler.Enabled(ctx, level) {
		return // 级别不够就不格式化，省掉 Sprintf 开销
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // 跳过 runtime.Callers、write、CtxXxx
	record := slog.NewRecord(time.Now(), level, formatMessage(format, args...), pcs[0])
	_ = handler.Handle(ctx, record)
}

// formatMessage 没有参数时原样输出，避免 "100%" 被当成占位符。
func formatMessage(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// ShortSource 用作 HandlerOptions.ReplaceAttr，把 source 缩短为 "目录/文件.go:行号"。
func ShortSource(groups []string, attr slog.Attr) slog.Attr {
	if attr.Key != slog.SourceKey || len(groups) > 0 {
		return attr
	}
	source, ok := attr.Value.Any().(*slog.Source)
	if !ok || source == nil {
		return attr
	}
	dir := filepath.Base(filepath.Dir(source.File))
	file := filepath.Base(source.File)
	return slog.String(slog.SourceKey, dir+"/"+file+":"+strconv.Itoa(source.Line))
}
