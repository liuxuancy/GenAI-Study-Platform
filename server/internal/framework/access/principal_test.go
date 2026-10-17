package access

import "testing"

func TestHasPermissionAdminBypass(t *testing.T) {
	admin := &Principal{Role: RoleAdmin} // 没有任何权限码
	for _, code := range []string{"system:user:view", "teacher:course:delete", "anything:at:all"} {
		if !admin.HasPermission(code) {
			t.Errorf("admin.HasPermission(%q) = false, want true", code)
		}
	}
}

func TestHasPermissionUsesWildcardGrants(t *testing.T) {
	teacher := &Principal{Role: RoleTeacher, Permissions: []string{"teacher:*"}}
	if !teacher.HasPermission("teacher:course:update") || !teacher.HasPermission("teacher:lesson:delete") {
		t.Error("teacher with teacher:* should manage courses and lessons")
	}
	if teacher.HasPermission("student:lesson:view") || teacher.HasPermission("system:user:view") {
		t.Error("teacher must not get student or system permissions")
	}
}

func TestHasPermissionAnyGrantMatches(t *testing.T) {
	student := &Principal{Role: RoleStudent, Permissions: []string{"student:catalog:view", "student:lesson:view"}}
	if !student.HasPermission("student:lesson:view") {
		t.Error("second grant should match")
	}
	if student.HasPermission("student:course:enroll") {
		t.Error("ungranted permission should be denied")
	}
	nobody := &Principal{Role: RoleStudent}
	if nobody.HasPermission("student:catalog:view") {
		t.Error("principal without grants should be denied")
	}
}

func TestRoles(t *testing.T) {
	if len(Roles()) != 3 {
		t.Fatalf("Roles() = %+v", Roles())
	}
	for _, code := range []string{RoleAdmin, RoleTeacher, RoleStudent} {
		if !IsValidRole(code) {
			t.Errorf("IsValidRole(%q) = false", code)
		}
	}
	if IsValidRole("ROLE_GUEST") || IsValidRole("") {
		t.Error("unknown role accepted")
	}
	// 修改返回值不能影响内部列表
	Roles()[0].Name = "changed"
	if Roles()[0].Name == "changed" {
		t.Error("Roles() must return a copy")
	}
}
