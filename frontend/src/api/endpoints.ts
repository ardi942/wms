import { api, toQuery } from './client'
import type {
  Category,
  Dashboard,
  Department,
  InventoryBalance,
  InventoryReportRow,
  ListResult,
  LoginResult,
  LowStockReportRow,
  Meta,
  Movement,
  Notification,
  Product,
  Role,
  StockRequest,
  StockRequestList,
  StockReturn,
  StockReturnList,
  Unit,
  UsageReportRow,
  User,
} from '../types'

interface Envelope<T> {
  data: T
  meta?: Meta
}

async function list<T>(path: string, params?: Record<string, unknown>): Promise<ListResult<T>> {
  const res = await api.get<Envelope<T[]>>(`${path}${toQuery(params)}`)
  return {
    items: res.data,
    meta: res.meta ?? { page: 1, page_size: res.data.length, total: res.data.length, total_pages: 1 },
  }
}

export const authApi = {
  login: (email: string, password: string) => api.post<Envelope<LoginResult>>('/auth/login', { email, password }),
  me: () => api.get<Envelope<{ user: User }>>('/auth/me'),
}

export const referenceApi = {
  departments: () => api.get<Envelope<Department[]>>('/departments'),
  categories: () => api.get<Envelope<Category[]>>('/categories'),
  units: () => api.get<Envelope<Unit[]>>('/units'),
}

export const productApi = {
  list: (params: Record<string, unknown>) => list<Product>('/products', params),
  get: (id: string) => api.get<Envelope<Product>>(`/products/${id}`),
  create: (body: unknown) => api.post<Envelope<Product>>('/products', body),
  update: (id: string, body: unknown) => api.patch<Envelope<Product>>(`/products/${id}`, body),
}

export const inventoryApi = {
  balances: (params: Record<string, unknown>) => list<InventoryBalance>('/inventory', params),
  product: (productId: string) => api.get<Envelope<InventoryBalance>>(`/inventory/${productId}`),
  movements: (params: Record<string, unknown>) => list<Movement>('/inventory/movements', params),
  receive: (body: unknown) => api.post<Envelope<Movement>>('/inventory/receipts', body),
  adjust: (body: unknown) => api.post<Envelope<Movement>>('/inventory/adjustments', body),
}

export const requestApi = {
  list: (params: Record<string, unknown>) => list<StockRequestList>('/stock-requests', params),
  get: (id: string) => api.get<Envelope<StockRequest>>(`/stock-requests/${id}`),
  create: (body: unknown) => api.post<Envelope<StockRequest>>('/stock-requests', body),
  fulfill: (id: string, body: unknown) => api.post<Envelope<StockRequest>>(`/stock-requests/${id}/fulfill`, body),
  reject: (id: string, body: unknown) => api.post<Envelope<StockRequest>>(`/stock-requests/${id}/reject`, body),
  cancel: (id: string) => api.post<Envelope<StockRequest>>(`/stock-requests/${id}/cancel`),
}

export const returnApi = {
  list: (params: Record<string, unknown>) => list<StockReturnList>('/stock-returns', params),
  get: (id: string) => api.get<Envelope<StockReturn>>(`/stock-returns/${id}`),
  create: (body: unknown) => api.post<Envelope<StockReturn>>('/stock-returns', body),
  confirm: (id: string, body: unknown) => api.post<Envelope<StockReturn>>(`/stock-returns/${id}/confirm`, body),
  reject: (id: string, body: unknown) => api.post<Envelope<StockReturn>>(`/stock-returns/${id}/reject`, body),
  cancel: (id: string) => api.post<Envelope<StockReturn>>(`/stock-returns/${id}/cancel`),
}

export const reportApi = {
  dashboard: () => api.get<Envelope<Dashboard>>('/reports/dashboard'),
  recentMovements: (params: Record<string, unknown>) =>
    list<Movement>('/reports/dashboard/recent-movements', params),
  inventory: (params: Record<string, unknown>) => list<InventoryReportRow>('/reports/inventory', params),
  movements: (params: Record<string, unknown>) => list<Movement>('/reports/stock-movements', params),
  departmentUsage: (params: Record<string, unknown>) => list<UsageReportRow>('/reports/department-usage', params),
  lowStock: (params: Record<string, unknown>) => list<LowStockReportRow>('/reports/low-stock', params),
}

export const userApi = {
  list: (params: Record<string, unknown>) => list<User>('/users', params),
  get: (id: string) => api.get<Envelope<User>>(`/users/${id}`),
  create: (body: unknown) => api.post<Envelope<User>>('/users', body),
  update: (id: string, body: unknown) => api.patch<Envelope<User>>(`/users/${id}`, body),
}

export const notificationApi = {
  list: (params: Record<string, unknown>) => list<Notification>('/notifications', params),
  unreadCount: () => api.get<Envelope<{ unread_count: number }>>('/notifications/unread-count'),
  markRead: (id: string) => api.patch<void>(`/notifications/${id}/read`),
  markAllRead: () => api.post<void>('/notifications/read-all'),
}

export const roleLabels: Record<Role, string> = {
  SUPERVISOR: 'Supervisor',
  WAREHOUSE_ADMIN: 'Warehouse Admin',
  DEPARTMENT_USER: 'Department User',
}
