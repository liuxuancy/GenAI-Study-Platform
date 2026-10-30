// 组装 HTTP 服务：创建 gin、挂全局中间件、配置 404，再注册 routes.go 里的路由表。

package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caveclass/internal/ai"
	"caveclass/internal/framework/access"
	"caveclass/internal/framework/identity"
	"caveclass/internal/framework/module"
	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/middleware"
	"caveclass/internal/platform/resp"
)

// Deps main.go 交给路由层的依赖，路由表用它们创建各模块的 handler。只放路由层真正用到的东西。
type Deps struct {
	DB          *gorm.DB              // 数据库连接：健康检查，以及各模块的 handler 和 service
	Catalog     *access.Catalog       // 权限码目录：system 模块返回目录、校验授权时用
	Cache       *access.Cache         // 权限缓存：认证中间件读权限；改密码、改授权后要清空
	TokenIssuer *identity.TokenIssuer // token 签发器：登录时签发，认证中间件校验

	UserDeletionGuards []module.UserDeletionGuard // 删除用户前，由其他模块检查是否还有数据引用该用户
	AI                 *ai.Service                // 整课生成。未配置时生成接口返回错误，其余接口不受影响
}

// NewHTTPHandler 全局只初始化一次，返回整个服务的 HTTP 入口（不是某个接口的业务 handler），交给 http.Server 使用。
func NewHTTPHandler(deps Deps) http.Handler {
	engine := gin.New()               // 不用 gin.Default()：日志和 panic 恢复由我们自己的中间件负责
	_ = engine.SetTrustedProxies(nil) // 本地部署，不信任代理头，直接用连接地址作为客户端 IP

	// -- 向gin中写入配置 --
	// 1.所有请求依次经过三层中间件：RequestID 最先，后续日志才带 request_id；Logger 在 ErrorHandler 外层才能记到最终状态
	engine.Use(middleware.RequestID(), middleware.Logger(), middleware.ErrorHandler())
	// 2.如果没有匹配上，报错接口不存在
	engine.NoRoute(resp.ToGinHandler(notFoundHandler))
	// 3.初始化注册全局路由表，后续使用该表匹配
	registerRoutes(engine.Group("/api"), deps)

	return engine
}

func notFoundHandler(*gin.Context) (any, error) {
	return nil, errs.NotFound("API not found")
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
