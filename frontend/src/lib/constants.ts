import type { Role } from '../types'

export const ROLE_SUPERVISOR: Role = 'SUPERVISOR'
export const ROLE_ADMIN: Role = 'WAREHOUSE_ADMIN'
export const ROLE_DEPARTMENT: Role = 'DEPARTMENT_USER'

export const PAGE_SIZES = [10, 20, 50]

export const navItems: Array<{ to: string; label: string; roles: Role[] }> = [
  { to: '/dashboard', label: 'Dashboard', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/products', label: 'Products', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/inventory', label: 'Inventory', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/receiving', label: 'Receiving', roles: ['WAREHOUSE_ADMIN'] },
  { to: '/requests', label: 'Stock Requests', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/returns', label: 'Returns', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/adjustments', label: 'Adjustments', roles: ['WAREHOUSE_ADMIN'] },
  { to: '/reports', label: 'Reports', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/users', label: 'Users', roles: ['WAREHOUSE_ADMIN'] },
  { to: '/notifications', label: 'Notifications', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
]
