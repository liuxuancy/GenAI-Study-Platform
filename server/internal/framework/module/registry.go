// Registry：启动时汇总所有模块，检查命名冲突、权限码归属和默认授权，任何问题都让启动失败。

package module

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"learnhub/internal/framework/access"
)

type Registry struct {
	modules       []Module
	catalog       *access.Catalog
	defaultGrants map[string][]string // 角色 → 所有模块的默认授权合并结果
}

// NewRegistry 按顺序汇总模块。出错说明某个模块写错了，应在启动时暴露。
func NewRegistry(modules ...Module) (*Registry, error) {
	if err := checkNamesAndNamespaces(modules); err != nil {
		return nil, err
	}

	// 收集权限码，检查各权限码是否属于本模块
	var definitions []access.PermissionDef
	for _, current := range modules {
		for _, definition := range current.Permissions() {
			if !slices.Contains(current.Namespaces(), access.Namespace(definition.Code)) {
				return nil, fmt.Errorf("module %s: permission %q is outside its namespaces %v",
					current.Name(), definition.Code, current.Namespaces())
			}
			definitions = append(definitions, definition)
		}
	}
	// 校验权限码本身的正确性和唯一性
	catalog, err := access.NewCatalog(definitions)
	if err != nil {
		return nil, err
	}
	// 把所有模块的授权按角色进行合并
	defaultGrants, err := collectDefaultGrants(modules, catalog)
	if err != nil {
		return nil, err
	}
	return &Registry{modules: modules, catalog: catalog, defaultGrants: defaultGrants}, nil
}

// checkNamesAndNamespaces 模块名和使用方都必须全局唯一，使用方只能是一个不含 ':' 和 '*' 的段。
func checkNamesAndNamespaces(modules []Module) error {
	moduleByName := make(map[string]bool)
	ownerByNamespace := make(map[string]string)
	for _, current := range modules {
		name := current.Name()
		if name == "" {
			return fmt.Errorf("module name is empty")
		}
		// 校验重复
		if moduleByName[name] {
			return fmt.Errorf("duplicate module name %q", name)
		}
		moduleByName[name] = true

		for _, namespace := range current.Namespaces() {
			if namespace == "" || strings.ContainsAny(namespace, ":*") {
				return fmt.Errorf("module %s: invalid namespace %q", name, namespace)
			}
			if owner, taken := ownerByNamespace[namespace]; taken {
				return fmt.Errorf("namespace %q is claimed by both %s and %s", namespace, owner, name)
			}
			ownerByNamespace[namespace] = name
		}
	}
	return nil
}

// collectDefaultGrants 校验并合并各模块的默认授权。
func collectDefaultGrants(modules []Module, catalog *access.Catalog) (map[string][]string, error) {
	merged := make(map[string][]string)
	for _, current := range modules {
		for role, codes := range current.DefaultGrants() {
			if !access.IsValidRole(role) {
				return nil, fmt.Errorf("module %s: unknown role %q in default grants", current.Name(), role)
			}
			// 管理员直接放行，不做任何校验/授权
			if role == access.RoleAdmin {
				return nil, fmt.Errorf("module %s: %s needs no grants", current.Name(), access.RoleAdmin)
			}
			for _, code := range codes {
				if !slices.Contains(current.Namespaces(), access.Namespace(code)) {
					return nil, fmt.Errorf("module %s: cannot grant %q outside its namespaces", current.Name(), code)
				}
				if err := catalog.ValidateGrant(code); err != nil {
					return nil, fmt.Errorf("module %s: default grant %q: %w", current.Name(), code, err)
				}
				if !slices.Contains(merged[role], code) {
					merged[role] = append(merged[role], code)
				}
			}
		}
	}
	return merged, nil
}

func (registry *Registry) Modules() []Module { return registry.modules }

func (registry *Registry) Catalog() *access.Catalog { return registry.catalog }

// DefaultGrants 返回合并后的默认授权（副本）。
func (registry *Registry) DefaultGrants() map[string][]string {
	copied := make(map[string][]string, len(registry.defaultGrants))
	for role, codes := range registry.defaultGrants {
		copied[role] = slices.Clone(codes)
	}
	return copied
}

// Models 返回所有模块需要建表的结构体。
func (registry *Registry) Models() []any {
	var models []any
	for _, current := range registry.modules {
		models = append(models, current.Models()...)
	}
	return models
}

// Register 依次注册各模块的路由。
func (registry *Registry) Register(api *gin.RouterGroup, deps Deps) {
	for _, current := range registry.modules {
		current.Register(api, deps)
	}
}
