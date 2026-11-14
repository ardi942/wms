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
import { useT } from '../../i18n/I18nProvider'
import { formatCurrency } from '../../lib/format'

const schema = z.object({
  sku: z.string().regex(/^[A-Z0-9][A-Z0-9_-]{1,31}$/, 'validation.skuFormat'),
  name: z.string().min(2, 'validation.nameMin').max(200, 'validation.tooLong'),
  description: z.string().max(2000, 'validation.tooLong').optional(),
  category_id: z.string().uuid('validation.selectCategory'),
  unit_id: z.string().uuid('validation.selectUnit'),
  min_stock: z.coerce.number().int().min(0, 'validation.minZero'),
  unit_cost: z.string().optional(),
})
type FormValues = z.infer<typeof schema>

const categoryPrefixes: Record<string, string> = {
  'Office Supplies': 'OFF',
  'IT Equipment': 'IT',
  Consumables: 'CON',
  Furniture: 'FUR',
  'Facility & Safety': 'FAC',
}

function categoryPrefix(name: string): string {
  const mapped = categoryPrefixes[name]
  if (mapped) return mapped
  const letters = name.replace(/[^A-Za-z]/g, '').toUpperCase()
  return (letters.slice(0, 3) || 'GEN').padEnd(2, 'X').slice(0, 3)
}

function nameInitials(name: string): string {
  const words = name
    .split(/[^A-Za-z0-9]+/)
    .map((word) => word.replace(/[^A-Za-z0-9]/g, ''))
    .filter(Boolean)
  const initials = words.slice(0, 3).map((word) => word[0]).join('').toUpperCase()
  if (initials.length >= 2) return initials
  const fallback = words.join('').slice(0, 3).toUpperCase()
  return (fallback || 'GEN').padEnd(2, 'X')
}

export function ProductFormPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const { t } = useT()
  const isSupervisor = user?.role === 'SUPERVISOR'
  const editing = Boolean(id)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [submitting, setSubmitting] = useState(false)
  const [generatingSku, setGeneratingSku] = useState(false)
  const product = useQuery({ queryKey: ['product', id], queryFn: () => productApi.get(id!), enabled: editing })
  const references = useQuery({
    queryKey: ['references'],
    queryFn: async () => {
      const [categories, units] = await Promise.all([referenceApi.categories(), referenceApi.units()])
      return { categories: categories.data, units: units.data }
    },
  })
  const { register, handleSubmit, reset, setError, setValue, watch, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { sku: '', name: '', description: '', category_id: '', unit_id: '', min_stock: 0, unit_cost: '' },
  })
  const categoryId = watch('category_id')
  const unitId = watch('unit_id')
  const unitCost = watch('unit_cost')
  const selectedCategory = references.data?.categories.find((c) => c.id === categoryId)
  const selectedUnit = references.data?.units.find((u) => u.id === unitId)
  const pricePreview = Number(unitCost)

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

  const generateSku = async () => {
    const name = watch('name')?.trim()
    if (!selectedCategory) {
      setError('category_id', { message: 'validation.selectCategory' })
      return
    }
    if (!name) {
      setError('name', { message: 'validation.required' })
      return
    }
    setGeneratingSku(true)
    try {
      const prefix = categoryPrefix(selectedCategory.name)
      const initials = nameInitials(name)
      const res = await productApi.list({ category_id: selectedCategory.id, page_size: 100 })
      const pattern = new RegExp(`^${prefix}-${initials}-(\\d{3})$`)
      const highest = res.items.reduce((max, p) => {
        const match = p.sku.match(pattern)
        return match ? Math.max(max, Number(match[1])) : max
      }, 0)
      setValue('sku', `${prefix}-${initials}-${String(highest + 1).padStart(3, '0')}`, { shouldValidate: true })
    } finally {
      setGeneratingSku(false)
    }
  }

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
      toast.success(editing ? t('toast.productUpdated') : t('toast.productCreated'))
      await queryClient.invalidateQueries({ queryKey: ['products'] })
      await queryClient.invalidateQueries({ queryKey: ['product', id] })
      await queryClient.invalidateQueries({ queryKey: ['inventory'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      navigate(editing ? `/products/${id}` : '/products')
    } catch (err) {
      const message = err instanceof ApiError ? err.message : t('toast.saveFailed')
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
      <PageHeader
        title={editing ? t('pages.editProduct') : t('pages.newProduct')}
        description={t('pages.productFormDescription')}
      />
      <Card className="p-6">
        <form className="space-y-4" onSubmit={handleSubmit(onSubmit)} noValidate>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              label={t('fields.sku')}
              hint={t('fields.skuHint')}
              example={t('fields.skuExample')}
              htmlFor="sku"
              error={errors.sku?.message}
            >
              <div className="flex gap-2">
                <Input id="sku" className="uppercase" {...register('sku')} />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="shrink-0"
                  loading={generatingSku}
                  title={t('fields.autoSkuHint')}
                  onClick={generateSku}
                >
                  {t('actions.autoGenerate')}
                </Button>
              </div>
            </Field>
            <Field
              label={t('fields.name')}
              hint={t('fields.nameHint')}
              example={t('fields.nameExample')}
              htmlFor="name"
              error={errors.name?.message}
            >
              <Input id="name" {...register('name')} />
            </Field>
            <Field
              label={t('fields.category')}
              hint={t('fields.categoryHint')}
              htmlFor="category_id"
              error={errors.category_id?.message}
            >
              <Select id="category_id" {...register('category_id')}>
                <option value="">{t('fields.categoryPlaceholder')}</option>
                {references.data?.categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </Select>
            </Field>
            <Field
              label={t('fields.unit')}
              hint={t('fields.unitHint')}
              htmlFor="unit_id"
              error={errors.unit_id?.message}
            >
              <Select id="unit_id" {...register('unit_id')}>
                <option value="">{t('fields.unitPlaceholder')}</option>
                {references.data?.units.map((u) => <option key={u.id} value={u.id}>{u.code} — {u.name}</option>)}
              </Select>
            </Field>
            <Field
              label={t('fields.minStock')}
              hint={t('fields.minStockHint')}
              example={t('fields.minStockExample')}
              htmlFor="min_stock"
              error={errors.min_stock?.message}
            >
              <div className="flex items-center gap-2">
                <Input id="min_stock" type="number" min={0} {...register('min_stock')} />
                <span className="shrink-0 text-sm text-slate-500">{selectedUnit?.code ?? t('common.unit')}</span>
              </div>
            </Field>
            {isSupervisor && (
              <Field
                label={t('fields.unitPrice')}
                hint={t('fields.unitPriceHint')}
                example={t('fields.unitPriceExample')}
                htmlFor="unit_cost"
                error={errors.unit_cost?.message}
              >
                <Input id="unit_cost" type="number" step="1" min={0} {...register('unit_cost')} />
                {unitCost !== '' && !Number.isNaN(pricePreview) && (
                  <p className="mt-1 text-xs text-slate-500">{formatCurrency(pricePreview)}</p>
                )}
              </Field>
            )}
          </div>
          <Field
            label={t('fields.description')}
            hint={t('fields.descriptionHint')}
            example={t('fields.descriptionExample')}
            htmlFor="description"
            error={errors.description?.message}
          >
            <Textarea id="description" {...register('description')} />
          </Field>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => navigate(-1)}>{t('common.cancel')}</Button>
            <Button type="submit" loading={submitting}>{editing ? t('actions.saveChanges') : t('actions.createProduct')}</Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
