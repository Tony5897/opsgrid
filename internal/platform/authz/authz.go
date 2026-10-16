package authz

import (
	"context"

	"github.com/google/uuid"
)

// Resource is an optional object under authorization. G1 only needs org-level
// checks (resource may be nil). *_assigned permissions require a ResourceLoader
// in later gates; until then those permissions deny when a relationship check
// would be required.
type Resource interface {
	OrganizationID() uuid.UUID
}

// Principal is the caller in an organization context after membership re-read.
type Principal struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           Role
}

// Authorizer decides whether a principal may exercise a permission.
type Authorizer struct{}

// New returns an Authorizer backed by the static role map.
func New() *Authorizer { return &Authorizer{} }

// Can reports whether p may exercise perm on resource.
// Owner always succeeds (allow-all). Role names are never compared by callers —
// only Permission values are.
//
// Assigned-scoped permissions (suffix _assigned / upload_assigned) return false
// when resource is nil; a ResourceLoader arrives in G2 with assignments.
func (a *Authorizer) Can(_ context.Context, p Principal, perm Permission, resource Resource) bool {
	if p.Role == RoleOwner {
		return true
	}
	if isAssignedScope(perm) && resource == nil {
		// Relationship check not available yet; deny closed.
		return false
	}
	if resource != nil && resource.OrganizationID() != p.OrganizationID {
		return false
	}
	for _, have := range PermissionsFor(p.Role) {
		if have == perm {
			return true
		}
	}
	return false
}

var assignedScope = map[Permission]struct{}{
	PermissionCustomerReadAssigned:      {},
	PermissionAssetReadAssigned:         {},
	PermissionWorkOrderReadAssigned:     {},
	PermissionWorkOrderStartAssigned:    {},
	PermissionWorkOrderCompleteAssigned: {},
	PermissionWorkOrderNoteAssigned:     {},
	PermissionAttachmentUploadAssigned:  {},
}

func isAssignedScope(perm Permission) bool {
	_, ok := assignedScope[perm]
	return ok
}
