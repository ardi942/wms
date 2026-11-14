import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { ArrowDownToLine, ArrowUpFromLine, ClipboardList, Package, TriangleAlert, Wallet } from 'lucide-react'
import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { reportApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Card, CardHeader } from '../../components/ui'
import { DataTable, ErrorState, Loading, PageHeader, Pagination, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { formatCurrency, formatDateTime, formatQty } from '../../lib/format'
import type { ChartPoint, Dashboard, LowStockItem, Movement, PendingPreview, UsageReportRow } from '../../types'

function KPI({ label, value, icon: Icon, tone }: { label: string; value: string; icon: typeof Package; tone: string }) {
  return (
    <Card className="flex min-w-0 items-center gap-3 p-3 sm:gap-4 sm:p-4">
      <div className={`grid h-9 w-9 shrink-0 place-items-center rounded-lg sm:h-10 sm:w-10 ${tone}`}>
        <Icon className="h-5 w-5" />
      </div>
      <div className="min-w-0">
        <p className="truncate text-[11px] font-medium uppercase tracking-wide text-slate-500 sm:text-xs">{label}</p>
        <p className="truncate text-lg font-semibold text-slate-900 sm:text-xl" title={value}>{value}</p>
      </div>
    </Card>
  )
}

function RecentMovementsCard({ title }: { title: string }) {
  const { t } = useT()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const query = useQuery({
    queryKey: ['dashboard', 'recent-movements', page, pageSize],
    queryFn: () => reportApi.recentMovements({ page, page_size: pageSize }),
  })

  const columns: Column<Movement>[] = [
    { key: 'product', header: t('details.product'), render: (m) => `${m.product.sku} — ${m.product.name}`, tooltip: (m) => `${m.product.sku} — ${m.product.name}` },
    { key: 'type', header: t('details.type'), render: (m) => <span className={m.movement_type === 'IN' ? 'text-emerald-600' : 'text-red-600'}>{m.movement_type}</span> },
    { key: 'qty', header: t('common.quantityShort'), render: (m) => `${m.quantity_delta > 0 ? '+' : ''}${formatQty(m.quantity_delta)}` },
    { key: 'ref', header: t('details.reference'), render: (m) => m.reference_number },
    { key: 'when', header: t('details.when'), render: (m) => formatDateTime(m.created_at) },
  ]

  return (
    <Card>
      <CardHeader title={title} />
      <DataTable
        columns={columns}
        rows={query.data?.items ?? []}
        loading={query.isLoading}
        error={query.error}
        onRetry={() => query.refetch()}
        emptyTitle={t('empty.noRecentMovements')}
        getRowKey={(m) => m.id}
        renderExpanded={(m) => (
          <dl className="grid gap-2 text-sm sm:grid-cols-2">
            <div><dt className="text-xs uppercase text-slate-500">{t('details.reference')}</dt><dd>{m.reference_number}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.department')}</dt><dd>{m.department?.name ?? '—'}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.actor')}</dt><dd>{m.actor.full_name}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.reasonsNotes')}</dt><dd>{m.reason ?? m.notes ?? '—'}</dd></div>
          </dl>
        )}
      />
      {query.data && (
        <Pagination
          meta={query.data.meta}
          onPage={setPage}
          onPageSize={(s) => { setPageSize(s); setPage(1) }}
        />
      )}
    </Card>
  )
}

function Chart({ data }: { data: ChartPoint[] }) {
  const { t } = useT()
  return (
    <div className="h-56 p-4 sm:h-64">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 5, right: 10, left: -20, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
          <XAxis dataKey="date" tick={{ fontSize: 11 }} tickFormatter={(v: string) => v.slice(5)} />
          <YAxis tick={{ fontSize: 11 }} />
          <Tooltip />
          <Legend />
          <Line type="monotone" dataKey="in_units" name={t('dashboard.in')} stroke="#059669" strokeWidth={2} dot={false} />
          <Line type="monotone" dataKey="out_units" name={t('dashboard.out')} stroke="#dc2626" strokeWidth={2} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}

function Supervisor({ data }: { data: Dashboard }) {
  const { t } = useT()
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
        <KPI label={t('dashboard.activeProducts')} value={formatQty(data.total_products)} icon={Package} tone="bg-blue-50 text-blue-700" />
        <KPI label={t('dashboard.stockUnits')} value={formatQty(data.total_stock_units)} icon={Package} tone="bg-emerald-50 text-emerald-700" />
        <KPI label={t('dashboard.inventoryValue')} value={formatCurrency(Number(data.inventory_value ?? 0))} icon={Wallet} tone="bg-violet-50 text-violet-700" />
        <KPI label={t('dashboard.lowStock')} value={formatQty(data.low_stock_count)} icon={TriangleAlert} tone="bg-amber-50 text-amber-700" />
      </div>
      <div className="grid grid-cols-2 gap-3 sm:gap-4">
        <KPI label={t('dashboard.incomingToday')} value={formatQty((data.incoming as { today_units: number })?.today_units)} icon={ArrowUpFromLine} tone="bg-emerald-50 text-emerald-700" />
        <KPI label={t('dashboard.outgoingToday')} value={formatQty((data.outgoing as { today_units: number })?.today_units)} icon={ArrowDownToLine} tone="bg-red-50 text-red-700" />
      </div>
      <Card>
        <CardHeader title={t('dashboard.movementTrend')} description={t('dashboard.movementTrendDescription')} />
        <Chart data={data.movement_chart ?? []} />
      </Card>
      <RecentMovementsCard title={t('dashboard.recentMovements')} />
    </div>
  )
}

function Admin({ data }: { data: Dashboard }) {
  const { t } = useT()
  const low = (data.low_stock ?? []) as LowStockItem[]
  const preview = (data.pending_preview ?? []) as PendingPreview[]
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
        <KPI label={t('dashboard.activeProducts')} value={formatQty(data.total_products)} icon={Package} tone="bg-blue-50 text-blue-700" />
        <KPI label={t('dashboard.stockUnits')} value={formatQty(data.total_stock_units)} icon={Package} tone="bg-emerald-50 text-emerald-700" />
        <KPI label={t('dashboard.lowStock')} value={formatQty(data.low_stock_count)} icon={TriangleAlert} tone="bg-amber-50 text-amber-700" />
        <KPI label={t('dashboard.pendingRequests')} value={formatQty(data.pending_requests)} icon={ClipboardList} tone="bg-violet-50 text-violet-700" />
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader title={t('dashboard.lowStockAttention')} action={<Link className="text-sm text-brand-600" to="/reports?tab=low-stock">{t('dashboard.viewReport')}</Link>} />
          {low.length === 0 ? <p className="px-5 py-6 text-sm text-slate-500">{t('empty.allAboveMinimum')}</p> : (
            <ul className="divide-y divide-slate-100">
              {low.map((p) => (
                <li key={p.product_id} className="flex items-start justify-between gap-3 px-5 py-3 text-sm">
                  <span className="min-w-0 break-words"><span className="font-medium">{p.sku}</span> — {p.name}</span>
                  <span className="shrink-0 text-right text-amber-700">{formatQty(p.quantity)} / {formatQty(p.min_stock)}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>
        <Card>
          <CardHeader title={t('dashboard.pendingRequestPreview')} action={<Link className="text-sm text-brand-600" to="/requests?status=PENDING">{t('dashboard.viewAll')}</Link>} />
          {preview.length === 0 ? <p className="px-5 py-6 text-sm text-slate-500">{t('empty.noPendingRequests')}</p> : (
            <ul className="divide-y divide-slate-100">
              {preview.map((r) => (
                <li key={r.id} className="px-5 py-3 text-sm">
                  <Link to={`/requests/${r.id}`} className="font-medium text-brand-600">{r.request_number}</Link>
                  <span className="text-slate-500"> · {r.department}</span>
                  <p className="truncate text-slate-600">{r.purpose}</p>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
      <RecentMovementsCard title={t('dashboard.todayActivity')} />
    </div>
  )
}

function Department({ data }: { data: Dashboard }) {
  const { t } = useT()
  const low = (data.low_stock ?? []) as LowStockItem[]
  const usage = (data.usage_last_30d ?? []) as UsageReportRow[]
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
        <KPI label={t('dashboard.availableProducts')} value={formatQty(data.available_products)} icon={Package} tone="bg-blue-50 text-blue-700" />
        <KPI label={t('dashboard.myPendingRequests')} value={formatQty(data.my_pending_requests)} icon={ClipboardList} tone="bg-amber-50 text-amber-700" />
        <KPI label={t('dashboard.fulfilled')} value={formatQty(data.my_fulfilled_requests)} icon={ArrowUpFromLine} tone="bg-emerald-50 text-emerald-700" />
        <KPI label={t('dashboard.partiallyFulfilled')} value={formatQty(data.my_partially_fulfilled_requests)} icon={ArrowDownToLine} tone="bg-violet-50 text-violet-700" />
      </div>
      <Card>
        <CardHeader title={t('dashboard.lowStockAttention')} action={<Link className="text-sm text-brand-600" to="/reports?tab=low-stock">{t('dashboard.viewReport')}</Link>} />
        {low.length === 0 ? <p className="px-5 py-6 text-sm text-slate-500">{t('empty.allAboveMinimum')}</p> : (
          <ul className="divide-y divide-slate-100">
            {low.map((p) => (
              <li key={p.product_id} className="flex items-start justify-between gap-3 px-5 py-3 text-sm">
                <span className="min-w-0 break-words"><span className="font-medium">{p.sku}</span> — {p.name}</span>
                <span className="shrink-0 text-right text-amber-700">{formatQty(p.quantity)} / {formatQty(p.min_stock)}</span>
              </li>
            ))}
          </ul>
        )}
      </Card>
      <Card>
        <CardHeader title={t('dashboard.myDepartmentUsage')} action={<Link className="text-sm text-brand-600" to="/reports?tab=department-usage">{t('dashboard.viewReport')}</Link>} />
        {usage.length === 0 ? <p className="px-5 py-6 text-sm text-slate-500">{t('empty.noUsageYet')}</p> : (
          <DataTable
            columns={[
              { key: 'product', header: t('details.product'), render: (u) => `${u.sku} — ${u.product}` },
              { key: 'issued', header: t('details.issued'), render: (u) => formatQty(u.issued_units) },
              { key: 'returned', header: t('details.returned'), render: (u) => formatQty(u.returned_units) },
              { key: 'net', header: t('details.netUsed'), render: (u) => formatQty(u.net_used) },
            ]}
            rows={usage}
            getRowKey={(u) => `${u.sku}-${u.department}`}
          />
        )}
      </Card>
      <RecentMovementsCard title={t('dashboard.recentDepartmentMovements')} />
    </div>
  )
}

export function DashboardPage() {
  const { user } = useAuth()
  const { t } = useT()
  const query = useQuery({
    queryKey: ['dashboard'],
    queryFn: async () => (await reportApi.dashboard()).data,
  })
  if (query.isLoading) return <Loading />
  if (query.error) return <ErrorState error={query.error} onRetry={() => query.refetch()} />
  const data = query.data as Dashboard
  return (
    <div>
      <PageHeader
        title={t('pages.dashboardWelcome', { name: user?.full_name.split(' ')[0] ?? '' })}
        description={t('pages.dashboardDescription')}
      />
      {user?.role === 'SUPERVISOR' && <Supervisor data={data} />}
      {user?.role === 'WAREHOUSE_ADMIN' && <Admin data={data} />}
      {user?.role === 'DEPARTMENT_USER' && <Department data={data} />}
    </div>
  )
}
