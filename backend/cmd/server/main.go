// CaveClass API 服务入口。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caveclass/internal/ai"
	"caveclass/internal/framework/access"
	"caveclass/internal/framework/identity"
	"caveclass/internal/framework/model"
	"caveclass/internal/framework/module"
	"caveclass/internal/framework/seed"
	"caveclass/internal/modules/account"
	"caveclass/internal/modules/course"
	"caveclass/internal/modules/system"
	"caveclass/internal/platform/config"
	"caveclass/internal/platform/db"
	"caveclass/internal/platform/logs"
	"caveclass/internal/router"
)

// getEnabledModules 返回要加载的业务模块，新增模块时在这里加一行。
func getEnabledModules() []module.Module {
	return []module.Module{
		account.Module{},
		system.Module{},
		course.Module{},
	}
}

func main() {
	if err := run(); err != nil {
		logs.CtxError(context.Background(), "server exited: %v", err)
		os.Exit(1)
	}
}

func run() error {
	resetDemo := flag.Bool("reset-demo", false, "删除本地数据库，重新写入种子数据和演示数据后启动（仅 dev）")
	flag.Parse()

	if err := loadDotEnv(); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogger(cfg)
	if cfg.UsesDevSecret() {
		logs.CtxWarn(context.Background(), "JWT_SECRET 未设置，正在使用内置开发密钥，仅限本地开发")
	}

	// 汇总模块：权限码、默认授权、数据表有任何冲突都在这里失败
	registry, err := module.NewRegistry(getEnabledModules()...)
	if err != nil {
		return fmt.Errorf("register modules: %w", err)
	}

	if *resetDemo {
		if !cfg.IsDev() {
			return errors.New("--reset-demo is only allowed when APP_ENV=dev")
		}
		if err := db.RemoveFiles(cfg.DBPath); err != nil {
			return err
		}
		logs.CtxWarn(context.Background(), "本地数据库已删除，将重新写入种子数据和演示数据：%s", cfg.DBPath)
	}

	// 建立数据库连接
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(database); err != nil {
			logs.CtxError(context.Background(), "close db: %v", err)
		}
	}()

	if err := prepareDatabase(database, registry, *resetDemo); err != nil {
		return err
	}

	aiService, err := ai.NewFromEnv()
	if err != nil {
		return fmt.Errorf("ai: %w", err)
	}

	gin.SetMode(gin.ReleaseMode) // 请求日志由我们自己的 Logger 输出
	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: router.NewHTTPHandler(router.Deps{
			DB:                 database,
			Catalog:            registry.GetCatalog(),
			Cache:              access.NewCache(cfg.PermissionCacheTTL, access.NewDBLoader(database)),
			TokenIssuer:        identity.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL),
			UserDeletionGuards: registry.GetUserDeletionGuards(),
			AI:                 aiService,
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	// 监听退出信号，os.Interrupt（Ctrl+C），syscall.SIGTERM（容器退出）
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 启动后台服务监听
	serverErr := make(chan error, 1)
	go func() {
		logs.CtxInfo(context.Background(), "server listening: addr=%s env=%s db=%s", cfg.HTTPAddr, cfg.Env, cfg.DBPath)
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
	logs.CtxInfo(context.Background(), "shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// prepareDatabase 建表、写种子数据；loadDemo 为 true 时再写入演示数据。
func prepareDatabase(database *gorm.DB, registry *module.Registry, loadDemo bool) error {
	ctx := context.Background()
	models := append(model.GetFrameworkModels(), registry.GetModels()...)
	if err := db.Migrate(database, models...); err != nil {
		return err
	}
	if err := seed.Run(ctx, database, registry.GetDefaultGrants()); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	if err := course.Seed(ctx, database); err != nil {
		return fmt.Errorf("seed courses: %w", err)
	}
	if loadDemo {
		if err := seed.LoadDemo(ctx, database, registry.GetModules()); err != nil {
			return fmt.Errorf("load demo data: %w", err)
		}
	}
	return nil
}

func setupLogger(cfg config.Config) {
	var handler slog.Handler
	if cfg.IsDev() {
		// dev 输出易读的文本日志
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelDebug,
			AddSource:   true,
			ReplaceAttr: logs.ShortSource,
		})
	} else {
		// prod 输出 JSON 便于采集
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			AddSource:   true,
			ReplaceAttr: logs.ShortSource,
		})
	}
	// 包一层 ContextHandler，自动带出 ctx 里的 request_id 等字段
	logs.Setup(handler)
}
