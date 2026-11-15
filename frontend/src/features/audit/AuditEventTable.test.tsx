import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { I18nProvider } from '../../i18n/I18nProvider'
import type { AuditEvent } from '../../types'
import { AuditEventTable } from './AuditEventTable'

const base: AuditEvent = {
  id: 'event-1',
  entity_type: 'PRODUCT',
  entity_id: 'product-1',
  entity_label: 'OFF-PAP-001 — A4 Copy Paper',
  action: 'PRODUCT_COST_UPDATED',
  actor: { id: 'user-1', full_name: 'Wahyu Pratama', email: 'warehouse@wms.local', role: 'WAREHOUSE_ADMIN' },
  reason: null,
  changed_fields: ['unit_cost'],
  before: { unit_cost: 55000 },
  after: { unit_cost: 56000 },
  redacted_fields: ['unit_cost'],
  request_id: 'req-1',
  created_at: '2026-11-15T09:00:00Z',
}

describe('AuditEventTable', () => {
  it('renders an audited cost change without exposing the value to an admin', () => {
    render(
      <I18nProvider>
        <AuditEventTable events={[base]} />
      </I18nProvider>,
    )
    expect(screen.getAllByText('Unit cost updated').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Unit cost changed (value hidden)').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Hidden').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Wahyu Pratama').length).toBeGreaterThan(0)
  })

  it('renders a readable status transition', () => {
    render(
      <I18nProvider>
        <AuditEventTable
          events={[{
            ...base,
            entity_type: 'USER',
            action: 'USER_DEACTIVATED',
            changed_fields: ['is_active'],
            before: { is_active: true },
            after: { is_active: false },
            redacted_fields: [],
            reason: 'Security review requested',
          }]}
        />
      </I18nProvider>,
    )
    expect(screen.getAllByText('Account deactivated').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Status changed from Active to Inactive').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Security review requested').length).toBeGreaterThan(0)
  })
})
