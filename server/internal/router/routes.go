// 路由表：所有接口都在这里登记，按模块分组。新增接口时在对应模块的分组里加一行。

package router

import (
	"github.com/gin-gonic/gin"

	"learnhub/internal/platform/resp"
)

// registerRoutes api 是 /api 分组，下面的路径都会自动带上 /api 前缀。
//
// 每一行的写法：方法、路径、所需权限（中间件）、handler。例如
//
//	system.GET("/users", access.RequirePermission("system:user:view"), resp.ToGinHandler(systemHandler.ListUsers))
func registerRoutes(api *gin.RouterGroup, deps Deps) {
	// 框架
	api.GET("/health", resp.ToGinHandler(healthHandler(deps.DB)))

	// account 模块 /api/account/*：登录、退出、me、个人信息、改密码（阶段 4）

	// system 模块 /api/system/*：用户管理、角色权限（阶段 5）

	// course 模块 /api/teacher/*（讲师侧）、/api/student/*（学生侧）（阶段 9）
}
