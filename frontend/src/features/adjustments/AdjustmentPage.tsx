import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { inventoryApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { Button, Card, CardHeader, Field, Input, Textarea } from '../../components/ui'
import { ConfirmDialog, DataTable, PageHeader, ProductPicker, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { formatDateTime, formatQty } from '../../lib/format'
import type { Movement, Product } from '../../types'

const schema = z.object({
  quantity_delta: z.coerce.number().int().refine((v) => v !== 0, 'validation.changeNonZero'),
  reason: z.string().min(5, 'validation.reasonMin').max(1000, 'validation.tooLong'),
})
type FormValues = z.infer<typeof schema>

export function AdjustmentPage() {
  const { t } = useT()
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
    if (!product) { setProductError('validation.selectProduct'); return }
    setProductError('')
    setPending(values)
    setConfirmOpen(true)
  }

  const submit = async () => {
    if (!product || !pending) return
    setSubmitting(true)
    try {
      await inventoryApi.adjust({ product_id: product.id, quantity_delta: Number(pending.quantity_delta), reason: pending.reason.trim() })
      toast.success(t('toast.stockAdjusted'))
      await queryClient.invalidateQueries({ queryKey: ['movements'] })
      await queryClient.invalidateQueries({ queryKey: ['inventory'] })
      await queryClient.invalidateQueries({ queryKey: ['products'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      setConfirmOpen(false)
      setPending(null)
      setProduct(null)
      reset()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t('toast.adjustmentFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  const columns: Column<Movement>[] = [
    { key: 'time', header: t('details.time'), render: (m) => formatDateTime(m.created_at) },
    { key: 'product', header: t('details.product'), render: (m) => `${m.product.sku} — ${m.product.name}` },
    { key: 'delta', header: t('details.change'), render: (m) => `${m.quantity_delta > 0 ? '+' : ''}${formatQty(m.quantity_delta)}` },
    { key: 'reason', header: t('details.reason'), render: (m) => m.reason ?? '—' },
    { key: 'by', header: t('details.by'), render: (m) => m.actor.full_name },
  ]

  return (
    <div>
      <PageHeader title={t('pages.adjustments')} description={t('pages.adjustmentsDescription')} />
      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="p-5">
          <h2 className="mb-4 text-base font-semibold">{t('actions.reviewAdjustment')}</h2>
          <form className="space-y-4" onSubmit={handleSubmit(prepare)} noValidate>
            <Field label={t('fields.product')} hint={t('fields.productHint')} error={productError}>
              <ProductPicker value={product} onChange={(p) => { setProduct(p); setProductError('') }} />
            </Field>
            <Field
              label={t('fields.stockChange')}
              hint={t('fields.stockChangeHint')}
              htmlFor="quantity_delta"
              error={errors.quantity_delta?.message}
            >
              <div className="flex items-center gap-2">
                <Input id="quantity_delta" type="number" {...register('quantity_delta')} />
                <span className="shrink-0 text-sm text-slate-500">{product?.unit.code ?? t('common.unit')}</span>
              </div>
            </Field>
            {product && (
              <p className={`text-sm ${projected < 0 ? 'text-red-600' : 'text-slate-600'}`}>
                {t('details.balanceAfter')}: <strong>{formatQty(projected)} {product.unit.code}</strong>
                {projected < 0 && ` — ${t('dialogs.negativeStock')}`}
              </p>
            )}
            <Field
              label={t('fields.reason')}
              hint={t('fields.reasonHint')}
              example={t('fields.reasonExample')}
              htmlFor="reason"
              error={errors.reason?.message}
            >
              <Textarea id="reason" placeholder={t('fields.reasonExample')} {...register('reason')} />
            </Field>
            <Button type="submit">{t('actions.reviewAdjustment')}</Button>
          </form>
        </Card>
        <Card>
          <CardHeader title={t('pages.adjustments')} />
          <DataTable columns={columns} rows={adjustments.data?.items ?? []} loading={adjustments.isLoading} error={adjustments.error} emptyTitle={t('empty.noAdjustments')} />
        </Card>
      </div>
      <ConfirmDialog
        open={confirmOpen}
        title={t('dialogs.confirmAdjustment')}
        onCancel={() => setConfirmOpen(false)}
        onConfirm={submit}
        loading={submitting}
        danger={delta < 0}
        confirmLabel={t('actions.adjust')}
      >
        {product && pending && (
          <p>
            {t('dialogs.adjustBody', {
              product: `${product.sku} — ${product.name}`,
              delta: `${Number(pending.quantity_delta) > 0 ? '+' : ''}${formatQty(Number(pending.quantity_delta))} ${product.unit.code}`,
              reason: pending.reason,
            })}
          </p>
        )}
      </ConfirmDialog>
    </div>
  )
}
