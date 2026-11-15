import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { returnApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { Button, Card, CardHeader, Input, Textarea } from '../../components/ui'
import { AutoRefreshIndicator, ConfirmDialog, ErrorState, Loading, PageHeader, StatusBadge, WorkflowStepper } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { MONITORING_POLL_MS } from '../../lib/constants'
import { formatDateTime, formatQty } from '../../lib/format'

const terminalStatuses = new Set(['CONFIRMED', 'REJECTED', 'CANCELLED'])

export function ReturnDetailPage() {
  const { id } = useParams()
  const { t } = useT()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [rejectOpen, setRejectOpen] = useState(false)
  const [cancelOpen, setCancelOpen] = useState(false)
  const [accepted, setAccepted] = useState<Record<string, number>>({})
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const dialogOpen = confirmOpen || rejectOpen || cancelOpen
  const ret = useQuery({
    queryKey: ['return', id],
    queryFn: () => returnApi.get(id!),
    refetchInterval: (query) => {
      const status = query.state.data?.data?.status
      if (!status || terminalStatuses.has(status) || dialogOpen) return false
      return MONITORING_POLL_MS
    },
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    placeholderData: keepPreviousData,
  })

  const row = ret.data?.data
  useEffect(() => {
    if (row) {
      const next: Record<string, number> = {}
      for (const item of row.items) next[item.id] = item.requested_qty
      setAccepted(next)
    }
  }, [row])

  if (ret.isLoading) return <Loading />
  if (ret.error) return <ErrorState error={ret.error} onRetry={() => ret.refetch()} />
  if (!row) return null

  const can = (action: string) => row.allowed_actions?.includes(action) ?? false

  const invalidate = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['return', id] }),
      queryClient.invalidateQueries({ queryKey: ['returns'] }),
      queryClient.invalidateQueries({ queryKey: ['inventory'] }),
      queryClient.invalidateQueries({ queryKey: ['movements'] }),
      queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
      queryClient.invalidateQueries({ queryKey: ['notifications'] }),
    ])
  }

  const confirm = async () => {
    const lines = row.items.map((item) => ({ item_id: item.id, accepted_qty: Number(accepted[item.id] ?? 0) }))
    if (!lines.some((l) => l.accepted_qty > 0)) { toast.error(t('validation.acceptedGreaterThanZero')); return }
    setSaving(true)
    try {
      await returnApi.confirm(row.id, { items: lines })
      toast.success(t('toast.returnConfirmed'))
      setConfirmOpen(false)
      await invalidate()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t('toast.confirmationFailed'))
    } finally {
      setSaving(false)
    }
  }

  const reject = async () => {
    if (reason.trim().length < 5) { toast.error(t('validation.reasonRequired')); return }
    setSaving(true)
    try {
      await returnApi.reject(row.id, { reason: reason.trim() })
      toast.success(t('toast.returnRejected'))
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
      await returnApi.cancel(row.id)
      toast.success(t('toast.returnCancelled'))
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
        title={row.return_number}
        description={t('details.submittedBy', {
          department: row.department.name,
          name: row.requested_by.full_name,
          date: formatDateTime(row.created_at),
        })}
        actions={
          <>
            <AutoRefreshIndicator
              updatedAt={ret.dataUpdatedAt}
              intervalMs={MONITORING_POLL_MS}
              isFetching={ret.isFetching}
              onRefresh={() => ret.refetch()}
            />
            <Link to="/returns"><Button variant="outline">{t('common.back')}</Button></Link>
            {can('reject') && <Button variant="danger" onClick={() => setRejectOpen(true)}>{t('actions.reject')}</Button>}
            {can('confirm') && <Button onClick={() => setConfirmOpen(true)}>{t('dialogs.confirmReturn')}</Button>}
            {can('cancel') && <Button variant="danger" onClick={() => setCancelOpen(true)}>{t('actions.cancel')}</Button>}
          </>
        }
      />
      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="p-5 lg:col-span-2">
          <WorkflowStepper
            current={row.status}
            steps={[
              { key: 'PENDING', label: t('status.PENDING') },
              { key: 'CONFIRMED', label: t('status.CONFIRMED') },
            ]}
          />
          <dl className="grid gap-4 sm:grid-cols-2">
            <div><dt className="text-xs uppercase text-slate-500">{t('details.status')}</dt><dd className="mt-1"><StatusBadge status={row.status} /></dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.reason')}</dt><dd className="mt-1">{row.reason}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.processedBy')}</dt><dd className="mt-1">{row.processed_by?.full_name ?? '—'}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">{t('details.processedAt')}</dt><dd className="mt-1">{formatDateTime(row.processed_at)}</dd></div>
          </dl>
          {row.rejection_reason && <div className="mt-4 rounded-md bg-red-50 p-3 text-sm text-red-700">{t('details.rejectionReason')}: {row.rejection_reason}</div>}
        </Card>
        <Card>
          <CardHeader title={t('details.items')} description={t('common.lineCount', { count: row.items.length })} />
          <ul className="divide-y divide-slate-100">
            {row.items.map((item) => (
              <li key={item.id} className="px-5 py-3 text-sm">
                <p className="font-medium">{item.product.sku} — {item.product.name}</p>
                <p className="text-slate-500">{t('details.requested')} {formatQty(item.requested_qty)} · {t('details.accepted')} {formatQty(item.accepted_qty)}</p>
              </li>
            ))}
          </ul>
        </Card>
      </div>

      <ConfirmDialog
        open={cancelOpen}
        title={t('dialogs.cancelReturn')}
        onCancel={() => setCancelOpen(false)}
        onConfirm={cancel}
        loading={saving}
        danger
        confirmLabel={t('actions.cancel')}
      >
        {t('dialogs.cancelReturnBody')}
      </ConfirmDialog>

      {rejectOpen && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center">
          <Card className="max-h-[90dvh] w-full max-w-md overflow-y-auto p-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))] sm:pb-5">
            <h3 className="text-lg font-semibold">{t('dialogs.rejectReturn')}</h3>
            <Textarea className="mt-3" value={reason} onChange={(e) => setReason(e.target.value)} placeholder={t('dialogs.reasonPlaceholder')} />
            <div className="mt-4 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end [&>*]:w-full sm:[&>*]:w-auto">
              <Button variant="outline" onClick={() => setRejectOpen(false)}>{t('common.cancel')}</Button>
              <Button variant="danger" loading={saving} onClick={reject}>{t('actions.reject')}</Button>
            </div>
          </Card>
        </div>
      )}

      {confirmOpen && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center">
          <Card className="max-h-[90dvh] w-full max-w-2xl overflow-y-auto p-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))] sm:pb-5">
            <h3 className="text-lg font-semibold">{t('dialogs.confirmReturn')}</h3>
            <p className="mt-1 text-sm text-slate-500">{t('dialogs.confirmReturnBody')}</p>
            <div className="mt-4 space-y-3">
              {row.items.map((item) => (
                <div key={item.id} className="grid items-center gap-3 rounded-lg border border-slate-200 p-3 sm:grid-cols-[1fr_120px]">
                  <div className="text-sm">
                    <p className="font-medium">{item.product.sku} — {item.product.name}</p>
                    <p className="text-slate-500">{t('details.requested')} {formatQty(item.requested_qty)}</p>
                  </div>
                  <Input
                    type="number"
                    min={0}
                    max={item.requested_qty}
                    value={accepted[item.id] ?? 0}
                    onChange={(e) => setAccepted((prev) => ({ ...prev, [item.id]: Math.max(0, Math.min(item.requested_qty, Number(e.target.value) || 0)) }))}
                  />
                </div>
              ))}
            </div>
            <div className="mt-4 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end [&>*]:w-full sm:[&>*]:w-auto">
              <Button variant="outline" onClick={() => setConfirmOpen(false)}>{t('common.cancel')}</Button>
              <Button loading={saving} onClick={confirm}>{t('dialogs.confirmReturn')}</Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  )
}
