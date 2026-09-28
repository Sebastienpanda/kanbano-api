// Package permissions holds the role rules of Kanbano, as pure functions of
// the caller's roles. It never touches the database: the roles are resolved
// once per request (repository.RoleRepository.ResolveAccess), then every
// handler asks these functions what the caller may do.
package permissions

// Organisation roles.
const (
	Owner  = "owner"
	Admin  = "admin"
	Member = "member"
)

// Workspace and task roles.
const (
	View = "view"
	Edit = "edit"
)

// Access is the caller's roles on one workspace, and optionally one task of
// it. An empty string means "no role".
type Access struct {
	// OrgRole is the caller's role in the workspace's organisation (Owner,
	// Admin or Member) when they have access to the workspace as a member,
	// "" otherwise (outsider, guest, or member it is not shared with).
	OrgRole string
	// WorkspaceRole is View or Edit when the workspace is shared with the
	// caller (always Edit for the owner), "" otherwise.
	WorkspaceRole string
	// GuestRole is the caller's guest role (View or Edit) on the requested
	// task, "" if they are not a guest on it or no task was requested.
	GuestRole string
	// IsGuest reports whether the caller is a guest on at least one task of
	// the workspace.
	IsGuest bool
}

// CanSeeWorkspace reports whether the workspace exists for the caller: as a
// member with access, or as a guest on one of its tasks. Otherwise the
// workspace is answered with a 404.
func CanSeeWorkspace(a Access) bool {
	return CanViewWorkspace(a) || a.IsGuest
}

// CanViewWorkspace reports whether the caller sees the whole board (columns
// and every task). A guest only sees the tasks shared with them.
func CanViewWorkspace(a Access) bool {
	return a.OrgRole == Owner || a.WorkspaceRole != ""
}

// CanEditWorkspace reports whether the caller may rename the workspace and,
// more generally, manage it: the owner, or an admin it is shared with,
// whatever their View/Edit role on it.
func CanEditWorkspace(a Access) bool {
	return a.OrgRole == Owner || (a.OrgRole == Admin && a.WorkspaceRole != "")
}

// CanCreateWorkspace reports whether a user whose role in the organisation
// is orgRole may create a workspace in it.
func CanCreateWorkspace(orgRole string) bool {
	return orgRole == Owner
}

// CanDeleteWorkspace reports whether the caller may delete the workspace.
func CanDeleteWorkspace(a Access) bool {
	return a.OrgRole == Owner
}

// CanCreateOrDeleteContent reports whether the caller may create or delete
// columns and tasks.
func CanCreateOrDeleteContent(a Access) bool {
	return CanEditWorkspace(a)
}

// CanEditContent reports whether the caller may modify and reorder columns
// and tasks.
func CanEditContent(a Access) bool {
	return CanEditWorkspace(a) || a.WorkspaceRole == Edit
}

// CanManageWorkspaceAccess reports whether the caller may share the
// workspace with a user whose organisation role is targetOrgRole, change
// their View/Edit role or remove their access. Nobody touches the owner; an
// admin only manages members.
func CanManageWorkspaceAccess(a Access, targetOrgRole string) bool {
	if targetOrgRole == Owner {
		return false
	}
	if a.OrgRole == Owner {
		return true
	}
	return CanEditWorkspace(a) && targetOrgRole == Member
}

// CanInviteGuest reports whether the caller may invite guests on the tasks
// of the workspace, change their role or remove them.
func CanInviteGuest(a Access) bool {
	return CanEditWorkspace(a)
}

// CanAssign reports whether the caller may assign someone to a task or
// unassign them. An assignment only shows who works on the task.
func CanAssign(a Access) bool {
	return CanEditContent(a)
}

// CanManageOrg reports whether the caller may rename or delete the
// organisation, invite into it, change a role in it or remove someone from
// it.
func CanManageOrg(orgRole string) bool {
	return orgRole == Owner
}

// CanViewMemberAccess reports whether the caller may see another member's
// access to the workspaces of the organisation. An admin only sees it on
// the workspaces shared with them.
func CanViewMemberAccess(orgRole string) bool {
	return orgRole == Owner || orgRole == Admin
}

// CanViewTask reports whether the caller sees the task requested in a.
func CanViewTask(a Access) bool {
	return CanViewWorkspace(a) || a.GuestRole != ""
}

// CanEditTaskDescription reports whether the caller may change the
// description of the task requested in a. It is the only field a guest
// with Edit may change.
func CanEditTaskDescription(a Access) bool {
	return CanEditContent(a) || a.GuestRole == Edit
}
