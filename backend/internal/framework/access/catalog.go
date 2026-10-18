// 权限码定义和目录：汇总各模块声明的权限码，校验授权。

package access

import (
	"fmt"

	"caveclass/internal/platform/errs"
)

type PermissionType string

const (
	PermissionTypeMenu   PermissionType = "MENU"   // 控制页面和菜单
	PermissionTypeButton PermissionType = "BUTTON" // 控制按钮，同时守护对应接口
	PermissionTypeAPI    PermissionType = "API"    // 只守护接口，没有界面元素
)

// PermissionDef 一个权限码的定义，由各模块在代码中声明。
//
// Code 格式为 使用方:资源:动作，例如 teacher:course:create
type PermissionDef struct {
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Type        PermissionType `json:"type"`
	Description string         `json:"description,omitempty"`
}

// Catalog 所有模块权限码的汇总，按声明顺序保存。只收录具体权限码，通配码只出现在授权里。
type Catalog struct {
	definitions []PermissionDef
}

// NewCatalog 初始化权限码目录。
// 全局只初始化一次：启动时由 NewRegistry 调用。
func NewCatalog(definitions []PermissionDef) (*Catalog, error) {
	catalog := &Catalog{}
	for _, definition := range definitions {
		// 格式检查
		if err := checkDefinition(definition); err != nil {
			return nil, err
		}
		// 如果有重复Code，只能是初始化时代码写错，应该直接报错让修复，而不是通过
		if catalog.Has(definition.Code) {
			return nil, fmt.Errorf("duplicate permission code %q", definition.Code)
		}
		// 将该Code收录到目录中
		catalog.definitions = append(catalog.definitions, definition)
	}
	return catalog, nil
}

func checkDefinition(definition PermissionDef) error {
	if !ValidCode(definition.Code) {
		return fmt.Errorf("invalid permission code %q", definition.Code)
	}
	if IsWildcard(definition.Code) {
		return fmt.Errorf("permission code %q: wildcards cannot be declared, only granted", definition.Code)
	}
	if definition.Name == "" {
		return fmt.Errorf("permission code %q: name is required", definition.Code)
	}
	switch definition.Type {
	case PermissionTypeMenu, PermissionTypeButton, PermissionTypeAPI:
		return nil
	default:
		return fmt.Errorf("permission code %q: unknown type %q", definition.Code, definition.Type)
	}
}

// GetAllDefinitions 按声明顺序返回全部定义（副本）。
func (catalog *Catalog) GetAllDefinitions() []PermissionDef {
	return append([]PermissionDef(nil), catalog.definitions...)
}

// Has 判断权限码是否在目录中。权限码只有几十个，逐个比较即可。
func (catalog *Catalog) Has(code string) bool {
	for _, definition := range catalog.definitions {
		if definition.Code == code {
			return true
		}
	}
	return false
}

// ValidateGrant 检查一个权限码能否授予给角色：
// 具体码必须在目录中；通配码必须覆盖至少一个目录中的码；不允许授予全局 '*'。
func (catalog *Catalog) ValidateGrant(code string) error {
	if !ValidCode(code) {
		return errs.BadRequest(fmt.Sprintf("权限码格式非法：%s", code))
	}
	if code == wildcard {
		return errs.BadRequest("不允许授予全局通配 *")
	}
	if !IsWildcard(code) {
		if !catalog.Has(code) {
			return errs.BadRequest(fmt.Sprintf("权限码不存在：%s", code))
		}
		return nil
	}
	for _, definition := range catalog.definitions {
		if Match(code, definition.Code) {
			return nil
		}
	}
	return errs.BadRequest(fmt.Sprintf("通配码没有覆盖任何权限码：%s", code))
}
