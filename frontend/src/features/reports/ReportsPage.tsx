import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Download } from 'lucide-react'
import { downloadCsv, saveBlob } from '../../api/client'
import { referenceApi, reportApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, Select } from '../../components/ui'
import { DataTable, FilterBar, Pagination, PageHeader, RangeDatePicker, SearchInput, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { cn } from '../../lib/utils'
import { formatCurrency, formatDate, formatDateTime, formatQty } from '../../lib/format'
import type { InventoryReportRow, ListResult, LowStockReportRow, Movement, UsageReportRow } from '../../types'

type Tab = 'inventory' | 'movements' | 'department-usage' | 'low-stock'

const allTabs: Array<{ id: Tab; labelKey: string; roles: string[] }> = [
  { id: 'inventory', labelKey: 'reports.tabInventory', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN'] },
  { id: 'movements', labelKey: 'reports.tabMovements', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN'] },
  { id: 'department-usage', labelKey: 'reports.tabDepartmentUsage', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN', 'DEPARTMENT_USER'] },
  { id: 'low-stock', labelKey: 'reports.tabLowStock', roles: ['SUPERVISOR', 'WAREHOUSE_ADMIN'] },
]

export function ReportsPage() {
  const { user } = useAuth()
  const { t } = useT()
  const [params, setParams] = useSearchParams()
  const tabs = allTabs.filter((tab) => user && tab.roles.includes(user.role))
  const requested = params.get('tab') as Tab | null
  const tab: Tab = tabs.some((item) => item.id === requested) ? (requested as Tab) : tabs[0]?.id ?? 'department-usage'
  const [search, setSearch] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [departmentId, setDepartmentId] = useState('')
  const [movementType, setMovementType] = useState('')
  const [lowStock, setLowStock] = useState(false)
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [exporting, setExporting] = useState(false)

  const categories = useQuery({ queryKey: ['categories'], queryFn: () => referenceApi.categories() })
  const departments = useQuery({ queryKey: ['departments'], queryFn: () => referenceApi.departments() })

  const filters = {
    search, category_id: categoryId, department_id: departmentId,
    movement_type: movementType, low_stock: lowStock || undefined,
    date_from: from, date_to: to, page, page_size: pageSize,
  }

  useEffect(() => { setPage(1) }, [tab, search, categoryId, departmentId, movementType, lowStock, from, to])

  const query = useQuery({
    queryKey: ['reports', tab, filters],
    queryFn: async (): Promise<ListResult<unknown>> => {
      switch (tab) {
        case 'inventory': return (await reportApi.inventory(filters)) as ListResult<unknown>
        case 'movements': return (await reportApi.movements(filters)) as ListResult<unknown>
        case 'department-usage': return (await reportApi.departmentUsage(filters)) as ListResult<unknown>
        case 'low-stock': return (await reportApi.lowStock(filters)) as ListResult<unknown>
      }
    },
  })

  const exportCsv = async () => {
    setExporting(true)
    try {
      const blob = await downloadCsv('/reports/export', { ...filters, report: tab })
      saveBlob(blob, `${tab}-${new Date().toISOString().slice(0, 10)}.csv`)
      toast.success(t('common.exportDownloaded'))
    } catch {
      toast.error(t('common.exportFailed'))
    } finally {
      setExporting(false)
    }
  }

  const isSupervisor = user?.role === 'SUPERVISOR'
  const inventoryColumns: Column<InventoryReportRow>[] = [
    { key: 'sku', header: t('fields.sku'), render: (r) => r.sku },
    { key: 'product', header: t('details.product'), render: (r) => r.product, tooltip: (r) => r.product },
    { key: 'category', header: t('details.category'), render: (r) => r.category },
    { key: 'qty', header: t('details.quantity'), render: (r) => <span className={r.is_low_stock ? 'text-amber-700' : ''}>{formatQty(r.quantity)} {r.unit}</span> },
    { key: 'min', header: t('details.minimum'), render: (r) => formatQty(r.min_stock) },
    ...(isSupervisor ? [
      { key: 'cost', header: t('details.unitCost'), render: (r: InventoryReportRow) => formatCurrency(r.unit_cost) },
      { key: 'value', header: t('details.stockValue'), render: (r: InventoryReportRow) => formatCurrency(r.stock_value) },
    ] : []),
    { key: 'last', header: t('details.lastMovement'), render: (r) => formatDate(r.last_movement_at) },
  ]
  const movementColumns: Column<Movement>[] = [
    { key: 'when', header: t('details.when'), render: (m) => formatDateTime(m.created_at) },
    { key: 'product', header: t('details.product'), render: (m) => `${m.product.sku} — ${m.product.name}`, tooltip: (m) => `${m.product.sku} — ${m.product.name}` },
    { key: 'type', header: t('details.type'), render: (m) => m.movement_type },
    { key: 'qty', header: t('details.change'), render: (m) => `${m.quantity_delta > 0 ? '+' : ''}${formatQty(m.quantity_delta)}` },
    { key: 'balance', header: t('details.balance'), render: (m) => formatQty(m.balance_after) },
    { key: 'ref', header: t('details.reference'), render: (m) => m.reference_number },
    { key: 'dept', header: t('details.department'), render: (m) => m.department?.name ?? '—' },
  ]
  const usageColumns: Column<UsageReportRow>[] = [
    { key: 'dept', header: t('details.department'), render: (r) => r.department },
    { key: 'sku', header: t('fields.sku'), render: (r) => r.sku },
    { key: 'product', header: t('details.product'), render: (r) => r.product, tooltip: (r) => r.product },
    { key: 'issued', header: t('details.issued'), render: (r) => formatQty(r.issued_units) },
    { key: 'returned', header: t('details.returned'), render: (r) => formatQty(r.returned_units) },
    { key: 'net', header: t('details.netUsed'), render: (r) => formatQty(r.net_used) },
    { key: 'last', header: t('details.lastIssued'), render: (r) => formatDateTime(r.last_issued_at) },
  ]
  const lowColumns: Column<LowStockReportRow>[] = [
    { key: 'sku', header: t('fields.sku'), render: (r) => r.sku },
    { key: 'product', header: t('details.product'), render: (r) => r.product, tooltip: (r) => r.product },
    { key: 'category', header: t('details.category'), render: (r) => r.category },
    { key: 'qty', header: t('details.quantity'), render: (r) => formatQty(r.quantity) },
    { key: 'min', header: t('details.minimum'), render: (r) => formatQty(r.min_stock) },
    { key: 'shortage', header: t('details.shortage'), render: (r) => <span className="font-medium text-red-600">{formatQty(r.shortage)}</span> },
    { key: 'last', header: t('details.lastMovement'), render: (r) => formatDate(r.last_movement_at) },
  ]

  const columns = (
    tab === 'inventory' ? inventoryColumns :
    tab === 'movements' ? movementColumns :
    tab === 'department-usage' ? usageColumns : lowColumns
  ) as Column<unknown>[]

  const expandedRow = (row: unknown) => {
    if (tab === 'inventory') {
      const r = row as InventoryReportRow
      return (
        <dl className="grid gap-2 text-sm sm:grid-cols-2">
          <div><dt className="text-xs uppercase text-slate-500">{t('details.category')}</dt><dd>{r.category}</dd></div>
          <div><dt className="text-xs uppercase text-slate-500">{t('details.quantity')}</dt><dd>{formatQty(r.quantity)} {r.unit}</dd></div>
          <div><dt className="text-xs uppercase text-slate-500">{t('details.minimum')}</dt><dd>{formatQty(r.min_stock)}</dd></div>
          <div><dt className="text-xs uppercase text-slate-500">{t('details.lastMovement')}</dt><dd>{formatDate(r.last_movement_at)}</dd></div>
          {isSupervisor && <div><dt className="text-xs uppercase text-slate-500">{t('details.stockValue')}</dt><dd>{formatCurrency(r.stock_value)}</dd></div>}
        </dl>
      )
    }
    if (tab === 'movements') {
      const m = row as Movement
      return (
        <dl className="grid gap-2 text-sm sm:grid-cols-2">
          <div><dt className="text-xs uppercase text-slate-500">{t('details.reference')}</dt><dd>{m.reference_number}</dd></div>
          <div><dt className="text-xs uppercase text-slate-500">{t('details.department')}</dt><dd>{m.department?.name ?? '—'}</dd></div>
          <div><dt className="text-xs uppercase text-slate-500">{t('details.supplier')}</dt><dd>{m.supplier ?? '—'}</dd></div>
          <div><dt className="text-xs uppercase text-slate-500">{t('details.notes')}</dt><dd>{m.notes ?? m.reason ?? '—'}</dd></div>
        </dl>
      )
    }
    if (tab === 'department-usage') {
      const u = row as UsageReportRow
      return (
        <dl className="grid gap-2 text-sm sm:grid-cols-2">
          <div><dt className="text-xs uppercase text-slate-500">{t('details.issued')}</dt><dd>{formatQty(u.issued_units)}</dd></div>
          <div><dt className="text-xs uppercase text-slate-500">{t('details.returned')}</dt><dd>{formatQty(u.returned_units)}</dd></div>
          <div><dt className="text-xs uppercase text-slate-500">{t('details.netUsed')}</dt><dd>{formatQty(u.net_used)}</dd></div>
          <div><dt className="text-xs uppercase text-slate-500">{t('details.lastIssued')}</dt><dd>{formatDateTime(u.last_issued_at)}</dd></div>
        </dl>
      )
    }
    const l = row as LowStockReportRow
    return (
      <dl className="grid gap-2 text-sm sm:grid-cols-2">
        <div><dt className="text-xs uppercase text-slate-500">{t('details.category')}</dt><dd>{l.category}</dd></div>
        <div><dt className="text-xs uppercase text-slate-500">{t('details.quantity')}</dt><dd>{formatQty(l.quantity)}</dd></div>
        <div><dt className="text-xs uppercase text-slate-500">{t('details.minimum')}</dt><dd>{formatQty(l.min_stock)}</dd></div>
        <div><dt className="text-xs uppercase text-slate-500">{t('details.shortage')}</dt><dd>{formatQty(l.shortage)}</dd></div>
        <div><dt className="text-xs uppercase text-slate-500">{t('details.lastMovement')}</dt><dd>{formatDate(l.last_movement_at)}</dd></div>
      </dl>
    )
  }

  const rowKey = (row: unknown, index: number) => {
    if (tab === 'movements') return (row as Movement).id
    if (tab === 'inventory') return (row as InventoryReportRow).sku
    if (tab === 'low-stock') return (row as LowStockReportRow).sku
    const u = row as UsageReportRow
    return `${u.department}-${u.sku}-${index}`
  }

  return (
    <div>
      <PageHeader
        title={t('pages.reports')}
        description={t('pages.reportsDescription')}
        actions={<Button variant="outline" loading={exporting} onClick={exportCsv}><Download className="h-4 w-4" /> {t('common.exportCsv')}</Button>}
      />
      <div className="mb-4 flex flex-wrap gap-2">
        {tabs.map((item) => (
          <button
            key={item.id}
            onClick={() => { setParams({ tab: item.id }); }}
            className={cn('rounded-full px-4 py-1.5 text-sm font-medium transition-colors', tab === item.id ? 'bg-brand-600 text-white' : 'bg-white text-slate-600 hover:bg-slate-200')}
          >
            {t(item.labelKey)}
          </button>
        ))}
      </div>
      <Card>
        <FilterBar>
          {(tab === 'inventory' || tab === 'movements') && <SearchInput value={search} onChange={setSearch} placeholder={t('common.searchProduct')} />}
          {(tab === 'inventory' || tab === 'low-stock') && (
            <Select className="h-9 w-44" value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
              <option value="">{t('common.allCategories')}</option>
              {categories.data?.data.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </Select>
          )}
          {tab === 'inventory' && (
            <label className="flex items-center gap-2 text-sm text-slate-600">
              <input type="checkbox" checked={lowStock} onChange={(e) => setLowStock(e.target.checked)} className="h-4 w-4 rounded border-slate-300" /> {t('filters.lowStockOnly')}
            </label>
          )}
          {tab === 'movements' && (
            <Select className="h-9 w-32" value={movementType} onChange={(e) => setMovementType(e.target.value)}>
              <option value="">{t('common.allTypes')}</option>
              <option value="IN">IN</option>
              <option value="OUT">OUT</option>
            </Select>
          )}
          {tab === 'department-usage' && user?.role !== 'DEPARTMENT_USER' && (
            <Select className="h-9 w-52" value={departmentId} onChange={(e) => setDepartmentId(e.target.value)}>
              <option value="">{t('common.allDepartments')}</option>
              {departments.data?.data.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
            </Select>
          )}
          <RangeDatePicker from={from} to={to} onFrom={setFrom} onTo={setTo} />
        </FilterBar>
        <DataTable columns={columns} rows={(query.data?.items ?? []) as never[]} loading={query.isLoading} error={query.error} onRetry={() => query.refetch()} emptyTitle={t('empty.noReportRows')} getRowKey={rowKey} renderExpanded={expandedRow} />
        {query.data?.meta && <Pagination meta={query.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
