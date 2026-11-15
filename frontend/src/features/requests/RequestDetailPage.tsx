import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { inventoryApi, requestApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, CardHeader, Input, Textarea } from '../../components/ui'
import { AutoRefreshIndicator, ConfirmDialog, ErrorState, Loading, PageHeader, StatusBadge, WorkflowStepper } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { MONITORING_POLL_MS } from '../../lib/constants'
import { formatDateTime, formatQty } from '../../lib/format'

const terminalStatuses = new Set(['FULFILLED', 'REJECTED', 'CANCELLED'])

export function RequestDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const { t } = useT()
  const queryClient = useQueryClient()
  const [fulfillOpen, setFulfillOpen] = useState(false)
  const [rejectOpen, setRejectOpen] = useState(false)
  const [cancelOpen, setCancelOpen] = useState(false)
  const [issued, setIssued] = useState<Record<string, number>>({})
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const dialogOpen = fulfillOpen || rejectOpen || cancelOpen
  const request = useQuery({
    queryKey: ['request', id],
    queryFn: () => requestApi.get(id!),
    refetchInterval: (query) => {
      const status = query.state.data?.data?.status
      if (!status || terminalStatuses.has(status) || dialogOpen) return false
      return MONITORING_POLL_MS
    },
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    placeholderData: keepPreviousData,
  })
  const balances = useQuery({
    queryKey: ['inventory', { fulfill: id }],
    queryFn: () => inventoryApi.balances({ page_size: 50 }),
    enabled: user?.role === 'WAREHOUSE_ADMIN',
  })

  const req = request.data?.data
  useEffect(() => {
    if (req) {
      const next: Record<string, number> = {}
      for (const item of req.items) next[item.id] = item.remaining_qty
      setIssued(next)
    }
  }, [req])

  if (request.isLoading) return <Loading />
  if (request.error) return <ErrorState error={request.error} onRetry={() => request.refetch()} />
  if (!req) return null

  const can = (action: string) => req.allowed_actions?.includes(action) ?? false
  const stockByProduct = new Map((balances.data?.items ?? []).map((b) => [b.product_id, b.quantity]))

  const invalidate = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['request', id] }),
      queryClient.invalidateQueries({ queryKey: ['requests'] }),
      queryClient.invalidateQueries({ queryKey: ['inventory'] }),
      queryClient.invalidateQueries({ queryKey: ['movements'] }),
      queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
      queryClient.invalidateQueries({ queryKey: ['notifications'] }),
    ])
  }

  const fulfill = async () => {
    const lines = Object.entries(issued)
      .map(([item_id, qty]) => ({ item_id, issued_quantity: Number(qty) }))
      .filter((l) => l.issued_quantity > 0)
    if (!lines.length) { toast.error(t('validation.atLeastOneLine')); return }
    setSaving(true)
    try {
      await requestApi.fulfill(req.id, { items: lines })
      toast.success(t('toast.requestFulfilled'))
      setFulfillOpen(false)
      await invalidate()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t('toast.fulfilmentFailed'))
    } finally {
      setSaving(false)
    }
  }

  const reject = async () => {
    if (reason.trim().length < 5) { toast.error(t('validation.reasonRequired')); return }
    setSaving(true)
    try {
      await requestApi.reject(req.id, { reason: reason.trim() })
      toast.success(t('toast.requestRejected'))
      setRejectOpen(false)
      setReason('')
      await invalidate()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t('toast.rejectionFailed'))
    } finally {
      setSaving(false)
    }
  }

  const cancel = async () => {
    setSaving(true)
    try {
      await requestApi.cancel(req.id)
      toast.success(t('toast.requestCancelled'))
      setCancelOpen(false)
      await invalidate()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t('toast.cancellationFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div>
      <PageHeader
        title={req.request_number}
        description={t('details.submittedBy', {
          department: req.department.name,
          name: req.requested_by.full_name,
          date: formatDateTime(req.created_at),
        })}
        actions={
          <>
            <AutoRefreshIndicator
              updatedAt={request.dataUpdatedAt}
              intervalMs={MONITORING_POLL_MS}
              isFetching={request.isFetching}
              onRefresh={() => request.refetch()}
            />
            <Link to="/requests"><Button variant="outline">{t('common.back')}</Button></Link>
            {can('reject') && <Button variant="danger" onClick={() => setRejectOpen(true)}>{t('actions.reject')}</Button>}
            {can('fulfill') && <Button onClick={() => setFulfillOpen(true)}>{t('actions.fulfill')}</Button>}
            {can('cancel') && <Button variant="danger" onClick={() => setCancelOpen(true)}>{t('actions.cancel')}</Button>}
          </>
        }
      />
      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="p-5 lg:col-span-2">
          <WorkflowStepper
            current={req.status}
            steps={[
              { key: 'PENDING', label: t('status.PENDING') },
              { key: 'PARTIALLY_FULFILLED', label: t('status.PARTIALLY_FULFILLED') },
              { key: 'FULFILLED', label: t('status.FULFILLED') },
            ]}
          />
          <dl className="grid gap-4 sm:grid-cols-2">
            <div><dt className="text-xs uppercase text-slate-500">{t('details.status')}</dt><dd className="mt-1"><StatusBadge status={req.status} /></dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.purpose')}</dt><dd className="mt-1">{req.purpose}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.processedBy')}</dt><dd className="mt-1">{req.processed_by?.full_name ?? '—'}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.processedAt')}</dt><dd className="mt-1">{formatDateTime(req.processed_at)}</dd></div>
          </dl>
          {req.rejection_reason && <div className="mt-4 rounded-md bg-red-50 p-3 text-sm text-red-700">{t('details.rejectionReason')}: {req.rejection_reason}</div>}
        </Card>
        <Card>
          <CardHeader title={t('details.items')} description={t('common.lineCount', { count: req.items.length })} />
          <ul className="divide-y divide-slate-100">
            {req.items.map((item) => (
              <li key={item.id} className="px-5 py-3 text-sm">
                <p className="font-medium">{item.product.sku} — {item.product.name}</p>
                <p className="text-slate-500">
                  {t('details.requested')} {formatQty(item.requested_qty)} {item.product.unit ?? ''} · {t('details.issued')} {formatQty(item.issued_qty)} · {t('details.remaining')} {formatQty(item.remaining_qty)}
                </p>
              </li>
            ))}
          </ul>
        </Card>
      </div>

      <ConfirmDialog
        open={cancelOpen}
        title={t('dialogs.cancelRequest')}
        onCancel={() => setCancelOpen(false)}
        onConfirm={cancel}
        loading={saving}
        danger
        confirmLabel={t('actions.cancel')}
      >
        {t('dialogs.cancelRequestBody')}
      </ConfirmDialog>

      {rejectOpen && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center">
          <Card className="max-h-[90dvh] w-full max-w-md overflow-y-auto p-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))] sm:pb-5">
            <h3 className="text-lg font-semibold">{t('dialogs.rejectRequest')}</h3>
            <p className="mt-1 text-sm text-slate-500">{t('dialogs.rejectRequestBody')}</p>
            <Textarea className="mt-3" value={reason} onChange={(e) => setReason(e.target.value)} placeholder={t('dialogs.reasonPlaceholder')} />
            <div className="mt-4 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end [&>*]:w-full sm:[&>*]:w-auto">
              <Button variant="outline" onClick={() => setRejectOpen(false)}>{t('common.cancel')}</Button>
              <Button variant="danger" loading={saving} onClick={reject}>{t('actions.reject')}</Button>
            </div>
          </Card>
        </div>
      )}

      {fulfillOpen && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center">
          <Card className="max-h-[90dvh] w-full max-w-2xl overflow-y-auto p-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))] sm:pb-5">
            <h3 className="text-lg font-semibold">{t('dialogs.fulfilRequest')}</h3>
            <p className="mt-1 text-sm text-slate-500">{t('dialogs.fulfilRequestBody')}</p>
            <div className="mt-4 space-y-3">
              {req.items.map((item) => (
                <div key={item.id} className="grid items-center gap-3 rounded-lg border border-slate-200 p-3 sm:grid-cols-[1fr_120px]">
                  <div className="text-sm">
                    <p className="font-medium">{item.product.sku} — {item.product.name}</p>
                    <p className="text-slate-500">
                      {t('details.remaining')} {formatQty(item.remaining_qty)} · {t('details.available')} {formatQty(stockByProduct.get(item.product.id) ?? 0)}
                    </p>
                  </div>
                  <Input
                    type="number"
                    min={0}
                    max={item.remaining_qty}
                    value={issued[item.id] ?? 0}
                    onChange={(e) => setIssued((prev) => ({ ...prev, [item.id]: Math.max(0, Math.min(item.remaining_qty, Number(e.target.value) || 0)) }))}
                  />
                </div>
              ))}
            </div>
            <div className="mt-4 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end [&>*]:w-full sm:[&>*]:w-auto">
              <Button variant="outline" onClick={() => setFulfillOpen(false)}>{t('common.cancel')}</Button>
              <Button loading={saving} onClick={fulfill}>{t('dialogs.fulfilRequest')}</Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  )
}
