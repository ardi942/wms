import { describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from '../../auth/AuthProvider'
import { I18nProvider } from '../../i18n/I18nProvider'
import { server } from '../../test/server'
import { MonitoringPage } from './MonitoringPage'

function renderMonitoring() {
  localStorage.setItem('wms.token', 'test-token')
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <MemoryRouter initialEntries={['/monitoring']}>
          <AuthProvider>
            <MonitoringPage />
          </AuthProvider>
        </MemoryRouter>
      </I18nProvider>
    </QueryClientProvider>,
  )
}

describe('MonitoringPage', () => {
  it('renders cross-division request rows without action controls', async () => {
    renderMonitoring()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('REQ-2026-0001')).toBeInTheDocument()
    expect(within(table).getByText('Information Technology')).toBeInTheDocument()
    expect(within(table).getByText('Ayu Lestari')).toBeInTheDocument()
    expect(within(table).getByText('Pending')).toBeInTheDocument()
    expect(within(table).queryByRole('button', { name: /fulfil|reject|cancel|confirm/i })).not.toBeInTheDocument()
  })

  it('switches to the returns tab and loads return rows', async () => {
    const user = userEvent.setup()
    renderMonitoring()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Returns' }))

    const table = await screen.findByRole('table')
    expect(within(table).getByText('RET-2026-0001')).toBeInTheDocument()
    expect(within(table).getByText('Finance')).toBeInTheDocument()
    expect(within(table).getByText('Budi Santoso')).toBeInTheDocument()
  })

  it('refreshes manually', async () => {
    let calls = 0
    server.use(
      http.get('*/api/v1/monitoring/requests', () => {
        calls += 1
        return HttpResponse.json({
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
        })
      }),
    )
    const user = userEvent.setup()
    renderMonitoring()
    await screen.findByRole('table')
    expect(calls).toBe(1)

    await user.click(screen.getByRole('button', { name: 'Refresh now' }))

    await waitFor(() => expect(calls).toBe(2))
  })
})
