// 认证中间件：校验 token，读取权限缓存，检查账号状态和 token 版本，把当前用户写入请求。

package identity

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/logs"
)

const bearerPrefix = "Bearer "

// Authenticate 挂在 /api 分组上：白名单以外的请求都必须带有效 token。
// 全局只初始化一次：启动时在路由表里注册。
func Authenticate(issuer *TokenIssuer, cache *access.Cache) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if IsPublicRoute(ctx.Request.Method, ctx.Request.URL.Path) {
			ctx.Next()
			return
		}
		principal, err := authenticateRequest(ctx, issuer, cache)
		if err != nil {
			_ = ctx.Error(err)
			ctx.Abort()
			return
		}
		access.SetPrincipal(ctx, principal)
		// 之后这个请求打的日志都会带上 user_id
		ctx.Request = ctx.Request.WithContext(logs.WithAttrs(ctx.Request.Context(), slog.Int64("user_id", principal.UserID)))
		ctx.Next()
	}
}

// authenticateRequest 依次检查：带了 token → token 有效 → 用户存在 → 未禁用 → token 版本一致。
func authenticateRequest(ctx *gin.Context, issuer *TokenIssuer, cache *access.Cache) (*access.Principal, error) {
	token, found := strings.CutPrefix(ctx.GetHeader("Authorization"), bearerPrefix)
	if !found || token == "" {
		return nil, errs.Unauthorized("not signed in")
	}
	claims, err := issuer.Parse(token)
	if err != nil {
		return nil, errs.Unauthorized("session expired or invalid, please sign in again")
	}
	// 根据token查询用户
	principal, err := cache.Get(ctx.Request.Context(), claims.UserID)
	if err != nil {
		return nil, err // 用户不存在时 loader 已经返回 401
	}
	if !principal.Active {
		return nil, errs.Unauthorized("account is disabled")
	}
	if principal.TokenVersion != claims.TokenVersion {
		return nil, errs.Unauthorized("session is no longer valid, please sign in again") // 改过密码、被重置密码或被禁用过
	}
	return principal, nil
}
