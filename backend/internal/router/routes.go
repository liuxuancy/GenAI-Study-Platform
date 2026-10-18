// 路由表：所有接口都在这里登记，按模块分组。新增接口时在对应模块的分组里加一行。

package router

import (
	"github.com/gin-gonic/gin"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/identity"
	"caveclass/internal/modules/account"
	"caveclass/internal/modules/system"
	"caveclass/internal/platform/resp"
)

// registerRoutes api 是 /api 分组，下面的路径都会自动带上 /api 前缀。
//
// 每一行的写法：方法、路径、所需权限（中间件）、handler。所需权限三选一：
//   - 不写：公开接口，必须同时登记在 identity/public.go 的白名单里
//   - access.RequireLogin()：只要求登录
//   - access.RequirePermission("使用方:资源:动作")：要求持有该权限码
func registerRoutes(api *gin.RouterGroup, deps Deps) {
	// 认证：白名单以外的请求都要先校验 token，必须在登记任何接口之前挂上
	api.Use(identity.Authenticate(deps.TokenIssuer, deps.Cache))

	// 框架
	api.GET("/health", resp.ToGinHandler(healthHandler(deps.DB))) // 公开

	// account 模块 /api/account/*：登录、退出、me、个人信息、改密码
	accountHandler := account.NewHandler(account.NewService(deps.DB, deps.TokenIssuer, deps.Cache))
	accountRoutes := api.Group("/account")
	accountRoutes.POST("/login", resp.ToGinHandler(accountHandler.Login)) // 公开
	accountRoutes.POST("/logout", access.RequireLogin(), resp.ToGinHandler(accountHandler.Logout))
	accountRoutes.GET("/me", access.RequireLogin(), resp.ToGinHandler(accountHandler.GetCurrentUser))
	accountRoutes.PUT("/profile", access.RequireLogin(), resp.ToGinHandler(accountHandler.UpdateProfile))
	accountRoutes.PUT("/password", access.RequireLogin(), resp.ToGinHandler(accountHandler.ChangePassword))

	// system 模块 /api/system/*：用户管理、角色权限，只有管理员使用
	systemHandler := system.NewHandler(
		system.NewUserService(deps.DB, deps.Cache, deps.UserDeletionGuards),
		system.NewRoleService(deps.DB, deps.Cache, deps.Catalog),
	)
	systemRoutes := api.Group("/system")
	systemRoutes.GET("/users", access.RequirePermission(system.PermissionUserView), resp.ToGinHandler(systemHandler.GetUserList))
	systemRoutes.POST("/users", access.RequirePermission(system.PermissionUserCreate), resp.ToGinHandler(systemHandler.CreateUser))
	systemRoutes.PUT("/users/:id", access.RequirePermission(system.PermissionUserUpdate), resp.ToGinHandler(systemHandler.UpdateUser))
	systemRoutes.PUT("/users/:id/password", access.RequirePermission(system.PermissionUserResetPassword), resp.ToGinHandler(systemHandler.ResetUserPassword))
	systemRoutes.DELETE("/users/:id", access.RequirePermission(system.PermissionUserDelete), resp.ToGinHandler(systemHandler.DeleteUser))
	systemRoutes.GET("/roles", access.RequirePermission(system.PermissionRoleView), resp.ToGinHandler(systemHandler.GetRoleList))
	systemRoutes.PUT("/roles/:code/permissions", access.RequirePermission(system.PermissionRoleAssign), resp.ToGinHandler(systemHandler.AssignRolePermissions))
	systemRoutes.GET("/permissions", access.RequirePermission(system.PermissionCatalogList), resp.ToGinHandler(systemHandler.GetPermissionCatalog))

	// course 模块 /api/teacher/*（讲师侧）、/api/student/*（学生侧）（阶段 9）
}
