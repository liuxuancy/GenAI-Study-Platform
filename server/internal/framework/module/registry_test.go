package module

import (
	"slices"
	"strings"
	"testing"

	"learnhub/internal/framework/access"
)

// fakeModule 用字段描述一个模块，方便在测试里构造各种写错的情况。
type fakeModule struct {
	name          string
	namespaces    []string
	permissions   []access.PermissionDef
	defaultGrants map[string][]string
	models        []any
}

func (fake *fakeModule) Name() string                        { return fake.name }
func (fake *fakeModule) Namespaces() []string                { return fake.namespaces }
func (fake *fakeModule) Permissions() []access.PermissionDef { return fake.permissions }
func (fake *fakeModule) DefaultGrants() map[string][]string  { return fake.defaultGrants }
func (fake *fakeModule) Models() []any                       { return fake.models }

var _ Module = (*fakeModule)(nil)

func permission(code string) access.PermissionDef {
	return access.PermissionDef{Code: code, Name: code, Type: access.PermissionTypeButton}
}

// courseModule 一个写对了的模块：同时拥有 teacher 和 student 两个使用方。
func courseModule() *fakeModule {
	return &fakeModule{
		name:       "course",
		namespaces: []string{"teacher", "student"},
		permissions: []access.PermissionDef{
			permission("teacher:course:view"),
			permission("teacher:course:create"),
			permission("student:catalog:view"),
		},
		defaultGrants: map[string][]string{
			access.RoleTeacher: {"teacher:*"},
			access.RoleStudent: {"student:catalog:view"},
		},
		models: []any{"course-table"},
	}
}

func systemModule() *fakeModule {
	return &fakeModule{
		name:        "system",
		namespaces:  []string{"system"},
		permissions: []access.PermissionDef{permission("system:user:view")},
		models:      []any{"user-table"},
	}
}

func TestNewRegistryCollectsModules(t *testing.T) {
	registry, err := NewRegistry(systemModule(), courseModule())
	if err != nil {
		t.Fatal(err)
	}

	codes := make([]string, 0)
	for _, definition := range registry.Catalog().All() {
		codes = append(codes, definition.Code)
	}
	want := []string{"system:user:view", "teacher:course:view", "teacher:course:create", "student:catalog:view"}
	if !slices.Equal(codes, want) {
		t.Errorf("catalog = %v, want %v (in module order)", codes, want)
	}

	grants := registry.DefaultGrants()
	if !slices.Equal(grants[access.RoleTeacher], []string{"teacher:*"}) ||
		!slices.Equal(grants[access.RoleStudent], []string{"student:catalog:view"}) {
		t.Errorf("default grants = %v", grants)
	}

	if models := registry.Models(); !slices.Equal(models, []any{"user-table", "course-table"}) {
		t.Errorf("models = %v", models)
	}
}

func TestNewRegistryWithoutModules(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Catalog().All()) != 0 || len(registry.DefaultGrants()) != 0 {
		t.Error("empty registry should have no permissions or grants")
	}
}

func TestDefaultGrantsReturnsCopy(t *testing.T) {
	registry, err := NewRegistry(courseModule())
	if err != nil {
		t.Fatal(err)
	}
	registry.DefaultGrants()[access.RoleTeacher][0] = "changed"
	if registry.DefaultGrants()[access.RoleTeacher][0] != "teacher:*" {
		t.Error("DefaultGrants must return a copy")
	}
}

func TestNewRegistryRejectsMistakes(t *testing.T) {
	cases := []struct {
		name    string
		modules func() []Module
		errPart string
	}{
		{"empty name", emptyName, "name is empty"},
		{"duplicate name", duplicateName, "duplicate module name"},
		{"namespace with colon", namespaceWithColon, "invalid namespace"},
		{"namespace claimed twice", namespaceClaimedTwice, "claimed by both"},
		{"permission outside namespace", permissionOutsideNamespace, "outside its namespaces"},
		{"duplicate permission", duplicatePermission, "duplicate permission code"},
		{"unknown role", unknownRole, "unknown role"},
		{"grant to admin", grantToAdmin, "needs no grants"},
		{"grant outside namespace", grantOutsideNamespace, "cannot grant"},
		{"grant not in catalog", grantNotInCatalog, "default grant"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewRegistry(testCase.modules()...)
			if err == nil || !strings.Contains(err.Error(), testCase.errPart) {
				t.Errorf("err = %v, want it to contain %q", err, testCase.errPart)
			}
		})
	}
}

func emptyName() []Module {
	broken := courseModule()
	broken.name = ""
	return []Module{broken}
}

func duplicateName() []Module {
	other := systemModule()
	other.name = "course"
	return []Module{courseModule(), other}
}

func namespaceWithColon() []Module {
	broken := systemModule()
	broken.namespaces = []string{"sys:tem"}
	return []Module{broken}
}

func namespaceClaimedTwice() []Module {
	other := systemModule()
	other.namespaces = []string{"teacher"}
	return []Module{courseModule(), other}
}

func permissionOutsideNamespace() []Module {
	broken := systemModule()
	broken.permissions = append(broken.permissions, permission("teacher:course:delete"))
	return []Module{broken}
}

func duplicatePermission() []Module {
	broken := courseModule()
	broken.permissions = append(broken.permissions, permission("teacher:course:view"))
	return []Module{broken}
}

func unknownRole() []Module {
	broken := courseModule()
	broken.defaultGrants = map[string][]string{"ROLE_GUEST": {"student:catalog:view"}}
	return []Module{broken}
}

func grantToAdmin() []Module {
	broken := courseModule()
	broken.defaultGrants = map[string][]string{access.RoleAdmin: {"teacher:*"}}
	return []Module{broken}
}

func grantOutsideNamespace() []Module {
	broken := systemModule()
	broken.defaultGrants = map[string][]string{access.RoleTeacher: {"teacher:course:view"}}
	return []Module{courseModule(), broken}
}

func grantNotInCatalog() []Module {
	broken := courseModule()
	broken.defaultGrants = map[string][]string{access.RoleStudent: {"student:course:enroll"}}
	return []Module{broken}
}
