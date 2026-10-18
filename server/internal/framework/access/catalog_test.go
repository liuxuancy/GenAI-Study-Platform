package access

import (
	"errors"
	"testing"

	"caveclass/internal/platform/errs"
)

func sampleDefinitions() []PermissionDef {
	return []PermissionDef{
		{Code: "teacher:course:view", Name: "查看课程", Type: PermissionTypeMenu},
		{Code: "teacher:course:create", Name: "新建课程", Type: PermissionTypeButton},
		{Code: "student:lesson:view", Name: "读取课时", Type: PermissionTypeAPI},
	}
}

func TestNewCatalogKeepsOrder(t *testing.T) {
	catalog, err := NewCatalog(sampleDefinitions())
	if err != nil {
		t.Fatal(err)
	}
	all := catalog.GetAllDefinitions()
	if len(all) != 3 || all[0].Code != "teacher:course:view" || all[2].Code != "student:lesson:view" {
		t.Errorf("All() = %+v", all)
	}
	if !catalog.Has("teacher:course:create") || catalog.Has("teacher:course:delete") {
		t.Error("Has() returned wrong result")
	}
}

func TestNewCatalogRejectsBadDefinitions(t *testing.T) {
	cases := map[string][]PermissionDef{
		"duplicate":      {{Code: "a:b:c", Name: "x", Type: PermissionTypeAPI}, {Code: "a:b:c", Name: "y", Type: PermissionTypeAPI}},
		"invalid code":   {{Code: "a::c", Name: "x", Type: PermissionTypeAPI}},
		"wildcard":       {{Code: "a:*", Name: "x", Type: PermissionTypeAPI}},
		"empty name":     {{Code: "a:b:c", Type: PermissionTypeAPI}},
		"unknown type":   {{Code: "a:b:c", Name: "x", Type: "PAGE"}},
		"lowercase type": {{Code: "a:b:c", Name: "x", Type: "menu"}},
	}
	for name, definitions := range cases {
		if _, err := NewCatalog(definitions); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestValidateGrant(t *testing.T) {
	catalog, err := NewCatalog(sampleDefinitions())
	if err != nil {
		t.Fatal(err)
	}
	allowed := []string{"teacher:course:view", "teacher:*", "teacher:course:*", "student:*"}
	rejected := []string{"teacher:course:delete", "system:*", "*", "teacher:*:view", "", "teacher:lesson:*"}

	for _, code := range allowed {
		if err := catalog.ValidateGrant(code); err != nil {
			t.Errorf("ValidateGrant(%q) = %v, want nil", code, err)
		}
	}
	for _, code := range rejected {
		err := catalog.ValidateGrant(code)
		if !errors.Is(err, errs.ErrBadRequest) {
			t.Errorf("ValidateGrant(%q) = %v, want BadRequest", code, err)
		}
	}
}
