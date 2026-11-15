import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil } from 'lucide-react'
import { toast } from 'sonner'
import { inventoryApi, productApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Badge, Button, Card, CardHeader } from '../../components/ui'
import { ConfirmDialog, DataTable, ErrorState, Loading, PageHeader, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { formatCurrency, formatDateTime, formatQty } from '../../lib/format'
import { AuditEventTable } from '../audit/AuditEventTable'
import type { Movement } from '../../types'

export function ProductDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const canEdit = user?.role === 'WAREHOUSE_ADMIN' || user?.role === 'SUPERVISOR'
  const product = useQuery({ queryKey: ['product', id], queryFn: () => productApi.get(id!) })
  const movements = useQuery({ queryKey: ['movements', { product_id: id }], queryFn: () => inventoryApi.movements({ product_id: id, page_size: 10 }) })
  const history = useQuery({
    queryKey: ['product-history', id],
    queryFn: () => productApi.history(id!, { page: 1, page_size: 20 }),
    enabled: Boolean(id) && canEdit,
  })

  if (product.isLoading) return <Loading />
  if (product.error) return <ErrorState error={product.error} onRetry={() => product.refetch()} />
  const p = product.data!.data
  const isAdmin = user?.role === 'WAREHOUSE_ADMIN'
  const isSupervisor = user?.role === 'SUPERVISOR'

  const toggleActive = async () => {
    setSaving(true)
    try {
      await productApi.update(p.id, { is_active: !p.is_active })
      toast.success(p.is_active ? t('toast.productDeactivated') : t('toast.productActivated'))
      await queryClient.invalidateQueries({ queryKey: ['product', id] })
      await queryClient.invalidateQueries({ queryKey: ['products'] })
      await queryClient.invalidateQueries({ queryKey: ['product-history', id] })
      setConfirmOpen(false)
    } catch {
      toast.error(t('toast.productUpdateFailed'))
    } finally {
      setSaving(false)
    }
  }

  const columns: Column<Movement>[] = [
    { key: 'when', header: t('details.when'), render: (m) => formatDateTime(m.created_at) },
    { key: 'type', header: t('details.type'), render: (m) => <span className={m.movement_type === 'IN' ? 'text-emerald-600' : 'text-red-600'}>{m.movement_type}</span> },
    { key: 'qty', header: t('details.change'), render: (m) => `${m.quantity_delta > 0 ? '+' : ''}${formatQty(m.quantity_delta)}` },
    { key: 'balance', header: t('details.balance'), render: (m) => formatQty(m.balance_after) },
    { key: 'ref', header: t('details.reference'), render: (m) => m.reference_number },
    { key: 'actor', header: t('details.by'), render: (m) => m.actor.full_name },
  ]

  return (
    <div>
      <PageHeader
        title={`${p.sku} — ${p.name}`}
        description={p.description ?? undefined}
        actions={
          <>
            <Link to={`/inventory/${p.id}`}><Button variant="outline">{t('actions.viewInventory')}</Button></Link>
            {canEdit && <Link to={`/products/${p.id}/edit`}><Button variant="secondary"><Pencil className="h-4 w-4" /> {isSupervisor && !isAdmin ? t('actions.setCost') : t('actions.edit')}</Button></Link>}
            {isAdmin && <Button variant={p.is_active ? 'danger' : 'primary'} onClick={() => setConfirmOpen(true)}>{p.is_active ? t('actions.deactivate') : t('actions.activate')}</Button>}
          </>
        }
      />
      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="p-5 lg:col-span-2">
          <dl className="grid gap-4 sm:grid-cols-2">
            <div><dt className="text-xs uppercase text-slate-500">{t('details.currentStock')}</dt><dd className="text-2xl font-semibold">{formatQty(p.current_stock)} {p.unit.code}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.minimumStock')}</dt><dd className="text-lg">{formatQty(p.min_stock)}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.category')}</dt><dd>{p.category.name}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.unit')}</dt><dd>{p.unit.name}</dd></div>
            {isSupervisor && <div><dt className="text-xs uppercase text-slate-500">{t('details.unitCost')}</dt><dd>{formatCurrency(p.unit_cost)}</dd></div>}
            {isSupervisor && <div><dt className="text-xs uppercase text-slate-500">{t('details.stockValue')}</dt><dd>{formatCurrency((p.unit_cost ?? 0) * p.current_stock)}</dd></div>}
            <div><dt className="text-xs uppercase text-slate-500">{t('details.status')}</dt><dd><Badge tone={p.is_active ? 'green' : 'gray'}>{p.is_active ? t('common.active') : t('common.inactive')}</Badge></dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.lowStock')}</dt><dd>{p.is_low_stock ? <span className="text-amber-700">{t('common.yes')}</span> : t('common.no')}</dd></div>
          </dl>
        </Card>
        <Card>
          <CardHeader title={t('details.quickActions')} />
          <div className="space-y-2 p-4">
            <Button className="w-full" variant="outline" onClick={() => navigate('/receiving')} disabled={!isAdmin}>{t('actions.receiveStock')}</Button>
            <Button className="w-full" variant="outline" onClick={() => navigate('/adjustments')} disabled={!isAdmin}>{t('actions.adjustStock')}</Button>
            <Button className="w-full" variant="outline" onClick={() => navigate('/requests/new')} disabled={user?.role !== 'DEPARTMENT_USER'}>{t('actions.requestProduct')}</Button>
          </div>
        </Card>
      </div>
      <Card className="mt-4">
        <CardHeader title={t('details.recentMovements')} action={<Link className="text-sm text-brand-600" to={`/inventory/${p.id}`}>{t('common.view')}</Link>} />
        <DataTable columns={columns} rows={movements.data?.items ?? []} loading={movements.isLoading} error={movements.error} emptyTitle={t('empty.noMovements')} />
      </Card>
      {canEdit && (
        <Card className="mt-4">
          <CardHeader title={t('audit.productHistory')} description={t('audit.productHistoryDescription')} />
          <AuditEventTable
            events={history.data?.items ?? []}
            loading={history.isLoading}
            error={history.error}
            onRetry={() => history.refetch()}
            showSubject={false}
          />
        </Card>
      )}
      <ConfirmDialog
        open={confirmOpen}
        title={p.is_active ? t('dialogs.deactivateProduct') : t('dialogs.activateProduct')}
        onCancel={() => setConfirmOpen(false)}
        onConfirm={toggleActive}
        loading={saving}
        danger={p.is_active}
        confirmLabel={p.is_active ? t('actions.deactivate') : t('actions.activate')}
      >
        {p.is_active ? t('dialogs.deactivateProductBody') : t('dialogs.activateProductBody')}
      </ConfirmDialog>
    </div>
  )
}
