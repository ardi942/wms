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
import { formatQty, statusLabel, statusTone } from '../../lib/format'
import { PAGE_SIZES } from '../../lib/constants'
import type { Meta } from '../../types'

export function PageHeader({ title, description, actions }: { title: string; description?: string; actions?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 className="text-2xl font-semibold text-slate-900">{title}</h1>
        {description && <p className="mt-1 text-sm text-slate-500">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
  )
}

export function Loading({ label = 'Loading…' }: { label?: string }) {
  const { t } = useT()
  const defaultLabel = t('common.loading')
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-sm text-slate-500">
      <Spinner /> {label || defaultLabel}
    </div>
  )
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const message = error instanceof ApiError ? error.message : error instanceof Error ? error.message : 'Something went wrong'
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16 text-center">
      <AlertTriangle className="h-8 w-8 text-red-500" />
      <p className="text-sm text-slate-600">{message}</p>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCw className="h-3.5 w-3.5" /> Retry
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
  const toggle = (key: string | number) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[640px] text-left text-sm">
        <thead className="border-b border-slate-200 bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
          <tr>
            {renderExpanded && <th className="w-10 px-4 py-3" aria-label="Expand" />}
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
  const start = meta.total === 0 ? 0 : (meta.page - 1) * meta.page_size + 1
  const end = Math.min(meta.page * meta.page_size, meta.total)
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 px-4 py-3 text-sm text-slate-600">
      <span>
        {start}–{end} of {meta.total}
      </span>
      <div className="flex items-center gap-3">
        <Select
          className="h-8 w-20 text-xs"
          value={meta.page_size}
          onChange={(e) => onPageSize(Number(e.target.value))}
          aria-label="Page size"
        >
          {PAGE_SIZES.map((s) => (
            <option key={s} value={s}>{s} / page</option>
          ))}
        </Select>
        <div className="flex gap-1">
          <Button variant="outline" size="sm" disabled={meta.page <= 1} onClick={() => onPage(meta.page - 1)}>Previous</Button>
          <Button variant="outline" size="sm" disabled={meta.page >= meta.total_pages} onClick={() => onPage(meta.page + 1)}>Next</Button>
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
  confirmLabel = 'Confirm',
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
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4">
      <Card className="w-full max-w-md p-5">
        <div className="flex items-start justify-between gap-4">
          <h3 className="text-lg font-semibold text-slate-900">{title}</h3>
          <button onClick={onCancel} className="text-slate-400 hover:text-slate-600" aria-label="Close"><X className="h-5 w-5" /></button>
        </div>
        <div className="mt-3 text-sm text-slate-600">{children}</div>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="outline" onClick={onCancel}>Cancel</Button>
          <Button variant={danger ? 'danger' : 'primary'} loading={loading} onClick={onConfirm}>{confirmLabel}</Button>
        </div>
      </Card>
    </div>
  )
}

export function FilterBar({ children }: { children: ReactNode }) {
  return <div className="flex flex-wrap items-end gap-3 border-b border-slate-200 px-4 py-3">{children}</div>
}

export function SearchInput({ value, onChange, placeholder }: { value: string; onChange: (v: string) => void; placeholder?: string }) {
  return (
    <div className="relative w-64">
      <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
      <Input value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder ?? 'Search…'} className="h-9 pl-8" />
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
  const label = from && to ? `${from} → ${to}` : from ? `From ${from}` : to ? `Until ${to}` : 'Any date'

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex h-9 items-center gap-2 rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-700 hover:bg-slate-50"
      >
        <CalendarDays className="h-4 w-4 text-slate-400" />
        {label}
      </button>
      {open && (
        <div className="absolute right-0 z-50 mt-2 w-max rounded-lg border border-slate-200 bg-white p-3 shadow-lg">
          <div className="mb-2 flex flex-wrap gap-1">
            <Button variant="ghost" size="sm" onClick={() => preset(1)}>Today</Button>
            <Button variant="ghost" size="sm" onClick={() => preset(7)}>7d</Button>
            <Button variant="ghost" size="sm" onClick={() => preset(30)}>30d</Button>
            <Button variant="ghost" size="sm" onClick={presetMonth}>This month</Button>
            <Button variant="ghost" size="sm" onClick={() => { onFrom(''); onTo('') }}>Clear</Button>
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
  value,
  onChange,
  disabled,
}: {
  value: Product | null
  onChange: (p: Product | null) => void
  disabled?: boolean
}) {
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
      <div className="flex h-10 items-center justify-between rounded-md border border-slate-300 bg-white px-3 text-sm">
        <span><span className="font-medium">{value.sku}</span> — {value.name} ({formatQty(value.current_stock)} {value.unit.code})</span>
        {!disabled && <button onClick={() => onChange(null)} aria-label="Clear" className="text-slate-400 hover:text-slate-600"><X className="h-4 w-4" /></button>}
      </div>
    )
  }

  return (
    <div className="relative">
      <Input
        value={query}
        disabled={disabled}
        onFocus={() => setOpen(true)}
        onChange={(e) => { setQuery(e.target.value); setOpen(true) }}
        placeholder="Search by SKU or product name…"
      />
      {open && (
        <div className="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-md border border-slate-200 bg-white shadow-lg">
          {loading && <div className="px-3 py-2 text-sm text-slate-500">Searching…</div>}
          {!loading && rows.length === 0 && <div className="px-3 py-2 text-sm text-slate-500">No matching products</div>}
          {!loading && rows.filter((r) => r.is_active).map((p) => (
            <button
              key={p.id}
              type="button"
              className="flex w-full items-center justify-between px-3 py-2 text-left text-sm hover:bg-slate-50"
              onClick={() => { onChange(p); setOpen(false); setQuery('') }}
            >
              <span><span className="font-medium">{p.sku}</span> — {p.name}</span>
              <span className="text-xs text-slate-500">{formatQty(p.current_stock)} {p.unit.code}</span>
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
      {terminal && <Badge tone="red">{statusLabel(current)}</Badge>}
    </ol>
  )
}
