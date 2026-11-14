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
import type { Product } from '../../types'

const schema = z.object({
  purpose: z.string().min(3, 'At least 3 characters').max(1000),
  items: z.array(z.object({
    product: z.custom<Product | null>((v) => Boolean(v), 'Select a product'),
    requested_qty: z.coerce.number().int().min(1).max(1_000_000),
  })).min(1).max(50),
})
type FormValues = z.infer<typeof schema>

export function RequestCreatePage() {
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
      setError('items', { message: 'Duplicate products are not allowed' })
      return
    }
    setSubmitting(true)
    try {
      const res = await requestApi.create({
        purpose: values.purpose.trim(),
        items: values.items.map((i) => ({ product_id: i.product!.id, requested_qty: Number(i.requested_qty) })),
      })
      toast.success('Request submitted')
      await queryClient.invalidateQueries({ queryKey: ['requests'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      navigate(`/requests/${res.data.id}`)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Request failed')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader title="New stock request" description="Add up to 50 product lines. Quantities are validated against availability when fulfilled." />
      <Card className="p-5">
        <form className="space-y-5" onSubmit={handleSubmit(onSubmit)} noValidate>
          <Field label="Purpose" htmlFor="purpose" error={errors.purpose?.message}>
            <Textarea id="purpose" placeholder="What is this request for?" {...register('purpose')} />
          </Field>
          <div>
            <div className="mb-2 flex items-center justify-between">
              <h2 className="text-sm font-medium text-slate-700">Items</h2>
              <Button type="button" variant="outline" size="sm" disabled={fields.length >= 50} onClick={() => append({ product: null, requested_qty: 1 })}>
                <Plus className="h-3.5 w-3.5" /> Add line
              </Button>
            </div>
            {errors.items?.root?.message && <p className="mb-2 text-xs text-red-600">{errors.items.root.message}</p>}
            {errors.items?.message && <p className="mb-2 text-xs text-red-600">{errors.items.message}</p>}
            <div className="space-y-3">
              {fields.map((field, index) => (
                <div key={field.id} className="grid gap-3 rounded-lg border border-slate-200 p-3 sm:grid-cols-[1fr_130px_40px]">
                  <Field label="Product" error={errors.items?.[index]?.product?.message}>
                    <ProductPicker value={items[index]?.product ?? null} onChange={(p) => setValue(`items.${index}.product`, p, { shouldValidate: true })} />
                  </Field>
                  <Field label="Quantity" error={errors.items?.[index]?.requested_qty?.message}>
                    <Input type="number" min={1} {...register(`items.${index}.requested_qty`)} />
                  </Field>
                  <div className="flex items-end">
                    <Button type="button" variant="ghost" aria-label="Remove line" disabled={fields.length === 1} onClick={() => remove(index)}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => navigate(-1)}>Cancel</Button>
            <Button type="submit" loading={submitting}>Submit request</Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
