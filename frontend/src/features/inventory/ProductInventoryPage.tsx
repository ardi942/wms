import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { inventoryApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Badge, Button, Card, CardHeader } from '../../components/ui'
import { DataTable, ErrorState, Loading, PageHeader, Pagination, RangeDatePicker, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { formatCurrency, formatDateTime, formatQty } from '../../lib/format'
import type { Movement } from '../../types'

export function ProductInventoryPage() {
  const { user } = useAuth()
  const { t } = useT()
  const { productId } = useParams()
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const filters = { product_id: productId, date_from: from, date_to: to, page, page_size: pageSize }
  const balance = useQuery({ queryKey: ['inventoryProduct', productId], queryFn: () => inventoryApi.product(productId!) })
  const movements = useQuery({ queryKey: ['movements', filters], queryFn: () => inventoryApi.movements(filters) })

  if (balance.isLoading) return <Loading />
  if (balance.error) return <ErrorState error={balance.error} onRetry={() => balance.refetch()} />
  const b = balance.data!.data

  const columns: Column<Movement>[] = [
    { key: 'when', header: t('details.when'), render: (m) => formatDateTime(m.created_at) },
    { key: 'type', header: t('details.type'), render: (m) => <span className={m.movement_type === 'IN' ? 'text-emerald-600' : 'text-red-600'}>{m.movement_type}</span> },
    { key: 'qty', header: t('details.change'), render: (m) => `${m.quantity_delta > 0 ? '+' : ''}${formatQty(m.quantity_delta)}` },
    { key: 'balance', header: t('details.balanceAfter'), render: (m) => formatQty(m.balance_after) },
    { key: 'ref', header: t('details.reference'), render: (m) => m.reference_number },
    { key: 'dept', header: t('details.department'), render: (m) => m.department?.name ?? '—' },
    { key: 'by', header: t('details.by'), render: (m) => m.actor.full_name },
  ]

  return (
    <div>
      <PageHeader
        title={`${b.sku} — ${b.name}`}
        description={t('pages.productInventoryDescription')}
        actions={<Link to={`/products/${b.product_id}`}><Button variant="outline">{t('actions.productDetail')}</Button></Link>}
      />
      <div className="mb-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card className="p-4"><p className="text-xs uppercase text-slate-500">{t('details.quantity')}</p><p className="text-2xl font-semibold">{formatQty(b.quantity)} {b.unit}</p></Card>
        <Card className="p-4"><p className="text-xs uppercase text-slate-500">{t('details.minimum')}</p><p className="text-2xl font-semibold">{formatQty(b.min_stock)}</p></Card>
        {user?.role === 'SUPERVISOR' && <Card className="p-4"><p className="text-xs uppercase text-slate-500">{t('details.stockValue')}</p><p className="text-2xl font-semibold">{formatCurrency(b.stock_value)}</p></Card>}
        <Card className="p-4"><p className="text-xs uppercase text-slate-500">{t('details.stockHealth')}</p><div className="mt-1"><Badge tone={b.is_low_stock ? 'amber' : 'green'}>{b.is_low_stock ? t('common.lowStock') : t('common.ok')}</Badge></div></Card>
      </div>
      <Card>
        <CardHeader title={t('details.movementHistory')} action={<RangeDatePicker from={from} to={to} onFrom={(v) => { setFrom(v); setPage(1) }} onTo={(v) => { setTo(v); setPage(1) }} />} />
        <DataTable columns={columns} rows={movements.data?.items ?? []} loading={movements.isLoading} error={movements.error} onRetry={() => movements.refetch()} emptyTitle={t('empty.noMovementsInRange')} />
        {movements.data && <Pagination meta={movements.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
