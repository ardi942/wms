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
import { useT } from '../../i18n/I18nProvider'
import { formatDateTime, formatQty, todayInput } from '../../lib/format'
import type { Movement, Product } from '../../types'

const schema = z.object({
  quantity: z.coerce.number().int().min(1, 'validation.qtyMin').max(1_000_000, 'validation.qtyMax'),
  supplier: z.string().min(1, 'validation.required').max(200, 'validation.tooLong'),
  reference_number: z.string().min(1, 'validation.required').max(100, 'validation.tooLong'),
  notes: z.string().max(1000, 'validation.tooLong').optional(),
})
type FormValues = z.infer<typeof schema>

export function ReceivingPage() {
  const { t } = useT()
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
      setProductError('validation.selectProduct')
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
      toast.success(t('toast.stockReceived'))
      await queryClient.invalidateQueries({ queryKey: ['movements'] })
      await queryClient.invalidateQueries({ queryKey: ['inventory'] })
      await queryClient.invalidateQueries({ queryKey: ['products'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      setConfirmOpen(false)
      setPending(null)
      setProduct(null)
      reset()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t('toast.receivingFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  const columns: Column<Movement>[] = [
    { key: 'time', header: t('details.time'), render: (m) => formatDateTime(m.created_at) },
    { key: 'product', header: t('details.product'), render: (m) => `${m.product.sku} — ${m.product.name}` },
    { key: 'qty', header: t('details.quantity'), render: (m) => `+${formatQty(m.quantity_delta)}` },
    { key: 'supplier', header: t('details.supplier'), render: (m) => m.supplier ?? '—' },
    { key: 'ref', header: t('details.reference'), render: (m) => m.reference_number },
  ]

  return (
    <div>
      <PageHeader title={t('pages.receiving')} description={t('pages.receivingDescription')} />
      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="p-5">
          <h2 className="mb-4 text-base font-semibold">{t('actions.receiveStock')}</h2>
          <form className="space-y-4" onSubmit={handleSubmit(prepare)} noValidate>
            <Field label={t('fields.product')} hint={t('fields.productHint')} error={productError}>
              <ProductPicker value={product} onChange={(p) => { setProduct(p); setProductError('') }} />
            </Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field
                label={t('fields.quantity')}
                hint={`${t('fields.quantityHint')} ${product?.unit.code ?? ''}`.trim()}
                htmlFor="quantity"
                error={errors.quantity?.message}
              >
                <Input id="quantity" type="number" min={1} {...register('quantity')} />
              </Field>
              <Field
                label={t('fields.supplier')}
                hint={t('fields.supplierHint')}
                example={t('fields.supplierExample')}
                htmlFor="supplier"
                error={errors.supplier?.message}
              >
                <Input id="supplier" placeholder={t('fields.supplierExample')} {...register('supplier')} />
              </Field>
            </div>
            <Field
              label={t('fields.reference')}
              hint={t('fields.referenceHint')}
              example={t('fields.referenceExample')}
              htmlFor="reference_number"
              error={errors.reference_number?.message}
            >
              <Input id="reference_number" placeholder={t('fields.referenceExample')} {...register('reference_number')} />
            </Field>
            <Field
              label={t('fields.notes')}
              hint={t('fields.notesHint')}
              example={t('fields.notesExample')}
              htmlFor="notes"
              error={errors.notes?.message}
            >
              <Textarea id="notes" {...register('notes')} />
            </Field>
            <Button type="submit">{t('actions.reviewReceipt')}</Button>
          </form>
        </Card>
        <Card>
          <CardHeader title={t('dashboard.todayActivity')} />
          <DataTable columns={columns} rows={receipts.data?.items ?? []} loading={receipts.isLoading} error={receipts.error} emptyTitle={t('empty.noReceiptsToday')} />
          {receipts.data && <Pagination meta={receipts.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
        </Card>
      </div>
      <ConfirmDialog
        open={confirmOpen}
        title={t('dialogs.confirmReceipt')}
        onCancel={() => setConfirmOpen(false)}
        onConfirm={submit}
        loading={submitting}
        confirmLabel={t('actions.receive')}
      >
        {product && pending && (
          <p>
            {t('dialogs.receiveBody', {
              qty: formatQty(Number(pending.quantity)),
              unit: product.unit.code,
              product: `${product.sku} — ${product.name}`,
              supplier: pending.supplier,
            })}
          </p>
        )}
      </ConfirmDialog>
    </div>
  )
}
