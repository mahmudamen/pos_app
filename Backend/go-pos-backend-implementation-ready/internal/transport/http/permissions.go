package http

// rolePermissions defines the static server-side RBAC matrix (roadmap B1).
// Client code and UI cannot bypass this authority.
var rolePermissions = map[string]map[string]struct{}{
	"owner":      {},
	"manager":    {},
	"cashier":    {},
	"saas_admin": {},
}

func init() {
	grant := func(roles []string, resource, action string) {
		perm := resource + ":" + action
		for _, r := range roles {
			if _, ok := rolePermissions[r]; !ok {
				rolePermissions[r] = map[string]struct{}{}
			}
			rolePermissions[r][perm] = struct{}{}
		}
	}
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "pos", "read")
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "pos", "open")
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "pos", "close")
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "pos", "sale")
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "pos", "discount")
	grant([]string{"owner", "manager", "saas_admin"}, "pos", "refund")
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "customers", "read")
	grant([]string{"owner", "manager", "saas_admin"}, "customers", "write")
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "catalog", "read")
	grant([]string{"owner", "manager", "saas_admin"}, "catalog", "write")
	grant([]string{"owner", "manager", "saas_admin"}, "inventory", "adjust")
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "dashboard", "read")
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "restaurant", "read")
	grant([]string{"owner", "manager", "saas_admin"}, "restaurant", "write")
	grant([]string{"saas_admin"}, "saas", "admin")
	grant([]string{"owner", "manager", "saas_admin"}, "notifications", "read")
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "community", "read")
	grant([]string{"owner", "manager", "saas_admin"}, "community", "write")
	grant([]string{"owner", "manager", "cashier", "saas_admin"}, "jobs", "read")
	grant([]string{"owner", "manager", "saas_admin"}, "jobs", "write")
	grant([]string{"owner", "manager", "saas_admin"}, "jobs", "manage")
	// National community (docs/24): any authenticated role may attempt to join /
	// invite; real gating is the community_members row checked in the handler
	// (e.g. guests can be invited but may not join unless offered a code).
	grant([]string{"owner", "manager", "cashier", "guest", "saas_admin"}, "community", "invite")
	// community.moderate is NOT in the static matrix: moderation rides on the
	// national membership role (moderator/admin) or the saas_admin org role,
	// both checked dynamically in the national handlers.
}

// HasPermission reports whether role may perform action on resource.
func HasPermission(role, resource, action string) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = perms[resource+":"+action]
	return ok
}
