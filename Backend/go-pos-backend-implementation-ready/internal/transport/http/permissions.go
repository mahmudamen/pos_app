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
