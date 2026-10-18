// 白名单：不需要登录就能访问的接口，只在这一处配置。

package identity

// publicRoutes 键的格式为 "方法 路径"，路径带 /api 前缀。
var publicRoutes = map[string]bool{
	"GET /api/health":         true,
	"POST /api/account/login": true,
}

// IsPublicRoute 判断请求是否在白名单里。
func IsPublicRoute(method, path string) bool {
	return publicRoutes[method+" "+path]
}
