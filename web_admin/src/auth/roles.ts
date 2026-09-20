export const SAAS_ROLES = ['saas_admin', 'superadmin']
export const STORE_ROLES = ['owner', 'manager', 'cashier']
export const MANAGER_ROLES = ['owner', 'manager']

export function isSaasAdmin(role: string | null | undefined): boolean {
  return SAAS_ROLES.includes(role ?? '')
}

export function isStoreRole(role: string | null | undefined): boolean {
  return STORE_ROLES.includes(role ?? '')
}

/** Can create/update catalog, employees, purchases and inventory. */
export function canManage(role: string | null | undefined): boolean {
  return MANAGER_ROLES.includes(role ?? '')
}

export function isOwner(role: string | null | undefined): boolean {
  return role === 'owner'
}

export const ROLE_LABELS: Record<string, string> = {
  owner: 'Owner',
  manager: 'Manager',
  cashier: 'Cashier',
  saas_admin: 'SaaS Admin',
  superadmin: 'Super Admin',
}