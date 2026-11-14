import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { referenceApi, userApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { Button, Card, Field, Input, Select } from '../../components/ui'
import { ErrorState, Loading, PageHeader } from '../../components/common'

const schema = z.object({
  email: z.string().email(),
  full_name: z.string().min(2).max(200),
  password: z.string().optional(),
  department_id: z.string().uuid('Choose a department'),
  is_active: z.boolean(),
})
type FormValues = z.infer<typeof schema>

export function UserFormPage() {
  const { id } = useParams()
  const editing = Boolean(id)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [submitting, setSubmitting] = useState(false)
  const user = useQuery({ queryKey: ['user', id], queryFn: () => userApi.get(id!), enabled: editing })
  const departments = useQuery({ queryKey: ['departments'], queryFn: () => referenceApi.departments() })
  const { register, handleSubmit, reset, setError, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '', full_name: '', password: '', department_id: '', is_active: true },
  })

  useEffect(() => {
    if (user.data) {
      const u = user.data.data
      reset({ email: u.email, full_name: u.full_name, password: '', department_id: u.department?.id ?? '', is_active: u.is_active })
    }
  }, [user.data, reset])

  const onSubmit = async (values: FormValues) => {
    if (!editing && (!values.password || values.password.length < 8)) {
      setError('password', { message: 'Password must be at least 8 characters' })
      return
    }
    if (values.password && values.password.length < 8) {
      setError('password', { message: 'Password must be at least 8 characters' })
      return
    }
    setSubmitting(true)
    try {
      if (editing) {
        await userApi.update(id!, {
          full_name: values.full_name.trim(),
          department_id: values.department_id,
          is_active: values.is_active,
          ...(values.password ? { password: values.password } : {}),
        })
      } else {
        await userApi.create({
          email: values.email.trim(),
          full_name: values.full_name.trim(),
          password: values.password,
          department_id: values.department_id,
        })
      }
      toast.success(editing ? 'User updated' : 'User created')
      await queryClient.invalidateQueries({ queryKey: ['users'] })
      navigate('/users')
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

  if (editing && user.isLoading) return <Loading />
  if (editing && user.error) return <ErrorState error={user.error} onRetry={() => user.refetch()} />

  return (
    <div className="mx-auto max-w-xl">
      <PageHeader title={editing ? 'Edit user' : 'New department user'} description="Department users can create requests and returns for their department." />
      <Card className="p-6">
        <form className="space-y-4" onSubmit={handleSubmit(onSubmit)} noValidate>
          <Field label="Email" htmlFor="email" error={errors.email?.message}>
            <Input id="email" type="email" disabled={editing} {...register('email')} />
          </Field>
          <Field label="Full name" htmlFor="full_name" error={errors.full_name?.message}>
            <Input id="full_name" {...register('full_name')} />
          </Field>
          <Field label="Department" htmlFor="department_id" error={errors.department_id?.message}>
            <Select id="department_id" {...register('department_id')}>
              <option value="">Select department</option>
              {departments.data?.data.map((d) => <option key={d.id} value={d.id}>{d.code} — {d.name}</option>)}
            </Select>
          </Field>
          <Field label={editing ? 'New password (optional)' : 'Password'} htmlFor="password" error={errors.password?.message}>
            <Input id="password" type="password" autoComplete="new-password" {...register('password')} />
          </Field>
          {editing && (
            <label className="flex items-center gap-2 text-sm text-slate-700">
              <input type="checkbox" className="h-4 w-4 rounded border-slate-300" {...register('is_active')} /> Active
            </label>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => navigate(-1)}>Cancel</Button>
            <Button type="submit" loading={submitting}>{editing ? 'Save changes' : 'Create user'}</Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
