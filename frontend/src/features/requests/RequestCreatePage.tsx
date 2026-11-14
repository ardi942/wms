import { useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { requestApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { Button, Card, Field, Input, Textarea } from '../../components/ui'
import { PageHeader, ProductPicker } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { formatQty } from '../../lib/format'
import type { Product } from '../../types'

const schema = z.object({
  purpose: z.string().min(3, 'validation.purposeMin').max(1000, 'validation.tooLong'),
  items: z.array(z.object({
    product: z.custom<Product | null>((v) => Boolean(v), 'validation.selectProduct'),
    requested_qty: z.coerce.number().int().min(1, 'validation.qtyMin').max(1_000_000, 'validation.qtyMax'),
  })).min(1).max(50),
})
type FormValues = z.infer<typeof schema>

export function RequestCreatePage() {
  const { t } = useT()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [submitting, setSubmitting] = useState(false)
  const { control, register, handleSubmit, watch, setValue, setError, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { purpose: '', items: [{ product: null, requested_qty: 1 }] },
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
    try {
      const res = await requestApi.create({
        purpose: values.purpose.trim(),
        items: values.items.map((i) => ({ product_id: i.product!.id, requested_qty: Number(i.requested_qty) })),
      })
      toast.success(t('toast.requestSubmitted'))
      await queryClient.invalidateQueries({ queryKey: ['requests'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      navigate(`/requests/${res.data.id}`)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t('toast.requestFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader title={t('pages.newRequest')} description={t('pages.newRequestDescription')} />
      <Card className="p-5">
        <form className="space-y-5" onSubmit={handleSubmit(onSubmit)} noValidate>
          <Field
            label={t('fields.purpose')}
            hint={t('fields.purposeHint')}
            example={t('fields.purposeExample')}
            htmlFor="purpose"
            error={errors.purpose?.message}
          >
            <Textarea id="purpose" placeholder={t('fields.purposeExample')} {...register('purpose')} />
          </Field>
          <div>
            <div className="mb-2 flex items-center justify-between">
              <h2 className="text-sm font-medium text-slate-700">{t('details.items')}</h2>
              <Button type="button" variant="outline" size="sm" disabled={fields.length >= 50} onClick={() => append({ product: null, requested_qty: 1 })}>
                <Plus className="h-3.5 w-3.5" /> {t('actions.addLine')}
              </Button>
            </div>
            {errors.items?.root?.message && <p className="mb-2 text-xs font-medium text-red-600">{t(errors.items.root.message)}</p>}
            {errors.items?.message && <p className="mb-2 text-xs font-medium text-red-600">{t(errors.items.message)}</p>}
            <div className="space-y-3">
              {fields.map((field, index) => {
                const selected = items[index]?.product ?? null
                const requested = Number(items[index]?.requested_qty ?? 0)
                const exceeds = selected ? requested > selected.current_stock : false
                return (
                  <div key={field.id} className="space-y-2 rounded-lg border border-slate-200 p-3">
                    <div className="grid gap-3 sm:grid-cols-[1fr_170px_40px]">
                      <Field label={t('fields.product')} hint={t('fields.productHint')} error={errors.items?.[index]?.product?.message}>
                        <ProductPicker value={selected} onChange={(p) => setValue(`items.${index}.product`, p, { shouldValidate: true })} />
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
                    {selected && (
                      <div className="space-y-1 text-xs">
                        <p className={exceeds ? 'text-amber-700' : 'text-slate-500'}>
                          {t('fields.availableStock', { qty: formatQty(selected.current_stock), unit: selected.unit.code })}
                        </p>
                        {exceeds && (
                          <p className="text-amber-700">
                            {t('fields.exceedsAvailable', {
                              requested: formatQty(requested),
                              available: formatQty(selected.current_stock),
                              unit: selected.unit.code,
                            })}
                          </p>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => navigate(-1)}>{t('common.cancel')}</Button>
            <Button type="submit" loading={submitting}>{t('actions.submitRequest')}</Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
