import { http, HttpResponse } from 'msw'
import type { User } from '../types'

export const demoUser: User = {
  id: '11111111-1111-4111-8111-111111111111',
  email: 'warehouse@wms.local',
  full_name: 'Wendy Warehouse',
  role: 'WAREHOUSE_ADMIN',
  department: null,
  is_active: true,
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
]
