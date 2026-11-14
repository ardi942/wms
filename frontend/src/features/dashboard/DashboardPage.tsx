import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { ArrowDownToLine, ArrowUpFromLine, ClipboardList, Package, TriangleAlert, Wallet } from 'lucide-react'
import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { reportApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Card, CardHeader } from '../../components/ui'
import { ErrorState, Loading, PageHeader } from '../../components/common'
import { formatCurrency, formatDateTime, formatQty } from '../../lib/format'
import type { ChartPoint, Dashboard, LowStockItem, Movement, PendingPreview, UsageReportRow } from '../../types'

function KPI({ label, value, icon: Icon, tone }: { label: string; value: string; icon: typeof Package; tone: string }) {
  return (
    <Card className="flex items-center gap-4 p-4">
      <div className={`grid h-10 w-10 place-items-center rounded-lg ${tone}`}>
        <Icon className="h-5 w-5" />
      </div>
      <div>
        <p className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</p>
        <p className="text-xl font-semibold text-slate-900">{value}</p>
      </div>
    </Card>
  )
}

function MovementRows({ rows }: { rows: Movement[] }) {
  if (!rows.length) return <p className="px-5 py-6 text-sm text-slate-500">No recent movements.</p>
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[560px] text-left text-sm">
        <thead className="border-b border-slate-200 bg-slate-50 text-xs uppercase text-slate-500">
          <tr><th className="px-4 py-2">Product</th><th className="px-4 py-2">Type</th><th className="px-4 py-2">Qty</th><th className="px-4 py-2">Reference</th><th className="px-4 py-2">When</th></tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {rows.map((m) => (
            <tr key={m.id}>
              <td className="px-4 py-2">{m.product.sku} — {m.product.name}</td>
              <td className="px-4 py-2"><span className={m.movement_type === 'IN' ? 'text-emerald-600' : 'text-red-600'}>{m.movement_type}</span></td>
              <td className="px-4 py-2">{m.quantity_delta > 0 ? '+' : ''}{formatQty(m.quantity_delta)}</td>
              <td className="px-4 py-2">{m.reference_number}</td>
              <td className="px-4 py-2 text-slate-500">{formatDateTime(m.created_at)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function Chart({ data }: { data: ChartPoint[] }) {
  return (
    <div className="h-64 p-4">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 5, right: 10, left: -20, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
          <XAxis dataKey="date" tick={{ fontSize: 11 }} tickFormatter={(v: string) => v.slice(5)} />
          <YAxis tick={{ fontSize: 11 }} />
          <Tooltip />
          <Legend />
          <Line type="monotone" dataKey="in_units" name="In" stroke="#059669" strokeWidth={2} dot={false} />
          <Line type="monotone" dataKey="out_units" name="Out" stroke="#dc2626" strokeWidth={2} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}

function Supervisor({ data }: { data: Dashboard }) {
  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KPI label="Active products" value={formatQty(data.total_products)} icon={Package} tone="bg-blue-50 text-blue-700" />
        <KPI label="Stock units" value={formatQty(data.total_stock_units)} icon={Package} tone="bg-emerald-50 text-emerald-700" />
        <KPI label="Inventory value" value={formatCurrency(Number(data.inventory_value ?? 0))} icon={Wallet} tone="bg-violet-50 text-violet-700" />
        <KPI label="Low stock" value={formatQty(data.low_stock_count)} icon={TriangleAlert} tone="bg-amber-50 text-amber-700" />
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <Card className="p-4"><p className="text-sm text-slate-500">Incoming today</p><p className="text-2xl font-semibold">{formatQty((data.incoming as { today_units: number })?.today_units)}</p></Card>
        <Card className="p-4"><p className="text-sm text-slate-500">Outgoing today</p><p className="text-2xl font-semibold">{formatQty((data.outgoing as { today_units: number })?.today_units)}</p></Card>
      </div>
      <Card>
        <CardHeader title="Movement trend" description="Units in and out over the last 14 days" />
        <Chart data={data.movement_chart ?? []} />
      </Card>
      <Card>
        <CardHeader title="Recent movements" />
        <MovementRows rows={data.recent_movements ?? []} />
      </Card>
    </div>
  )
}

function Admin({ data }: { data: Dashboard }) {
  const low = (data.low_stock ?? []) as LowStockItem[]
  const preview = (data.pending_preview ?? []) as PendingPreview[]
  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KPI label="Active products" value={formatQty(data.total_products)} icon={Package} tone="bg-blue-50 text-blue-700" />
        <KPI label="Stock units" value={formatQty(data.total_stock_units)} icon={Package} tone="bg-emerald-50 text-emerald-700" />
        <KPI label="Low stock" value={formatQty(data.low_stock_count)} icon={TriangleAlert} tone="bg-amber-50 text-amber-700" />
        <KPI label="Pending requests" value={formatQty(data.pending_requests)} icon={ClipboardList} tone="bg-violet-50 text-violet-700" />
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader title="Low stock attention" action={<Link className="text-sm text-brand-600" to="/reports?tab=low-stock">View report</Link>} />
          {low.length === 0 ? <p className="px-5 py-6 text-sm text-slate-500">All products are above their minimum.</p> : (
            <ul className="divide-y divide-slate-100">
              {low.map((p) => (
                <li key={p.product_id} className="flex items-center justify-between px-5 py-3 text-sm">
                  <span><span className="font-medium">{p.sku}</span> — {p.name}</span>
                  <span className="text-amber-700">{formatQty(p.quantity)} / {formatQty(p.min_stock)}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>
        <Card>
          <CardHeader title="Pending request preview" action={<Link className="text-sm text-brand-600" to="/requests?status=PENDING">View all</Link>} />
          {preview.length === 0 ? <p className="px-5 py-6 text-sm text-slate-500">No pending requests.</p> : (
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
      <Card>
        <CardHeader title="Today's activity" />
        <MovementRows rows={data.recent_movements ?? []} />
      </Card>
    </div>
  )
}

function Department({ data }: { data: Dashboard }) {
  const usage = (data.usage_last_30d ?? []) as UsageReportRow[]
  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KPI label="Available products" value={formatQty(data.available_products)} icon={Package} tone="bg-blue-50 text-blue-700" />
        <KPI label="My pending requests" value={formatQty(data.my_pending_requests)} icon={ClipboardList} tone="bg-amber-50 text-amber-700" />
        <KPI label="Fulfilled" value={formatQty(data.my_fulfilled_requests)} icon={ArrowUpFromLine} tone="bg-emerald-50 text-emerald-700" />
        <KPI label="Partially fulfilled" value={formatQty(data.my_partially_fulfilled_requests)} icon={ArrowDownToLine} tone="bg-violet-50 text-violet-700" />
      </div>
      <Card>
        <CardHeader title="My department usage (30 days)" action={<Link className="text-sm text-brand-600" to="/reports?tab=department-usage">View report</Link>} />
        {usage.length === 0 ? <p className="px-5 py-6 text-sm text-slate-500">No usage yet.</p> : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[560px] text-left text-sm">
              <thead className="border-b border-slate-200 bg-slate-50 text-xs uppercase text-slate-500"><tr><th className="px-4 py-2">Product</th><th className="px-4 py-2">Issued</th><th className="px-4 py-2">Returned</th><th className="px-4 py-2">Net used</th></tr></thead>
              <tbody className="divide-y divide-slate-100">
                {usage.map((u) => (
                  <tr key={`${u.sku}-${u.department}`}><td className="px-4 py-2">{u.sku} — {u.product}</td><td className="px-4 py-2">{formatQty(u.issued_units)}</td><td className="px-4 py-2">{formatQty(u.returned_units)}</td><td className="px-4 py-2">{formatQty(u.net_used)}</td></tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
      <Card>
        <CardHeader title="Recent department movements" />
        <MovementRows rows={data.recent_department_movements ?? []} />
      </Card>
    </div>
  )
}

export function DashboardPage() {
  const { user } = useAuth()
  const query = useQuery({ queryKey: ['dashboard'], queryFn: async () => (await reportApi.dashboard()).data })
  if (query.isLoading) return <Loading />
  if (query.error) return <ErrorState error={query.error} onRetry={() => query.refetch()} />
  const data = query.data as Dashboard
  return (
    <div>
      <PageHeader title={`Welcome, ${user?.full_name.split(' ')[0]}`} description="Here is the current state of the warehouse." />
      {user?.role === 'SUPERVISOR' && <Supervisor data={data} />}
      {user?.role === 'WAREHOUSE_ADMIN' && <Admin data={data} />}
      {user?.role === 'DEPARTMENT_USER' && <Department data={data} />}
    </div>
  )
}
