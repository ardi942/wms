import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { productApi, referenceApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, Field, Input, Select, Textarea } from '../../components/ui'
import { ErrorState, Loading, PageHeader } from '../../components/common'

const schema = z.object({
  sku: z.string().regex(/^[A-Z0-9][A-Z0-9_-]{1,31}$/, 'Use 2–32 uppercase letters, digits, _ or -'),
  name: z.string().min(2, 'At least 2 characters').max(200),
  description: z.string().max(2000).optional(),
  category_id: z.string().uuid('Choose a category'),
  unit_id: z.string().uuid('Choose a unit'),
  min_stock: z.coerce.number().int().min(0),
  unit_cost: z.string().optional(),
})
type FormValues = z.infer<typeof schema>

export function ProductFormPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const isSupervisor = user?.role === 'SUPERVISOR'
  const editing = Boolean(id)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [submitting, setSubmitting] = useState(false)
  const product = useQuery({ queryKey: ['product', id], queryFn: () => productApi.get(id!), enabled: editing })
  const references = useQuery({
    queryKey: ['references'],
    queryFn: async () => {
      const [categories, units] = await Promise.all([referenceApi.categories(), referenceApi.units()])
      return { categories: categories.data, units: units.data }
    },
  })
  const { register, handleSubmit, reset, setError, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { sku: '', name: '', description: '', category_id: '', unit_id: '', min_stock: 0, unit_cost: '' },
  })

  useEffect(() => {
    if (product.data) {
      const p = product.data.data
      reset({
        sku: p.sku, name: p.name, description: p.description ?? '',
        category_id: p.category.id, unit_id: p.unit.id, min_stock: p.min_stock,
        unit_cost: p.unit_cost === null ? '' : String(p.unit_cost),
      })
    }
  }, [product.data, reset])

  const onSubmit = async (values: FormValues) => {
    setSubmitting(true)
    const body: Record<string, unknown> = {
      sku: values.sku.trim().toUpperCase(),
      name: values.name.trim(),
      description: values.description?.trim() || null,
      category_id: values.category_id,
      unit_id: values.unit_id,
      min_stock: Number(values.min_stock),
    }
    if (isSupervisor) body.unit_cost = values.unit_cost ? Number(values.unit_cost) : null
    try {
      if (editing) await productApi.update(id!, body)
      else await productApi.create(body)
      toast.success(editing ? 'Product updated' : 'Product created')
      await queryClient.invalidateQueries({ queryKey: ['products'] })
      await queryClient.invalidateQueries({ queryKey: ['product', id] })
      await queryClient.invalidateQueries({ queryKey: ['inventory'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      navigate(editing ? `/products/${id}` : '/products')
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'Save failed'
      if (err instanceof ApiError && err.details) {
        for (const [field, value] of Object.entries(err.details)) {
          if (field in schema.shape) setError(field as keyof FormValues, { message: String(value) })
        }
      }
      toast.error(message)
    } finally {
      setSubmitting(false)
    }
  }

  if (editing && product.isLoading) return <Loading />
  if (editing && product.error) return <ErrorState error={product.error} onRetry={() => product.refetch()} />

  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader title={editing ? 'Edit product' : 'New product'} description="SKU and stock thresholds drive inventory alerts." />
      <Card className="p-6">
        <form className="space-y-4" onSubmit={handleSubmit(onSubmit)} noValidate>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="SKU" htmlFor="sku" error={errors.sku?.message}>
              <Input id="sku" className="uppercase" {...register('sku')} />
            </Field>
            <Field label="Name" htmlFor="name" error={errors.name?.message}>
              <Input id="name" {...register('name')} />
            </Field>
            <Field label="Category" htmlFor="category_id" error={errors.category_id?.message}>
              <Select id="category_id" {...register('category_id')}>
                <option value="">Select category</option>
                {references.data?.categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </Select>
            </Field>
            <Field label="Unit" htmlFor="unit_id" error={errors.unit_id?.message}>
              <Select id="unit_id" {...register('unit_id')}>
                <option value="">Select unit</option>
                {references.data?.units.map((u) => <option key={u.id} value={u.id}>{u.code} — {u.name}</option>)}
              </Select>
            </Field>
            <Field label="Minimum stock" htmlFor="min_stock" error={errors.min_stock?.message}>
              <Input id="min_stock" type="number" min={0} {...register('min_stock')} />
            </Field>
            {isSupervisor && (
              <Field label="Unit cost (IDR)" htmlFor="unit_cost" error={errors.unit_cost?.message}>
                <Input id="unit_cost" type="number" step="1" min={0} {...register('unit_cost')} />
              </Field>
            )}
          </div>
          <Field label="Description (optional)" htmlFor="description" error={errors.description?.message}>
            <Textarea id="description" {...register('description')} />
          </Field>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => navigate(-1)}>Cancel</Button>
            <Button type="submit" loading={submitting}>{editing ? 'Save changes' : 'Create product'}</Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
