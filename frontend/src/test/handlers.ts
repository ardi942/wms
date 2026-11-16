import { http, HttpResponse } from 'msw'
import type { User } from '../types'

export const demoUser: User = {
  id: '11111111-1111-4111-8111-111111111111',
  email: 'warehouse@wms.local',
  full_name: 'Wendy Warehouse',
  role: 'WAREHOUSE_ADMIN',
  department: null,
  is_active: true,
  image_url: null,
  image_updated_at: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

export const handlers = [
  http.post('*/api/v1/auth/login', async ({ request }) => {
    const body = (await request.json()) as { email?: string; password?: string }
    if (body.email === demoUser.email && body.password === 'Warehouse123!') {
      return HttpResponse.json({ data: { token: 'test-token', user: demoUser } })
    }
    return HttpResponse.json(
      { error: { code: 'UNAUTHORIZED', message: 'invalid email or password', details: null } },
      { status: 401 },
    )
  }),
  http.get('*/api/v1/auth/me', () => HttpResponse.json({ data: { user: demoUser } })),
  http.get('*/api/v1/notifications/unread-count', () => HttpResponse.json({ data: { unread_count: 1 } })),
  http.get('*/api/v1/departments', () => HttpResponse.json({ data: [{ id: 'd1', code: 'IT', name: 'Information Technology' }] })),
  http.get('*/api/v1/categories', () => HttpResponse.json({ data: [{ id: 'c1', name: 'IT Equipment' }] })),
  http.get('*/api/v1/units', () => HttpResponse.json({ data: [{ id: 'u1', code: 'PCS', name: 'Pieces' }] })),
  http.get('*/api/v1/monitoring/requests', () => HttpResponse.json({
    data: [{
      id: 'req-1',
      request_number: 'REQ-2026-0001',
      status: 'PENDING',
      department: { id: 'd1', code: 'IT', name: 'Information Technology' },
      requested_by: { id: 'staff-1', full_name: 'Ayu Lestari' },
      item_count: 3,
      total_requested: 10,
      total_issued: 4,
      created_at: '2026-11-15T08:00:00Z',
      updated_at: '2026-11-15T08:15:00Z',
    }],
    meta: { page: 1, page_size: 20, total: 1, total_pages: 1 },
  })),
  http.get('*/api/v1/monitoring/returns', () => HttpResponse.json({
    data: [{
      id: 'ret-1',
      return_number: 'RET-2026-0001',
      status: 'CONFIRMED',
      department: { id: 'd2', code: 'FIN', name: 'Finance' },
      requested_by: { id: 'staff-2', full_name: 'Budi Santoso' },
      item_count: 2,
      total_requested: 5,
      total_accepted: 5,
      created_at: '2026-11-15T09:00:00Z',
      updated_at: '2026-11-15T09:30:00Z',
    }],
    meta: { page: 1, page_size: 20, total: 1, total_pages: 1 },
  })),
]
