import { useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { returnApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { Button, Card, Field, Input, Textarea } from '../../components/ui'
import { PageHeader, ProductPicker } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { formatQty } from '../../lib/format'
import type { Product } from '../../types'

const schema = z.object({
  reason: z.string().min(5, 'validation.reasonMin').max(1000, 'validation.tooLong'),
  items: z.array(z.object({
    product: z.custom<Product | null>((v) => Boolean(v), 'validation.selectProduct'),
    requested_qty: z.coerce.number().int().min(1, 'validation.qtyMin').max(1_000_000, 'validation.qtyMax'),
  })).min(1).max(50),
})
type FormValues = z.infer<typeof schema>

export function ReturnCreatePage() {
  const { t } = useT()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [submitting, setSubmitting] = useState(false)
  const [apiError, setApiError] = useState('')
  const { control, register, handleSubmit, watch, setValue, setError, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { reason: '', items: [{ product: null, requested_qty: 1 }] },
  })
  const { fields, append, remove } = useFieldArray({ control, name: 'items' })
  const items = watch('items')

  const onSubmit = async (values: FormValues) => {
    const ids = values.items.map((i) => i.product?.id).filter(Boolean) as string[]
    if (new Set(ids).size !== ids.length) {
      setError('items', { message: 'validation.duplicateProducts' })
      return
    }
    setSubmitting(true)
    setApiError('')
    try {
      const res = await returnApi.create({
        reason: values.reason.trim(),
        items: values.items.map((i) => ({ product_id: i.product!.id, requested_qty: Number(i.requested_qty) })),
      })
      toast.success(t('toast.returnSubmitted'))
      await queryClient.invalidateQueries({ queryKey: ['returns'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      navigate(`/returns/${res.data.id}`)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'RETURN_EXCEEDS_ISSUED' && err.details) {
        const productId = String(err.details.product_id ?? '')
        const netIssued = Number(err.details.net_issued ?? 0)
        const line = values.items.find((item) => item.product?.id === productId)
        setApiError(t('details.maximumReturnable', { qty: formatQty(netIssued), unit: line?.product?.unit.code ?? '' }).trim())
      }
      toast.error(err instanceof ApiError ? err.message : t('toast.returnFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader title={t('pages.newReturn')} description={t('pages.newReturnDescription')} />
      <Card className="p-5">
        <form className="space-y-5" onSubmit={handleSubmit(onSubmit)} noValidate>
          <p className="rounded-md bg-slate-50 p-3 text-sm text-slate-600">{t('fields.returnableHint')}</p>
          <Field
            label={t('fields.returnReason')}
            hint={t('fields.returnReasonHint')}
            example={t('fields.returnReasonExample')}
            htmlFor="reason"
            error={errors.reason?.message}
          >
            <Textarea id="reason" placeholder={t('fields.returnReasonExample')} {...register('reason')} />
          </Field>
          <div>
            <div className="mb-2 flex items-center justify-between">
              <h2 className="text-sm font-medium text-slate-700">{t('details.items')}</h2>
              <Button type="button" variant="outline" size="sm" disabled={fields.length >= 50} onClick={() => append({ product: null, requested_qty: 1 })}>
                <Plus className="h-3.5 w-3.5" /> {t('actions.addLine')}
              </Button>
            </div>
            {errors.items?.message && <p className="mb-2 text-xs font-medium text-red-600">{t(errors.items.message)}</p>}
            {apiError && <p className="mb-2 rounded-md bg-amber-50 p-2 text-xs font-medium text-amber-800">{apiError}</p>}
            <div className="space-y-3">
              {fields.map((field, index) => {
                const selected = items[index]?.product ?? null
                return (
                  <div key={field.id} className="grid gap-3 rounded-lg border border-slate-200 p-3 sm:grid-cols-[1fr_170px_40px]">
                    <Field label={t('fields.product')} hint={t('fields.productHint')} error={errors.items?.[index]?.product?.message}>
                      <ProductPicker value={selected} onChange={(p) => { setValue(`items.${index}.product`, p, { shouldValidate: true }); setApiError('') }} />
                    </Field>
                    <Field
                      label={t('fields.quantity')}
                      hint={selected ? `${t('fields.quantityHint')} ${selected.unit.code}` : t('fields.quantityHint')}
                      error={errors.items?.[index]?.requested_qty?.message}
                    >
                      <Input type="number" min={1} {...register(`items.${index}.requested_qty`)} />
                    </Field>
                    <div className="flex items-end">
                      <Button type="button" variant="ghost" aria-label={t('actions.removeLine')} disabled={fields.length === 1} onClick={() => remove(index)}>
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => navigate(-1)}>{t('common.cancel')}</Button>
            <Button type="submit" loading={submitting}>{t('actions.submitReturn')}</Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
