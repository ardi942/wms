import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { inventoryApi, requestApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, CardHeader, Input, Textarea } from '../../components/ui'
import { ConfirmDialog, ErrorState, Loading, PageHeader, StatusBadge } from '../../components/common'
import { formatDateTime, formatQty } from '../../lib/format'

export function RequestDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const request = useQuery({ queryKey: ['request', id], queryFn: () => requestApi.get(id!) })
  const balances = useQuery({
    queryKey: ['inventory', { fulfill: id }],
    queryFn: () => inventoryApi.balances({ page_size: 50 }),
    enabled: user?.role === 'WAREHOUSE_ADMIN',
  })
  const [fulfillOpen, setFulfillOpen] = useState(false)
  const [rejectOpen, setRejectOpen] = useState(false)
  const [cancelOpen, setCancelOpen] = useState(false)
  const [issued, setIssued] = useState<Record<string, number>>({})
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)

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

  const isAdmin = user?.role === 'WAREHOUSE_ADMIN'
  const isDept = user?.role === 'DEPARTMENT_USER'
  const canAct = req.status === 'PENDING' || req.status === 'PARTIALLY_FULFILLED'
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
    if (!lines.length) { toast.error('Enter at least one quantity'); return }
    setSaving(true)
    try {
      await requestApi.fulfill(req.id, { items: lines })
      toast.success('Request fulfilled')
      setFulfillOpen(false)
      await invalidate()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Fulfilment failed')
    } finally {
      setSaving(false)
    }
  }

  const reject = async () => {
    if (reason.trim().length < 5) { toast.error('Reason must be at least 5 characters'); return }
    setSaving(true)
    try {
      await requestApi.reject(req.id, { reason: reason.trim() })
      toast.success('Request rejected')
      setRejectOpen(false)
      setReason('')
      await invalidate()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Rejection failed')
    } finally {
      setSaving(false)
    }
  }

  const cancel = async () => {
    setSaving(true)
    try {
      await requestApi.cancel(req.id)
      toast.success('Request cancelled')
      setCancelOpen(false)
      await invalidate()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Cancellation failed')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div>
      <PageHeader
        title={req.request_number}
        description={`${req.department.name} · submitted by ${req.requested_by.full_name} on ${formatDateTime(req.created_at)}`}
        actions={
          <>
            <Link to="/requests"><Button variant="outline">Back</Button></Link>
            {isAdmin && canAct && <Button variant="danger" onClick={() => setRejectOpen(true)}>Reject</Button>}
            {isAdmin && canAct && <Button onClick={() => setFulfillOpen(true)}>Fulfil</Button>}
            {isDept && req.status === 'PENDING' && <Button variant="danger" onClick={() => setCancelOpen(true)}>Cancel request</Button>}
          </>
        }
      />
      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="p-5 lg:col-span-2">
          <dl className="grid gap-4 sm:grid-cols-2">
            <div><dt className="text-xs uppercase text-slate-500">Status</dt><dd className="mt-1"><StatusBadge status={req.status} /></dd></div>
            <div><dt className="text-xs uppercase text-slate-500">Purpose</dt><dd className="mt-1">{req.purpose}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">Processed by</dt><dd className="mt-1">{req.processed_by?.full_name ?? '—'}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">Processed at</dt><dd className="mt-1">{formatDateTime(req.processed_at)}</dd></div>
          </dl>
          {req.rejection_reason && <div className="mt-4 rounded-md bg-red-50 p-3 text-sm text-red-700">Rejection reason: {req.rejection_reason}</div>}
        </Card>
        <Card>
          <CardHeader title="Items" description={`${req.items.length} line(s)`} />
          <ul className="divide-y divide-slate-100">
            {req.items.map((item) => (
              <li key={item.id} className="px-5 py-3 text-sm">
                <p className="font-medium">{item.product.sku} — {item.product.name}</p>
                <p className="text-slate-500">Requested {formatQty(item.requested_qty)} {item.product.unit ?? ''} · Issued {formatQty(item.issued_qty)} · Remaining {formatQty(item.remaining_qty)}</p>
              </li>
            ))}
          </ul>
        </Card>
      </div>

      <ConfirmDialog open={cancelOpen} title="Cancel request?" onCancel={() => setCancelOpen(false)} onConfirm={cancel} loading={saving} danger confirmLabel="Cancel request">
        This request will be cancelled and can no longer be fulfilled.
      </ConfirmDialog>

      {rejectOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4">
          <Card className="w-full max-w-md p-5">
            <h3 className="text-lg font-semibold">Reject request</h3>
            <p className="mt-1 text-sm text-slate-500">Provide a reason. The requesting department will be notified.</p>
            <Textarea className="mt-3" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Reason for rejection (min. 5 characters)" />
            <div className="mt-4 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setRejectOpen(false)}>Cancel</Button>
              <Button variant="danger" loading={saving} onClick={reject}>Reject request</Button>
            </div>
          </Card>
        </div>
      )}

      {fulfillOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4">
          <Card className="max-h-[90vh] w-full max-w-2xl overflow-y-auto p-5">
            <h3 className="text-lg font-semibold">Fulfil request</h3>
            <p className="mt-1 text-sm text-slate-500">Enter the quantity issued per line. Omit lines with zero; at least one line must be issued.</p>
            <div className="mt-4 space-y-3">
              {req.items.map((item) => (
                <div key={item.id} className="grid items-center gap-3 rounded-lg border border-slate-200 p-3 sm:grid-cols-[1fr_120px]">
                  <div className="text-sm">
                    <p className="font-medium">{item.product.sku} — {item.product.name}</p>
                    <p className="text-slate-500">Remaining {formatQty(item.remaining_qty)} · Available {formatQty(stockByProduct.get(item.product.id) ?? 0)}</p>
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
            <div className="mt-4 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setFulfillOpen(false)}>Cancel</Button>
              <Button loading={saving} onClick={fulfill}>Fulfil request</Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  )
}
