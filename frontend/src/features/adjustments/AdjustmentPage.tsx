import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { inventoryApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { HelpCircle } from 'lucide-react'
import { Button, Card, CardHeader, Field, Input, Textarea, Tooltip } from '../../components/ui'
import { ConfirmDialog, DataTable, PageHeader, ProductPicker, type Column } from '../../components/common'
import { formatDateTime, formatQty } from '../../lib/format'
import type { Movement, Product } from '../../types'

const schema = z.object({
  quantity_delta: z.coerce.number().int().refine((v) => v !== 0, 'Change cannot be zero'),
  reason: z.string().min(5).max(1000),
})
type FormValues = z.infer<typeof schema>

export function AdjustmentPage() {
  const queryClient = useQueryClient()
  const [product, setProduct] = useState<Product | null>(null)
  const [productError, setProductError] = useState('')
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [pending, setPending] = useState<FormValues | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const { register, handleSubmit, watch, reset, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { quantity_delta: 1, reason: '' },
  })
  const delta = Number(watch('quantity_delta') || 0)
  const projected = (product?.current_stock ?? 0) + delta
  const adjustments = useQuery({
    queryKey: ['movements', { reference_type: 'ADJUSTMENT' }],
    queryFn: () => inventoryApi.movements({ reference_type: 'ADJUSTMENT', page_size: 20 }),
  })

  const prepare = (values: FormValues) => {
    if (!product) { setProductError('Select a product'); return }
    setProductError('')
    setPending(values)
    setConfirmOpen(true)
  }

  const submit = async () => {
    if (!product || !pending) return
    setSubmitting(true)
    try {
      await inventoryApi.adjust({ product_id: product.id, quantity_delta: Number(pending.quantity_delta), reason: pending.reason.trim() })
      toast.success('Stock adjusted')
      await queryClient.invalidateQueries({ queryKey: ['movements'] })
      await queryClient.invalidateQueries({ queryKey: ['inventory'] })
      await queryClient.invalidateQueries({ queryKey: ['products'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      setConfirmOpen(false)
      setPending(null)
      setProduct(null)
      reset()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Adjustment failed')
    } finally {
      setSubmitting(false)
    }
  }

  const columns: Column<Movement>[] = [
    { key: 'time', header: 'Time', render: (m) => formatDateTime(m.created_at) },
    { key: 'product', header: 'Product', render: (m) => `${m.product.sku} — ${m.product.name}` },
    { key: 'delta', header: 'Change', render: (m) => `${m.quantity_delta > 0 ? '+' : ''}${formatQty(m.quantity_delta)}` },
    { key: 'reason', header: 'Reason', render: (m) => m.reason ?? '—' },
    { key: 'by', header: 'By', render: (m) => m.actor.full_name },
  ]

  return (
    <div>
      <PageHeader title="Adjustments" description="Correct stock levels with an auditable reason." />
      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="p-5">
          <h2 className="mb-4 text-base font-semibold">New adjustment</h2>
          <form className="space-y-4" onSubmit={handleSubmit(prepare)} noValidate>
            <Field label="Product" error={productError}>
              <ProductPicker value={product} onChange={(p) => { setProduct(p); setProductError('') }} />
            </Field>
            <Field label="Quantity change" htmlFor="quantity_delta" error={errors.quantity_delta?.message}>
              <div className="flex items-center gap-2">
                <Input id="quantity_delta" type="number" {...register('quantity_delta')} />
                <Tooltip content="Positive numbers add stock, negative numbers remove it.">
                  <HelpCircle className="h-4 w-4 text-slate-400" />
                </Tooltip>
              </div>
            </Field>
            {product && (
              <p className={`text-sm ${projected < 0 ? 'text-red-600' : 'text-slate-600'}`}>
                Resulting balance: <strong>{formatQty(projected)} {product.unit.code}</strong>
                {projected < 0 && ' — this adjustment would make stock negative.'}
              </p>
            )}
            <Field label="Reason" htmlFor="reason" error={errors.reason?.message}>
              <Textarea id="reason" placeholder="Stocktake correction, damage write-off…" {...register('reason')} />
            </Field>
            <Button type="submit">Review adjustment</Button>
          </form>
        </Card>
        <Card>
          <CardHeader title="Recent adjustments" />
          <DataTable columns={columns} rows={adjustments.data?.items ?? []} loading={adjustments.isLoading} error={adjustments.error} emptyTitle="No adjustments yet" />
        </Card>
      </div>
      <ConfirmDialog open={confirmOpen} title="Confirm adjustment" onCancel={() => setConfirmOpen(false)} onConfirm={submit} loading={submitting} danger={delta < 0} confirmLabel="Adjust">
        {product && pending && (
          <p>Adjust <strong>{product.sku}</strong> by <strong>{Number(pending.quantity_delta) > 0 ? '+' : ''}{formatQty(Number(pending.quantity_delta))}</strong> for: {pending.reason}</p>
        )}
      </ConfirmDialog>
    </div>
  )
}
