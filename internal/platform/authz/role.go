package authz

// Role is a membership role stored in app.organization_memberships.
type Role string

const (
	RoleOwner       Role = "owner"
	RoleAdmin       Role = "admin"
	RoleDispatcher  Role = "dispatcher"
	RoleTechnician  Role = "technician"
	RoleViewer      Role = "viewer"
)

// ParseRole validates a DB/API role string.
func ParseRole(s string) (Role, bool) {
	switch Role(s) {
	case RoleOwner, RoleAdmin, RoleDispatcher, RoleTechnician, RoleViewer:
		return Role(s), true
	default:
		return "", false
	}
}

// rolePermissions is the static, code-reviewed map (overview §6 + platform.operate for owner).
// Owner is handled as allow-all in PermissionsFor / Can — not listed here.
var rolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermissionOrganizationRead,
		PermissionMemberRead,
		PermissionMemberManage,
		PermissionCustomerRead,
		PermissionCustomerCreate,
		PermissionCustomerUpdate,
		PermissionCustomerDelete,
		PermissionAssetRead,
		PermissionAssetCreate,
		PermissionAssetUpdate,
		PermissionAssetDelete,
		PermissionWorkOrderRead,
		PermissionWorkOrderCreate,
		PermissionWorkOrderUpdate,
		PermissionWorkOrderAssign,
		PermissionScheduleRead,
		PermissionScheduleManage,
		PermissionAuditRead,
		PermissionReportRead,
	},
	RoleDispatcher: {
		PermissionCustomerRead,
		PermissionAssetRead,
		PermissionWorkOrderRead,
		PermissionWorkOrderCreate,
		PermissionWorkOrderUpdate,
		PermissionWorkOrderAssign,
		PermissionScheduleRead,
		PermissionScheduleManage,
		PermissionReportRead,
	},
	RoleTechnician: {
		PermissionCustomerReadAssigned,
		PermissionAssetReadAssigned,
		PermissionWorkOrderReadAssigned,
		PermissionWorkOrderStartAssigned,
		PermissionWorkOrderCompleteAssigned,
		PermissionWorkOrderNoteAssigned,
		PermissionAttachmentUploadAssigned,
	},
	RoleViewer: {
		PermissionCustomerRead,
		PermissionAssetRead,
		PermissionWorkOrderRead,
		PermissionScheduleRead,
	},
}

// PermissionsFor returns the concrete permissions for a role.
// Owner receives every permission in the catalog (allow-all / "*").
func PermissionsFor(role Role) []Permission {
	if role == RoleOwner {
		return append([]Permission(nil), AllPermissions()...)
	}
	perms := rolePermissions[role]
	out := make([]Permission, len(perms))
	copy(out, perms)
	return out
}

// PermissionStrings returns permissions as plain strings for API / JSON export.
func PermissionStrings(role Role) []string {
	perms := PermissionsFor(role)
	out := make([]string, len(perms))
	for i, p := range perms {
		out[i] = string(p)
	}
	return out
}
