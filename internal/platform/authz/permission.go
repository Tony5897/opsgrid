// Package authz is the application authorization layer (IMPLEMENTATION_PLAN §4.4).
// Role names are never compared in business code — call Can with a Permission.
package authz

// Permission is a stable capability string. UI hiding uses the same strings
// via packages/api-client/permissions.json; the server always re-checks.
type Permission string

// Catalog of permissions used by OpsGrid. Wildcards in the architecture
// overview are expanded here into concrete strings.
const (
	PermissionOrganizationRead Permission = "organization.read"
	PermissionMemberRead       Permission = "member.read"
	PermissionMemberManage     Permission = "member.manage"

	PermissionCustomerRead         Permission = "customer.read"
	PermissionCustomerReadAssigned Permission = "customer.read_assigned"
	PermissionCustomerCreate       Permission = "customer.create"
	PermissionCustomerUpdate       Permission = "customer.update"
	PermissionCustomerDelete       Permission = "customer.delete"

	PermissionAssetRead         Permission = "asset.read"
	PermissionAssetReadAssigned Permission = "asset.read_assigned"
	PermissionAssetCreate       Permission = "asset.create"
	PermissionAssetUpdate       Permission = "asset.update"
	PermissionAssetDelete       Permission = "asset.delete"

	PermissionWorkOrderRead              Permission = "work_order.read"
	PermissionWorkOrderReadAssigned      Permission = "work_order.read_assigned"
	PermissionWorkOrderCreate            Permission = "work_order.create"
	PermissionWorkOrderUpdate            Permission = "work_order.update"
	PermissionWorkOrderAssign            Permission = "work_order.assign"
	PermissionWorkOrderStartAssigned     Permission = "work_order.start_assigned"
	PermissionWorkOrderCompleteAssigned  Permission = "work_order.complete_assigned"
	PermissionWorkOrderNoteAssigned      Permission = "work_order.note_assigned"

	PermissionScheduleRead   Permission = "schedule.read"
	PermissionScheduleManage Permission = "schedule.manage"

	PermissionAuditRead  Permission = "audit.read"
	PermissionReportRead Permission = "report.read"

	PermissionAttachmentUploadAssigned Permission = "attachment.upload_assigned"

	// PermissionPlatformOperate gates Platform Health / DLQ redrive
	// (IMPLEMENTATION_PLAN §4.9). Forward-looking; not listed in overview §6.
	PermissionPlatformOperate Permission = "platform.operate"
)

// AllPermissions is every concrete permission (for owner expansion and docs).
func AllPermissions() []Permission {
	return []Permission{
		PermissionOrganizationRead,
		PermissionMemberRead,
		PermissionMemberManage,
		PermissionCustomerRead,
		PermissionCustomerReadAssigned,
		PermissionCustomerCreate,
		PermissionCustomerUpdate,
		PermissionCustomerDelete,
		PermissionAssetRead,
		PermissionAssetReadAssigned,
		PermissionAssetCreate,
		PermissionAssetUpdate,
		PermissionAssetDelete,
		PermissionWorkOrderRead,
		PermissionWorkOrderReadAssigned,
		PermissionWorkOrderCreate,
		PermissionWorkOrderUpdate,
		PermissionWorkOrderAssign,
		PermissionWorkOrderStartAssigned,
		PermissionWorkOrderCompleteAssigned,
		PermissionWorkOrderNoteAssigned,
		PermissionScheduleRead,
		PermissionScheduleManage,
		PermissionAuditRead,
		PermissionReportRead,
		PermissionAttachmentUploadAssigned,
		PermissionPlatformOperate,
	}
}
