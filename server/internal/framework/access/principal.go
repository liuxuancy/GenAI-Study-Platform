// 三个固定角色，以及当前请求的身份 Principal。

package access

// 三个固定角色，用户创建后角色不可修改。
const (
	RoleAdmin   = "ROLE_ADMIN"
	RoleTeacher = "ROLE_TEACHER"
	RoleStudent = "ROLE_STUDENT"
)

type RoleDef struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var roles = []RoleDef{
	{Code: RoleAdmin, Name: "管理员"},
	{Code: RoleTeacher, Name: "讲师"},
	{Code: RoleStudent, Name: "学生"},
}

// GetRoles 返回全部角色（副本）。
func GetRoles() []RoleDef {
	return append([]RoleDef(nil), roles...)
}

// GetRoleName 返回角色的显示名，未知角色返回角色码本身。
func GetRoleName(code string) string {
	for _, role := range roles {
		if role.Code == code {
			return role.Name
		}
	}
	return code
}

func IsValidRole(code string) bool {
	for _, role := range roles {
		if role.Code == code {
			return true
		}
	}
	return false
}

// Principal 当前请求用户的身份和权限快照。
// 由缓存共享给多个请求，调用方只读，不要修改。
type Principal struct {
	UserID       int64
	Username     string
	Role         string
	Active       bool // false 表示已禁用
	TokenVersion int
	Permissions  []string // 角色持有的权限码，可包含通配码
}

func (principal *Principal) IsAdmin() bool {
	return principal.Role == RoleAdmin
}

// HasPermission 判断是否拥有 required 权限：管理员直接放行，否则任一授权码匹配即可。
func (principal *Principal) HasPermission(required string) bool {
	if principal.IsAdmin() {
		return true
	}
	for _, granted := range principal.Permissions {
		if Match(granted, required) {
			return true
		}
	}
	return false
}
