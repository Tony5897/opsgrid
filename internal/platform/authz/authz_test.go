package authz_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/Tony5897/opsgrid/internal/platform/authz"
)

func TestOwnerHasPlatformOperate(t *testing.T) {
	t.Parallel()
	a := authz.New()
	p := authz.Principal{UserID: uuid.New(), OrganizationID: uuid.New(), Role: authz.RoleOwner}
	if !a.Can(context.Background(), p, authz.PermissionPlatformOperate, nil) {
		t.Fatal("owner must have platform.operate")
	}
}

func TestViewerLacksPlatformOperate(t *testing.T) {
	t.Parallel()
	a := authz.New()
	p := authz.Principal{UserID: uuid.New(), OrganizationID: uuid.New(), Role: authz.RoleViewer}
	if a.Can(context.Background(), p, authz.PermissionPlatformOperate, nil) {
		t.Fatal("viewer must not have platform.operate")
	}
}

func TestAssignedDeniesWithoutResource(t *testing.T) {
	t.Parallel()
	a := authz.New()
	p := authz.Principal{UserID: uuid.New(), OrganizationID: uuid.New(), Role: authz.RoleTechnician}
	if a.Can(context.Background(), p, authz.PermissionWorkOrderReadAssigned, nil) {
		t.Fatal("*_assigned must deny closed when resource is nil (G1)")
	}
}

func TestDispatcherCanAssignWorkOrder(t *testing.T) {
	t.Parallel()
	a := authz.New()
	p := authz.Principal{UserID: uuid.New(), OrganizationID: uuid.New(), Role: authz.RoleDispatcher}
	if !a.Can(context.Background(), p, authz.PermissionWorkOrderAssign, nil) {
		t.Fatal("dispatcher must work_order.assign")
	}
	if a.Can(context.Background(), p, authz.PermissionMemberManage, nil) {
		t.Fatal("dispatcher must not member.manage")
	}
}

func TestRolePermissionMatrix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		role authz.Role
		perm authz.Permission
		want bool
	}{
		{authz.RoleOwner, authz.PermissionAuditRead, true},
		{authz.RoleAdmin, authz.PermissionMemberManage, true},
		{authz.RoleAdmin, authz.PermissionPlatformOperate, false},
		{authz.RoleDispatcher, authz.PermissionScheduleManage, true},
		{authz.RoleDispatcher, authz.PermissionAuditRead, false},
		{authz.RoleTechnician, authz.PermissionWorkOrderCreate, false},
		{authz.RoleViewer, authz.PermissionWorkOrderRead, true},
		{authz.RoleViewer, authz.PermissionWorkOrderCreate, false},
	}
	a := authz.New()
	for _, tc := range cases {
		p := authz.Principal{UserID: uuid.New(), OrganizationID: uuid.New(), Role: tc.role}
		got := a.Can(context.Background(), p, tc.perm, nil)
		if got != tc.want {
			t.Errorf("role=%s perm=%s: got %v want %v", tc.role, tc.perm, got, tc.want)
		}
	}
}

func TestParseRole(t *testing.T) {
	t.Parallel()
	if _, ok := authz.ParseRole("admin"); !ok {
		t.Fatal("admin should parse")
	}
	if _, ok := authz.ParseRole("nope"); ok {
		t.Fatal("invalid role must fail")
	}
}

func TestOwnerPermissionsCoverCatalog(t *testing.T) {
	t.Parallel()
	got := authz.PermissionsFor(authz.RoleOwner)
	all := authz.AllPermissions()
	if len(got) != len(all) {
		t.Fatalf("owner perms %d != catalog %d", len(got), len(all))
	}
}
