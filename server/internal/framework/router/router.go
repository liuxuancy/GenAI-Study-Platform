// Package router 组装中间件和所有路由。
package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"learnhub/internal/framework/module"
	"learnhub/internal/platform/errs"
	"learnhub/internal/platform/middleware"
	"learnhub/internal/platform/resp"
)

type Deps struct {
	DB       *gorm.DB
	Registry *module.Registry
}

func New(deps Deps) *gin.Engine {
	engine := gin.New()
	_ = engine.SetTrustedProxies(nil) // 本地部署，不信任代理头，直接用连接地址作为客户端 IP

	// 顺序有讲究：RequestID 最先，后续日志才带 request_id；Logger 在 ErrorHandler 外层才能记到最终状态
	engine.Use(middleware.RequestID(), middleware.Logger(), middleware.ErrorHandler())
	engine.NoRoute(resp.H(notFoundHandler))

	api := engine.Group("/api")
	api.GET("/health", resp.H(healthHandler(deps.DB)))

	// 各业务模块自己注册路由
	deps.Registry.Register(api, module.Deps{DB: deps.DB, Catalog: deps.Registry.Catalog()})

	return engine
}

func notFoundHandler(*gin.Context) (any, error) {
	return nil, errs.NotFound("接口不存在")
}

// healthHandler 顺带检查数据库是否可用。
func healthHandler(database *gorm.DB) resp.HandlerFunc {
	return func(ctx *gin.Context) (any, error) {
		if err := database.WithContext(ctx.Request.Context()).Exec("SELECT 1").Error; err != nil {
			return nil, errs.Internal(err)
		}
		return gin.H{"status": "ok"}, nil
	}
}
