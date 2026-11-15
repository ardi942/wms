import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nProvider } from '../../i18n/I18nProvider'
import type { User } from '../../types'
import { UserStatusDialog } from './UserStatusDialog'

const user: User = {
  id: 'user-1',
  email: 'staff@wms.local',
  full_name: 'Budi Santoso',
  role: 'DEPARTMENT_USER',
  department: null,
  is_active: true,
  created_at: '2026-11-15T09:00:00Z',
  updated_at: '2026-11-15T09:00:00Z',
}

describe('UserStatusDialog', () => {
  it('requires a reason of at least five characters', async () => {
    const onConfirm = vi.fn()
    render(
      <I18nProvider>
        <UserStatusDialog open user={user} onCancel={() => {}} onConfirm={onConfirm} />
      </I18nProvider>,
    )
    const actor = userEvent.setup()
    await actor.click(screen.getByRole('button', { name: 'Deactivate' }))
    expect(screen.getByText('Reason must be at least 5 characters')).toBeInTheDocument()
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('submits a trimmed reason', async () => {
    const onConfirm = vi.fn()
    render(
      <I18nProvider>
        <UserStatusDialog open user={user} onCancel={() => {}} onConfirm={onConfirm} />
      </I18nProvider>,
    )
    const actor = userEvent.setup()
    await actor.type(screen.getByLabelText('Reason for status change'), '  Account left the company  ')
    await actor.click(screen.getByRole('button', { name: 'Deactivate' }))
    expect(onConfirm).toHaveBeenCalledWith('Account left the company')
  })
})
