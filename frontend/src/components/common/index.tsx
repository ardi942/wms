import { Fragment, useEffect, useRef, useState, type ReactNode } from 'react'
import { useT } from '../../i18n/I18nProvider'
import { AlertTriangle, CalendarDays, ChevronRight, Inbox, RefreshCw, Search, X } from 'lucide-react'
import { DayPicker, type DateRange } from 'react-day-picker'
import 'react-day-picker/style.css'
import { ApiError } from '../../api/client'
import { productApi } from '../../api/endpoints'
import type { Product } from '../../types'
import { Button, Card, Input, Select, Spinner, Badge } from '../ui'
import { cn } from '../../lib/utils'
import { formatQty, formatTime, statusLabel, statusTone } from '../../lib/format'
import { PAGE_SIZES } from '../../lib/constants'
import type { Meta } from '../../types'

export function PageHeader({ title, description, actions }: { title: string; description?: string; actions?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:flex-wrap sm:items-start sm:justify-between">
      <div className="min-w-0">
        <h1 className="text-xl font-semibold text-slate-900 sm:text-2xl">{title}</h1>
        {description && <p className="mt-1 text-sm text-slate-500">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2 [&>*]:flex-1 sm:[&>*]:flex-none">{actions}</div>}
    </div>
  )
}

export function Loading({ label }: { label?: string }) {
  const { t } = useT()
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-sm text-slate-500">
      <Spinner /> {label ?? t('common.loading')}
    </div>
  )
}

export function AutoRefreshIndicator({
  updatedAt,
  intervalMs,
  isFetching,
  onRefresh,
}: {
  updatedAt: number
  intervalMs: number
  isFetching: boolean
  onRefresh: () => void
}) {
  const { t } = useT()
  return (
    <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500">
      <span className="relative flex h-2 w-2" aria-hidden="true">
        <span className={cn('absolute inline-flex h-full w-full rounded-full bg-emerald-400', isFetching && 'animate-ping')} />
        <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500" />
      </span>
      <span>{t('common.autoRefresh', { seconds: Math.round(intervalMs / 1000) })}</span>
      {updatedAt > 0 && (
        <span aria-live="polite">{t('common.lastUpdated', { time: formatTime(updatedAt) })}</span>
      )}
      <button
        type="button"
        onClick={onRefresh}
        aria-label={t('common.refreshNow')}
        title={t('common.refreshNow')}
        className="grid h-8 w-8 place-items-center rounded-md text-slate-500 hover:bg-slate-100 disabled:opacity-50"
        disabled={isFetching}
      >
        <RefreshCw className={cn('h-3.5 w-3.5', isFetching && 'animate-spin')} />
      </button>
    </div>
  )
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const { t } = useT()
  const message = error instanceof ApiError ? error.message : error instanceof Error ? error.message : t('common.somethingWrong')
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16 text-center">
      <AlertTriangle className="h-8 w-8 text-red-500" />
      <p className="text-sm text-slate-600">{message}</p>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCw className="h-3.5 w-3.5" /> {t('common.retry')}
        </Button>
      )}
    </div>
  )
}

export function EmptyState({ title, description, action }: { title: string; description?: string; action?: ReactNode }) {
  const { t } = useT()
  const defaultTitle = t('common.noRecords')
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-14 text-center">
      <Inbox className="h-8 w-8 text-slate-300" />
      <p className="font-medium text-slate-700">{title || defaultTitle}</p>
      {description && <p className="max-w-sm text-sm text-slate-500">{description}</p>}
      {action}
    </div>
  )
}

export function StatusBadge({ status }: { status: string }) {
  const { t } = useT()
  const key = `status.${status}`
  const translated = t(key)
  return <Badge tone={statusTone(status)}>{translated === key ? statusLabel(status) : translated}</Badge>
}

export interface Column<T> {
  key: string
  header: string
  className?: string
  render: (row: T) => ReactNode
  tooltip?: (row: T) => string | undefined
}

export function DataTable<T>({
  columns,
  rows,
  loading,
  error,
  onRetry,
  emptyTitle,
  onRowClick,
  renderExpanded,
  getRowKey,
}: {
  columns: Column<T>[]
  rows: T[]
  loading?: boolean
  error?: unknown
  onRetry?: () => void
  emptyTitle?: string
  onRowClick?: (row: T) => void
  renderExpanded?: (row: T) => ReactNode
  getRowKey?: (row: T, index: number) => string | number
}) {
  const { t } = useT()
  const [expanded, setExpanded] = useState<Set<string | number>>(new Set())
  if (loading) return <Loading />
  if (error) return <ErrorState error={error} onRetry={onRetry} />
  if (!rows.length) return <EmptyState title={emptyTitle ?? t('common.noRecords')} />
  const columnCount = columns.length + (renderExpanded ? 1 : 0)
  const primaryIndex = columns.findIndex((c) => c.header)
  const primary = primaryIndex >= 0 ? columns[primaryIndex] : undefined
  const detailColumns = columns.filter((c, index) => index !== primaryIndex && c.header)
  const actionColumns = columns.filter((c) => !c.header)
  const toggle = (key: string | number) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }
  return (
    <>
      <div className="divide-y divide-slate-100 md:hidden" data-testid="data-table-mobile">
        {rows.map((row, index) => {
          const key = getRowKey ? getRowKey(row, index) : index
          const interactive = Boolean(onRowClick)
          return (
            <div
              key={key}
              role={interactive ? 'button' : undefined}
              tabIndex={interactive ? 0 : undefined}
              onClick={onRowClick ? () => onRowClick(row) : undefined}
              onKeyDown={
                interactive
                  ? (event) => {
                      if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault()
                        onRowClick?.(row)
                      }
                    }
                  : undefined
              }
              className={cn('bg-white p-4', interactive && 'cursor-pointer active:bg-slate-50')}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1 text-sm font-medium text-slate-900">
                  {primary ? primary.render(row) : null}
                </div>
                {actionColumns.length > 0 && (
                  <div className="flex shrink-0 items-center gap-2" onClick={(event) => event.stopPropagation()}>
                    {actionColumns.map((c) => (
                      <Fragment key={c.key}>{c.render(row)}</Fragment>
                    ))}
                  </div>
                )}
              </div>
              {detailColumns.length > 0 && (
                <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
                  {detailColumns.map((c) => (
                    <Fragment key={c.key}>
                      <dt className="text-slate-500">{c.header}</dt>
                      <dd className="min-w-0 break-words text-right text-slate-700">{c.render(row)}</dd>
                    </Fragment>
                  ))}
                </dl>
              )}
              {renderExpanded && (
                <details className="mt-2" onClick={(event) => event.stopPropagation()}>
                  <summary className="cursor-pointer text-xs font-medium text-brand-600">{t('common.details')}</summary>
                  <div className="mt-2 rounded-md bg-slate-50 p-3 text-sm">{renderExpanded(row)}</div>
                </details>
              )}
            </div>
          )
        })}
      </div>
      <div className="hidden overflow-x-auto md:block">
        <table className="w-full min-w-[640px] text-left text-sm">
          <thead className="border-b border-slate-200 bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
            <tr>
              {renderExpanded && <th className="w-10 px-4 py-3" aria-label={t('actions.expand')} />}
              {columns.map((c) => (
                <th key={c.key} className={cn('px-4 py-3 font-medium', c.className)}>{c.header}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {rows.map((row, index) => {
              const key = getRowKey ? getRowKey(row, index) : index
              const isOpen = expanded.has(key)
              return (
                <Fragment key={key}>
                  <tr
                    className={cn('bg-white transition-colors', onRowClick && 'cursor-pointer hover:bg-slate-50')}
                    onClick={onRowClick ? () => onRowClick(row) : undefined}
                  >
                    {renderExpanded && (
                      <td className="px-4 py-3 align-middle">
                        <button
                          type="button"
                          aria-label={isOpen ? t('actions.collapse') : t('actions.expand')}
                          aria-expanded={isOpen}
                          onClick={(e) => { e.stopPropagation(); toggle(key) }}
                          className="rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
                        >
                          <ChevronRight className={cn('h-4 w-4 transition-transform', isOpen && 'rotate-90')} />
                        </button>
                      </td>
                    )}
                    {columns.map((c) => (
                      <td
                        key={c.key}
                        title={c.tooltip?.(row)}
                        className={cn('px-4 py-3 align-middle', c.className)}
                      >
                        {c.render(row)}
                      </td>
                    ))}
                  </tr>
                  {renderExpanded && isOpen && (
                    <tr className="bg-slate-50">
                      <td colSpan={columnCount} className="px-4 py-4">
                        {renderExpanded(row)}
                      </td>
                    </tr>
                  )}
                </Fragment>
              )
            })}
          </tbody>
        </table>
      </div>
    </>
  )
}

export function Pagination({
  meta,
  onPage,
  onPageSize,
}: {
  meta: Meta
  onPage: (page: number) => void
  onPageSize: (size: number) => void
}) {
  const { t } = useT()
  const start = meta.total === 0 ? 0 : (meta.page - 1) * meta.page_size + 1
  const end = Math.min(meta.page * meta.page_size, meta.total)
  return (
    <div className="flex flex-col gap-3 border-t border-slate-200 px-4 py-3 text-sm text-slate-600 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
      <span>{t('common.pageRange', { start, end, total: meta.total })}</span>
      <div className="flex items-center justify-between gap-3 sm:justify-end">
        <Select
          className="h-11 w-28 text-base sm:h-8 sm:w-24 sm:text-xs"
          value={meta.page_size}
          onChange={(e) => onPageSize(Number(e.target.value))}
          aria-label={t('common.pageSize')}
        >
          {PAGE_SIZES.map((s) => (
            <option key={s} value={s}>{t('common.perPage', { size: s })}</option>
          ))}
        </Select>
        <div className="flex gap-1">
          <Button variant="outline" size="sm" disabled={meta.page <= 1} onClick={() => onPage(meta.page - 1)}>{t('common.previous')}</Button>
          <Button variant="outline" size="sm" disabled={meta.page >= meta.total_pages} onClick={() => onPage(meta.page + 1)}>{t('common.next')}</Button>
        </div>
      </div>
    </div>
  )
}

export function ConfirmDialog({
  open,
  title,
  children,
  onCancel,
  onConfirm,
  confirmLabel,
  loading,
  danger,
}: {
  open: boolean
  title: string
  children?: ReactNode
  onCancel: () => void
  onConfirm: () => void
  confirmLabel?: string
  loading?: boolean
  danger?: boolean
}) {
  const { t } = useT()
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center">
      <Card className="max-h-[90dvh] w-full max-w-md overflow-y-auto p-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))] sm:pb-5">
        <div className="flex items-start justify-between gap-4">
          <h3 className="text-lg font-semibold text-slate-900">{title}</h3>
          <button onClick={onCancel} className="grid h-10 w-10 place-items-center rounded-md text-slate-400 hover:bg-slate-100 hover:text-slate-600" aria-label={t('common.close')}><X className="h-5 w-5" /></button>
        </div>
        <div className="mt-3 text-sm text-slate-600">{children}</div>
        <div className="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end [&>*]:w-full sm:[&>*]:w-auto">
          <Button variant="outline" onClick={onCancel}>{t('common.cancel')}</Button>
          <Button variant={danger ? 'danger' : 'primary'} loading={loading} onClick={onConfirm}>{confirmLabel ?? t('common.confirm')}</Button>
        </div>
      </Card>
    </div>
  )
}

export function FilterBar({ children }: { children: ReactNode }) {
  return <div className="flex flex-col gap-3 border-b border-slate-200 px-4 py-3 sm:flex-row sm:flex-wrap sm:items-end">{children}</div>
}

export function SearchInput({ value, onChange, placeholder }: { value: string; onChange: (v: string) => void; placeholder?: string }) {
  const { t } = useT()
  return (
    <div className="relative w-full sm:w-64">
      <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
      <Input value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder ?? t('common.search')} className="h-11 pl-8 sm:h-9" />
    </div>
  )
}

function isoDate(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function parseIsoDate(s: string): Date | undefined {
  if (!s) return undefined
  const [y, m, d] = s.split('-').map(Number)
  if (!y || !m || !d) return undefined
  return new Date(y, m - 1, d)
}

export function RangeDatePicker({ from, to, onFrom, onTo }: { from: string; to: string; onFrom: (v: string) => void; onTo: (v: string) => void }) {
  const { t } = useT()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const range: DateRange | undefined = from || to ? { from: parseIsoDate(from), to: parseIsoDate(to) } : undefined

  const setRange = (r: DateRange | undefined) => {
    onFrom(r?.from ? isoDate(r.from) : '')
    onTo(r?.to ? isoDate(r.to) : '')
  }
  const preset = (days: number) => {
    const end = new Date()
    const start = new Date()
    start.setDate(end.getDate() - (days - 1))
    onFrom(isoDate(start))
    onTo(isoDate(end))
    setOpen(false)
  }
  const presetMonth = () => {
    const end = new Date()
    const start = new Date(end.getFullYear(), end.getMonth(), 1)
    onFrom(isoDate(start))
    onTo(isoDate(end))
    setOpen(false)
  }
  const label = from && to ? t('common.dateRange', { from, to }) : from ? t('common.fromDate', { date: from }) : to ? t('common.untilDate', { date: to }) : t('common.anyDate')

  return (
    <div className="relative w-full sm:w-auto" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex h-11 w-full items-center justify-between gap-2 rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-700 hover:bg-slate-50 sm:h-9 sm:w-auto sm:justify-start"
      >
        <CalendarDays className="h-4 w-4 shrink-0 text-slate-400" />
        <span className="truncate">{label}</span>
      </button>
      {open && (
        <div className="absolute left-0 right-0 z-50 mt-2 max-h-[70dvh] w-auto max-w-[calc(100vw-2rem)] overflow-auto rounded-lg border border-slate-200 bg-white p-3 shadow-lg sm:left-auto sm:right-0 sm:w-max">
          <div className="mb-2 flex flex-wrap gap-1">
            <Button variant="ghost" size="sm" onClick={() => preset(1)}>{t('common.today')}</Button>
            <Button variant="ghost" size="sm" onClick={() => preset(7)}>{t('common.last7')}</Button>
            <Button variant="ghost" size="sm" onClick={() => preset(30)}>{t('common.last30')}</Button>
            <Button variant="ghost" size="sm" onClick={presetMonth}>{t('common.thisMonth')}</Button>
            <Button variant="ghost" size="sm" onClick={() => { onFrom(''); onTo('') }}>{t('common.clear')}</Button>
          </div>
          <DayPicker
            mode="range"
            selected={range}
            onSelect={setRange}
            numberOfMonths={1}
            defaultMonth={parseIsoDate(from) ?? parseIsoDate(to) ?? new Date()}
          />
        </div>
      )}
    </div>
  )
}

export const DateRangeFilter = RangeDatePicker

export function ProductPicker({
  id,
  value,
  onChange,
  disabled,
  'aria-describedby': ariaDescribedBy,
  'aria-invalid': ariaInvalid,
}: {
  id?: string
  value: Product | null
  onChange: (p: Product | null) => void
  disabled?: boolean
  'aria-describedby'?: string
  'aria-invalid'?: boolean
}) {
  const { t } = useT()
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const [rows, setRows] = useState<Product[]>([])
  const [loading, setLoading] = useState(false)
  const timer = useRef<number | null>(null)

  useEffect(() => {
    if (!open) return
    if (timer.current) window.clearTimeout(timer.current)
    timer.current = window.setTimeout(async () => {
      setLoading(true)
      try {
        const res = await productApi.list({ search: query, page_size: 10 })
        setRows(res.items)
      } finally {
        setLoading(false)
      }
    }, 300)
    return () => {
      if (timer.current) window.clearTimeout(timer.current)
    }
  }, [query, open])

  if (value) {
    return (
      <div className="flex min-h-10 items-center justify-between gap-2 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm">
        <span className="min-w-0">
          <span className="block truncate">
            <span className="font-medium">{value.sku}</span> — {value.name}
          </span>
          <span className="block text-xs text-slate-500">
            {value.category.name} · {formatQty(value.current_stock)} {value.unit.code}
            {value.is_low_stock ? ` · ${t('common.lowStock')}` : ''}
          </span>
        </span>
        {!disabled && <button onClick={() => onChange(null)} aria-label={t('common.clear')} className="shrink-0 text-slate-400 hover:text-slate-600"><X className="h-4 w-4" /></button>}
      </div>
    )
  }

  return (
    <div className="relative">
      <Input
        id={id}
        value={query}
        disabled={disabled}
        onFocus={() => setOpen(true)}
        onChange={(e) => { setQuery(e.target.value); setOpen(true) }}
        placeholder={t('common.searchBySkuOrName')}
        aria-describedby={ariaDescribedBy}
        aria-invalid={ariaInvalid}
      />
      {open && (
        <div className="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-md border border-slate-200 bg-white shadow-lg">
          {loading && <div className="px-3 py-2 text-sm text-slate-500">{t('empty.searching')}</div>}
          {!loading && rows.length === 0 && <div className="px-3 py-2 text-sm text-slate-500">{t('empty.noMatchingProducts')}</div>}
          {!loading && rows.filter((r) => r.is_active).map((p) => (
            <button
              key={p.id}
              type="button"
              className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-slate-50"
              onClick={() => { onChange(p); setOpen(false); setQuery('') }}
            >
              <span className="min-w-0">
                <span className="block truncate"><span className="font-medium">{p.sku}</span> — {p.name}</span>
                <span className="block text-xs text-slate-500">{p.category.name} · {formatQty(p.current_stock)} {p.unit.code}</span>
              </span>
              {p.is_low_stock && <Badge tone="amber">{t('common.lowStock')}</Badge>}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}


/* WorkflowStepper — appended per plan §3.3.1 */
export function WorkflowStepper({
  steps,
  current,
}: {
  steps: Array<{ key: string; label: string }>
  current: string
}) {
  const { t } = useT()
  const terminal = current === 'REJECTED' || current === 'CANCELLED'
  const currentIndex = steps.findIndex((s) => s.key === current)
  return (
    <ol className="flex flex-wrap items-center gap-2">
      {steps.map((step, index) => {
        const done = !terminal && currentIndex > index
        const active = !terminal && currentIndex === index
        return (
          <li key={step.key} className="flex items-center gap-2">
            <span
              className={cn(
                'grid h-6 w-6 place-items-center rounded-full text-xs font-semibold',
                done && 'bg-emerald-100 text-emerald-700',
                active && 'bg-brand-600 text-white',
                !done && !active && 'bg-slate-100 text-slate-500',
              )}
            >
              {done ? '✓' : index + 1}
            </span>
            <span className={cn('text-sm', active ? 'font-medium text-slate-900' : 'text-slate-500')}>
              {step.label}
            </span>
            {index < steps.length - 1 && <span className="mx-1 h-px w-6 bg-slate-200" />}
          </li>
        )
      })}
      {terminal && <Badge tone="red">{t(`status.${current}`)}</Badge>}
    </ol>
  )
}
