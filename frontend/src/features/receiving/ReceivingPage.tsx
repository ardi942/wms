import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { inventoryApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { Button, Card, CardHeader, Field, Input, Textarea } from '../../components/ui'
import { ConfirmDialog, DataTable, PageHeader, Pagination, ProductPicker, type Column } from '../../components/common'
import { formatDateTime, formatQty, todayInput } from '../../lib/format'
import type { Movement, Product } from '../../types'

const schema = z.object({
  quantity: z.coerce.number().int().min(1).max(1_000_000),
  supplier: z.string().min(1).max(200),
  reference_number: z.string().min(1).max(100),
  notes: z.string().max(1000).optional(),
})
type FormValues = z.infer<typeof schema>

export function ReceivingPage() {
  const queryClient = useQueryClient()
  const [product, setProduct] = useState<Product | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [pending, setPending] = useState<FormValues | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [productError, setProductError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const { register, handleSubmit, reset, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { quantity: 1, supplier: '', reference_number: '', notes: '' },
  })
  const receipts = useQuery({
    queryKey: ['movements', { today: todayInput(), reference_type: 'RECEIPT', page, page_size: pageSize }],
    queryFn: () => inventoryApi.movements({ date_from: todayInput(), date_to: todayInput(), reference_type: 'RECEIPT', page, page_size: pageSize }),
  })

  const prepare = (values: FormValues) => {
    if (!product) {
      setProductError('Select a product')
      return
    }
    setProductError('')
    setPending(values)
    setConfirmOpen(true)
  }

  const submit = async () => {
    if (!product || !pending) return
    setSubmitting(true)
    try {
      await inventoryApi.receive({
        product_id: product.id,
        quantity: Number(pending.quantity),
        supplier: pending.supplier.trim(),
        reference_number: pending.reference_number.trim(),
        notes: pending.notes?.trim() || null,
      })
      toast.success('Stock received')
      await queryClient.invalidateQueries({ queryKey: ['movements'] })
      await queryClient.invalidateQueries({ queryKey: ['inventory'] })
      await queryClient.invalidateQueries({ queryKey: ['products'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      setConfirmOpen(false)
      setPending(null)
      setProduct(null)
      reset()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Receiving failed')
    } finally {
      setSubmitting(false)
    }
  }

  const columns: Column<Movement>[] = [
    { key: 'time', header: 'Time', render: (m) => formatDateTime(m.created_at) },
    { key: 'product', header: 'Product', render: (m) => `${m.product.sku} — ${m.product.name}` },
    { key: 'qty', header: 'Quantity', render: (m) => `+${formatQty(m.quantity_delta)}` },
    { key: 'supplier', header: 'Supplier', render: (m) => m.supplier ?? '—' },
    { key: 'ref', header: 'Reference', render: (m) => m.reference_number },
  ]

  return (
    <div>
      <PageHeader title="Receiving" description="Record incoming stock and review today's receipts." />
      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="p-5">
          <h2 className="mb-4 text-base font-semibold">Receive stock</h2>
          <form className="space-y-4" onSubmit={handleSubmit(prepare)} noValidate>
            <Field label="Product" error={productError}>
              <ProductPicker value={product} onChange={(p) => { setProduct(p); setProductError('') }} />
            </Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Quantity" htmlFor="quantity" error={errors.quantity?.message}>
                <Input id="quantity" type="number" min={1} {...register('quantity')} />
              </Field>
              <Field label="Supplier" htmlFor="supplier" error={errors.supplier?.message}>
                <Input id="supplier" placeholder="Acme Supplies" {...register('supplier')} />
              </Field>
            </div>
            <Field label="Reference number" htmlFor="reference_number" error={errors.reference_number?.message}>
              <Input id="reference_number" placeholder="PO-10293" {...register('reference_number')} />
            </Field>
            <Field label="Notes (optional)" htmlFor="notes" error={errors.notes?.message}>
              <Textarea id="notes" {...register('notes')} />
            </Field>
            <Button type="submit">Review receipt</Button>
          </form>
        </Card>
        <Card>
          <CardHeader title="Today's receipts" />
          <DataTable columns={columns} rows={receipts.data?.items ?? []} loading={receipts.isLoading} error={receipts.error} emptyTitle="No receipts yet today" />
          {receipts.data && <Pagination meta={receipts.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
        </Card>
      </div>
      <ConfirmDialog open={confirmOpen} title="Confirm receipt" onCancel={() => setConfirmOpen(false)} onConfirm={submit} loading={submitting} confirmLabel="Receive">
        {product && pending && (
          <p>Receive <strong>{formatQty(Number(pending.quantity))} {product.unit.code}</strong> of <strong>{product.sku} — {product.name}</strong> from {pending.supplier}.</p>
        )}
      </ConfirmDialog>
    </div>
  )
}
