import type { Role } from '../types'

export const ROLE_SUPERVISOR: Role = 'SUPERVISOR'
export const ROLE_ADMIN: Role = 'WAREHOUSE_ADMIN'
export const ROLE_DEPARTMENT: Role = 'DEPARTMENT_USER'

export const PAGE_SIZES = [10, 20, 50]

export const navItems: Array<{ to: string; key: string; roles: Role[] }> = [
  { to: '/dashboard', key: 'nav.dashboard', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/products', key: 'nav.products', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/inventory', key: 'nav.inventory', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/receiving', key: 'nav.receiving', roles: ['WAREHOUSE_ADMIN'] },
  { to: '/requests', key: 'nav.requests', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/returns', key: 'nav.returns', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/adjustments', key: 'nav.adjustments', roles: ['WAREHOUSE_ADMIN'] },
  { to: '/reports', key: 'nav.reports', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { to: '/users', key: 'nav.users', roles: ['WAREHOUSE_ADMIN'] },
  { to: '/notifications', key: 'nav.notifications', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
]
