import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { returnApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, CardHeader, Input, Textarea } from '../../components/ui'
import { ConfirmDialog, ErrorState, Loading, PageHeader, StatusBadge } from '../../components/common'
import { formatDateTime, formatQty } from '../../lib/format'

export function ReturnDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const ret = useQuery({ queryKey: ['return', id], queryFn: () => returnApi.get(id!) })
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [rejectOpen, setRejectOpen] = useState(false)
  const [cancelOpen, setCancelOpen] = useState(false)
  const [accepted, setAccepted] = useState<Record<string, number>>({})
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)

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

  const isAdmin = user?.role === 'WAREHOUSE_ADMIN'
  const isDept = user?.role === 'DEPARTMENT_USER'

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
    if (!lines.some((l) => l.accepted_qty > 0)) { toast.error('At least one accepted quantity must be greater than zero'); return }
    setSaving(true)
    try {
      await returnApi.confirm(row.id, { items: lines })
      toast.success('Return confirmed')
      setConfirmOpen(false)
      await invalidate()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Confirmation failed')
    } finally {
      setSaving(false)
    }
  }

  const reject = async () => {
    if (reason.trim().length < 5) { toast.error('Reason must be at least 5 characters'); return }
    setSaving(true)
    try {
      await returnApi.reject(row.id, { reason: reason.trim() })
      toast.success('Return rejected')
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
      await returnApi.cancel(row.id)
      toast.success('Return cancelled')
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
        title={row.return_number}
        description={`${row.department.name} · submitted by ${row.requested_by.full_name} on ${formatDateTime(row.created_at)}`}
        actions={
          <>
            <Link to="/returns"><Button variant="outline">Back</Button></Link>
            {isAdmin && row.status === 'PENDING' && <Button variant="danger" onClick={() => setRejectOpen(true)}>Reject</Button>}
            {isAdmin && row.status === 'PENDING' && <Button onClick={() => setConfirmOpen(true)}>Confirm return</Button>}
            {isDept && row.status === 'PENDING' && <Button variant="danger" onClick={() => setCancelOpen(true)}>Cancel return</Button>}
          </>
        }
      />
      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="p-5 lg:col-span-2">
          <dl className="grid gap-4 sm:grid-cols-2">
            <div><dt className="text-xs uppercase text-slate-500">Status</dt><dd className="mt-1"><StatusBadge status={row.status} /></dd></div>
            <div><dt className="text-xs uppercase text-slate-500">Reason</dt><dd className="mt-1">{row.reason}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">Processed by</dt><dd className="mt-1">{row.processed_by?.full_name ?? '—'}</dd></div>
            <div><dt className="text-xs uppercase text-slate-500">Processed at</dt><dd className="mt-1">{formatDateTime(row.processed_at)}</dd></div>
          </dl>
          {row.rejection_reason && <div className="mt-4 rounded-md bg-red-50 p-3 text-sm text-red-700">Rejection reason: {row.rejection_reason}</div>}
        </Card>
        <Card>
          <CardHeader title="Items" description={`${row.items.length} line(s)`} />
          <ul className="divide-y divide-slate-100">
            {row.items.map((item) => (
              <li key={item.id} className="px-5 py-3 text-sm">
                <p className="font-medium">{item.product.sku} — {item.product.name}</p>
                <p className="text-slate-500">Requested {formatQty(item.requested_qty)} · Accepted {formatQty(item.accepted_qty)}</p>
              </li>
            ))}
          </ul>
        </Card>
      </div>

      <ConfirmDialog open={cancelOpen} title="Cancel return?" onCancel={() => setCancelOpen(false)} onConfirm={cancel} loading={saving} danger confirmLabel="Cancel return">
        This return will be cancelled and cannot be confirmed.
      </ConfirmDialog>

      {rejectOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4">
          <Card className="w-full max-w-md p-5">
            <h3 className="text-lg font-semibold">Reject return</h3>
            <Textarea className="mt-3" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Reason for rejection (min. 5 characters)" />
            <div className="mt-4 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setRejectOpen(false)}>Cancel</Button>
              <Button variant="danger" loading={saving} onClick={reject}>Reject return</Button>
            </div>
          </Card>
        </div>
      )}

      {confirmOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4">
          <Card className="max-h-[90vh] w-full max-w-2xl overflow-y-auto p-5">
            <h3 className="text-lg font-semibold">Confirm return</h3>
            <p className="mt-1 text-sm text-slate-500">Enter the accepted quantity for every line. Rejected quantities stay in the department's issued balance.</p>
            <div className="mt-4 space-y-3">
              {row.items.map((item) => (
                <div key={item.id} className="grid items-center gap-3 rounded-lg border border-slate-200 p-3 sm:grid-cols-[1fr_120px]">
                  <div className="text-sm">
                    <p className="font-medium">{item.product.sku} — {item.product.name}</p>
                    <p className="text-slate-500">Requested {formatQty(item.requested_qty)}</p>
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
            <div className="mt-4 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setConfirmOpen(false)}>Cancel</Button>
              <Button loading={saving} onClick={confirm}>Confirm return</Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  )
}
