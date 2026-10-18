// Registry：启动时汇总所有模块，检查命名冲突、权限码归属和默认授权，任何问题都让启动失败。

package module

import (
	"fmt"
	"slices"
	"strings"

	"caveclass/internal/framework/access"
)

type Registry struct {
	modules       []Module
	catalog       *access.Catalog
	defaultGrants map[string][]string // 角色 → 所有模块的默认授权合并结果
}

// NewRegistry 按顺序汇总模块。出错说明某个模块写错了，应在启动时暴露。
// 全局只初始化一次：启动时由 main.go 调用（测试里每个用例各建一个）。
func NewRegistry(modules ...Module) (*Registry, error) {
	if err := checkNamesAndNamespaces(modules); err != nil {
		return nil, err
	}

	// 收集权限码，检查各权限码是否属于本模块
	var definitions []access.PermissionDef
	for _, current := range modules {
		for _, definition := range current.GetPermissions() {
			if !slices.Contains(current.GetNamespaces(), access.GetNamespace(definition.Code)) {
				return nil, fmt.Errorf("module %s: permission %q is outside its namespaces %v",
					current.GetName(), definition.Code, current.GetNamespaces())
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
		name := current.GetName()
		if name == "" {
			return fmt.Errorf("module name is empty")
		}
		// 校验重复
		if moduleByName[name] {
			return fmt.Errorf("duplicate module name %q", name)
		}
		moduleByName[name] = true

		for _, namespace := range current.GetNamespaces() {
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
		for role, codes := range current.GetDefaultGrants() {
			if !access.IsValidRole(role) {
				return nil, fmt.Errorf("module %s: unknown role %q in default grants", current.GetName(), role)
			}
			// 管理员直接放行，不做任何校验/授权
			if role == access.RoleAdmin {
				return nil, fmt.Errorf("module %s: %s needs no grants", current.GetName(), access.RoleAdmin)
			}
			for _, code := range codes {
				if !slices.Contains(current.GetNamespaces(), access.GetNamespace(code)) {
					return nil, fmt.Errorf("module %s: cannot grant %q outside its namespaces", current.GetName(), code)
				}
				if err := catalog.ValidateGrant(code); err != nil {
					return nil, fmt.Errorf("module %s: default grant %q: %w", current.GetName(), code, err)
				}
				if !slices.Contains(merged[role], code) {
					merged[role] = append(merged[role], code)
				}
			}
		}
	}
	return merged, nil
}

func (registry *Registry) GetModules() []Module { return registry.modules }

func (registry *Registry) GetCatalog() *access.Catalog { return registry.catalog }

// GetDefaultGrants 返回合并后的默认授权（副本）。
func (registry *Registry) GetDefaultGrants() map[string][]string {
	copied := make(map[string][]string, len(registry.defaultGrants))
	for role, codes := range registry.defaultGrants {
		copied[role] = slices.Clone(codes)
	}
	return copied
}

// GetModels 返回所有模块需要建表的结构体。
func (registry *Registry) GetModels() []any {
	var models []any
	for _, current := range registry.modules {
		models = append(models, current.GetModels()...)
	}
	return models
}
