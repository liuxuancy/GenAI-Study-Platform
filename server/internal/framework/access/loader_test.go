package access_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/errs"
)

func TestDBLoaderLoadsRolePermissions(t *testing.T) {
	database := testutil.NewDB(t)
	testutil.MustExec(t, database,
		"INSERT INTO sys_role_permission (role, permission_code) VALUES (?, ?), (?, ?), (?, ?)",
		access.RoleStudent, "student:my:view",
		access.RoleStudent, "student:catalog:view",
		access.RoleTeacher, "teacher:*")

	principal, err := access.NewDBLoader(database).LoadPrincipal(context.Background(), testutil.GetUserID(t, database, "student1"))
	if err != nil {
		t.Fatal(err)
	}
	if principal.Username != "student1" || principal.Role != access.RoleStudent || !principal.Active || principal.TokenVersion != 0 {
		t.Errorf("principal = %+v", principal)
	}
	if want := []string{"student:catalog:view", "student:my:view"}; !slices.Equal(principal.Permissions, want) {
		t.Errorf("permissions = %v, want %v (only the student's own grants, sorted)", principal.Permissions, want)
	}
}

func TestDBLoaderAdminHasNoGrantRows(t *testing.T) {
	database := testutil.NewDB(t)
	principal, err := access.NewDBLoader(database).LoadPrincipal(context.Background(), testutil.GetUserID(t, database, "admin"))
	if err != nil {
		t.Fatal(err)
	}
	if !principal.IsAdmin() || principal.Permissions == nil || len(principal.Permissions) != 0 {
		t.Errorf("admin principal = %+v, want IsAdmin and an empty (non-nil) permission list", principal)
	}
}

func TestDBLoaderUnknownUserIsUnauthorized(t *testing.T) {
	database := testutil.NewDB(t)
	_, err := access.NewDBLoader(database).LoadPrincipal(context.Background(), 99999)
	if !errors.Is(err, errs.ErrUnauthorized) {
		t.Errorf("err = %v, want 401", err)
	}
}
