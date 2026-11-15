import type { ReactNode } from 'react'
import { DataTable, type Column } from '../../components/common'
import { Badge } from '../../components/ui'
import { useT } from '../../i18n/I18nProvider'
import { formatDateTime } from '../../lib/format'
import { cn } from '../../lib/utils'
import type { AuditAction, AuditEvent } from '../../types'

type Translator = (key: string, params?: Record<string, string | number>) => string

const statusActions: AuditAction[] = ['USER_ACTIVATED', 'USER_DEACTIVATED', 'PRODUCT_ACTIVATED', 'PRODUCT_DEACTIVATED']

export function auditActionLabel(action: string, t: Translator): string {
  const key = `audit.action.${action}`
  const value = t(key)
  return value === key ? action.toLowerCase().replaceAll('_', ' ') : value
}

export function auditFieldLabel(field: string, t: Translator): string {
  const key = `audit.field.${field}`
  const value = t(key)
  return value === key ? field.toLowerCase().replaceAll('_', ' ') : value
}

function fieldValue(event: AuditEvent, field: string, value: unknown, t: Translator): ReactNode {
  if (event.redacted_fields.includes(field)) return <span className="text-slate-400">{t('audit.hidden')}</span>
  if (value === null || value === undefined || value === '') return t('common.notAvailable')
  if (field === 'is_active' && typeof value === 'boolean') return value ? t('common.active') : t('common.inactive')
  if (typeof value === 'boolean') return value ? t('common.yes') : t('common.no')
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function statusValue(value: unknown, t: Translator): string {
  return value === true ? t('common.active') : t('common.inactive')
}

export function auditChangeSummary(event: AuditEvent, t: Translator): string {
  if (event.action === 'PRODUCT_COST_UPDATED' && event.redacted_fields.includes('unit_cost')) {
    return t('audit.costHidden')
  }
  if (event.action === 'USER_PASSWORD_CHANGED') return t('audit.passwordChanged')
  if (statusActions.includes(event.action)) {
    const before = event.before?.is_active
    const after = event.after?.is_active
    if (typeof before === 'boolean' && typeof after === 'boolean') {
      return t('audit.statusTransition', { from: statusValue(before, t), to: statusValue(after, t) })
    }
  }
  if (!event.changed_fields.length) return t('audit.noFieldChanges')
  return event.changed_fields.map((field) => auditFieldLabel(field, t)).join(', ')
}

function rowsFor(event: AuditEvent, side: 'before' | 'after', t: Translator): Array<{ field: string; value: ReactNode }> {
  const values = event[side] ?? {}
  const fields = new Set([...Object.keys(values), ...event.redacted_fields])
  return Array.from(fields)
    .filter((field) => event.changed_fields.includes(field) || field === 'is_active' || field === 'unit_cost' || event.redacted_fields.includes(field))
    .map((field) => ({ field, value: fieldValue(event, field, values[field], t) }))
}

export function AuditEventDetail({ event }: { event: AuditEvent }) {
  const { t } = useT()
  const before = rowsFor(event, 'before', t)
  const after = rowsFor(event, 'after', t)
  const fields = Array.from(new Set([...before.map((row) => row.field), ...after.map((row) => row.field)]))
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-1.5">
        {event.changed_fields.map((field) => (
          <Badge key={field} tone="blue">{auditFieldLabel(field, t)}</Badge>
        ))}
      </div>
      {fields.length > 0 && (
        <div className="overflow-x-auto rounded-md border border-slate-200 bg-white">
          <table className="w-full min-w-[420px] text-left text-sm">
            <thead className="border-b border-slate-200 bg-slate-50 text-xs uppercase text-slate-500">
              <tr>
                <th className="px-3 py-2 font-medium">{t('audit.fieldName')}</th>
                <th className="px-3 py-2 font-medium">{t('audit.before')}</th>
                <th className="px-3 py-2 font-medium">{t('audit.after')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {fields.map((field) => (
                <tr key={field}>
                  <td className="px-3 py-2 text-slate-600">{auditFieldLabel(field, t)}</td>
                  <td className="px-3 py-2 text-slate-700">{before.find((row) => row.field === field)?.value ?? t('common.notAvailable')}</td>
                  <td className="px-3 py-2 text-slate-700">{after.find((row) => row.field === field)?.value ?? t('common.notAvailable')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {event.redacted_fields.length > 0 && (
        <p className="text-xs text-amber-700">{t('audit.redactedNotice')}</p>
      )}
      {event.reason && (
        <p className="text-sm text-slate-600"><span className="font-medium">{t('details.reason')}:</span> {event.reason}</p>
      )}
      {event.request_id && (
        <p className="text-xs text-slate-500">{t('audit.requestId')}: <span className="font-mono">{event.request_id}</span></p>
      )}
    </div>
  )
}

export function AuditEventTable({
  events,
  loading,
  error,
  onRetry,
  showSubject = true,
  emptyTitle,
  className,
}: {
  events: AuditEvent[]
  loading?: boolean
  error?: unknown
  onRetry?: () => void
  showSubject?: boolean
  emptyTitle?: string
  className?: string
}) {
  const { t } = useT()
  const columns: Column<AuditEvent>[] = [
    { key: 'when', header: t('details.when'), render: (event) => formatDateTime(event.created_at) },
    {
      key: 'event',
      header: t('audit.event'),
      render: (event) => (
        <div>
          <p className="font-medium text-slate-900">{auditActionLabel(event.action, t)}</p>
          <p className="mt-0.5 text-xs text-slate-500">{auditChangeSummary(event, t)}</p>
        </div>
      ),
    },
    ...(showSubject ? [{
      key: 'subject' as const,
      header: t('audit.subject'),
      render: (event: AuditEvent) => (
        <div className="min-w-0">
          <p className="truncate font-medium text-slate-800">{event.entity_label || event.entity_id}</p>
          <p className="text-xs text-slate-500">{t(`audit.entity.${event.entity_type}`)}</p>
        </div>
      ),
    }] : []),
    {
      key: 'actor',
      header: t('details.by'),
      render: (event) => (
        <div>
          <p>{event.actor.full_name}</p>
          <p className="text-xs text-slate-500">{t(`role.${event.actor.role}`)}</p>
        </div>
      ),
    },
    {
      key: 'reason',
      header: t('details.reason'),
      render: (event) => <span className={cn(!event.reason && 'text-slate-400')}>{event.reason || t('common.notAvailable')}</span>,
    },
  ]
  return (
    <div className={className}>
      <DataTable
        columns={columns}
        rows={events}
        loading={loading}
        error={error}
        onRetry={onRetry}
        getRowKey={(event) => event.id}
        renderExpanded={(event) => <AuditEventDetail event={event} />}
        emptyTitle={emptyTitle ?? t('audit.noEvents')}
      />
    </div>
  )
}
