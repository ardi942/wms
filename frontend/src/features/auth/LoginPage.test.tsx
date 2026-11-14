import { describe, expect, it } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { I18nProvider } from '../../i18n/I18nProvider'
import { AuthProvider } from '../../auth/AuthProvider'
import { LoginPage } from './LoginPage'

function renderLogin() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <MemoryRouter initialEntries={['/login']}>
          <AuthProvider>
            <Routes>
              <Route path="/login" element={<LoginPage />} />
              <Route path="/dashboard" element={<div>Dashboard home</div>} />
            </Routes>
          </AuthProvider>
        </MemoryRouter>
      </I18nProvider>
    </QueryClientProvider>,
  )
}

describe('LoginPage', () => {
  it('validates empty input', async () => {
    const user = userEvent.setup()
    renderLogin()
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByText('Enter a valid email')).toBeInTheDocument()
    expect(await screen.findByText('Password is required')).toBeInTheDocument()
  })

  it('logs in and stores the token', async () => {
    const user = userEvent.setup()
    renderLogin()
    await user.type(screen.getByLabelText('Email'), 'warehouse@wms.local')
    await user.type(screen.getByLabelText('Password'), 'Warehouse123!')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    await waitFor(() => expect(localStorage.getItem('wms.token')).toBe('test-token'))
    expect(await screen.findByText('Dashboard home')).toBeInTheDocument()
  })
})
