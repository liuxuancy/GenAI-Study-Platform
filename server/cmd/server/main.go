// LearnHub API 服务入口。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"learnhub/internal/config"
	"learnhub/internal/db"
	"learnhub/internal/router"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogger(cfg)
	if cfg.UsesDevSecret() {
		slog.Warn("JWT_SECRET 未设置，正在使用内置开发密钥，仅限本地开发")
	}

	// 建立数据库连接
	gdb, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(gdb); err != nil {
			slog.Error("close db", "err", err)
		}
	}()

	gin.SetMode(gin.ReleaseMode) // 请求日志由我们自己的 Logger 输出
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router.New(router.Deps{DB: gdb}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	// 监听退出信号，os.Interrupt（Ctrl+C），syscall.SIGTERM（容器退出）
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 启动后台服务监听
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", cfg.HTTPAddr, "env", cfg.Env, "db", cfg.DBPath)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		// 第一种情况，服务自己退出
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		// 第二种情况，收到退出信号
	}

	// 收到 Ctrl+C / SIGTERM：最多等 5 秒让进行中的请求处理完
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func setupLogger(cfg config.Config) {
	var handler slog.Handler
	if cfg.IsDev() {
		// dev 输出易读的文本日志
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		// prod 输出 JSON 便于采集
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	slog.SetDefault(slog.New(handler))
}
