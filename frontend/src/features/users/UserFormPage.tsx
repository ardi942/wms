import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { referenceApi, userApi } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { Button, Card, CardHeader, Field, Input, Select } from '../../components/ui'
import { ErrorState, Loading, PageHeader } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { AuditEventTable } from '../audit/AuditEventTable'
import { UserStatusDialog } from './UserStatusDialog'

const schema = z.object({
  email: z.string().email('validation.email'),
  full_name: z.string().min(2, 'validation.nameMin').max(200, 'validation.tooLong'),
  password: z.string().optional(),
  department_id: z.string().uuid('validation.selectDepartment'),
})
type FormValues = z.infer<typeof schema>

export function UserFormPage() {
  const { id } = useParams()
  const { t } = useT()
  const editing = Boolean(id)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [submitting, setSubmitting] = useState(false)
  const [statusOpen, setStatusOpen] = useState(false)
  const [statusSaving, setStatusSaving] = useState(false)
  const user = useQuery({ queryKey: ['user', id], queryFn: () => userApi.get(id!), enabled: editing })
  const history = useQuery({
    queryKey: ['user-history', id],
    queryFn: () => userApi.history(id!, { page: 1, page_size: 10 }),
    enabled: editing,
  })
  const departments = useQuery({ queryKey: ['departments'], queryFn: () => referenceApi.departments() })
  const { register, handleSubmit, reset, setError, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '', full_name: '', password: '', department_id: '' },
  })

  useEffect(() => {
    if (user.data) {
      const u = user.data.data
      reset({ email: u.email, full_name: u.full_name, password: '', department_id: u.department?.id ?? '' })
    }
  }, [user.data, reset])

  const onSubmit = async (values: FormValues) => {
    if (!editing && (!values.password || values.password.length < 8)) {
      setError('password', { message: 'validation.passwordMin' })
      return
    }
    if (values.password && values.password.length < 8) {
      setError('password', { message: 'validation.passwordMin' })
      return
    }
    setSubmitting(true)
    try {
      if (editing) {
        await userApi.update(id!, {
          full_name: values.full_name.trim(),
          department_id: values.department_id,
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
      toast.success(editing ? t('toast.userUpdated') : t('toast.userCreated'))
      await queryClient.invalidateQueries({ queryKey: ['users'] })
      if (editing) {
        await queryClient.invalidateQueries({ queryKey: ['user', id] })
        await queryClient.invalidateQueries({ queryKey: ['user-history', id] })
      }
      navigate('/users')
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

  const changeStatus = async (reason: string) => {
    if (!user.data) return
    setStatusSaving(true)
    try {
      const active = user.data.data.is_active
      if (active) await userApi.deactivate(user.data.data.id, reason)
      else await userApi.activate(user.data.data.id, reason)
      toast.success(active ? t('toast.userDeactivated') : t('toast.userActivated'))
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['user', id] }),
        queryClient.invalidateQueries({ queryKey: ['users'] }),
        queryClient.invalidateQueries({ queryKey: ['user-history', id] }),
      ])
      setStatusOpen(false)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t('toast.saveFailed'))
    } finally {
      setStatusSaving(false)
    }
  }

  if (editing && user.isLoading) return <Loading />
  if (editing && user.error) return <ErrorState error={user.error} onRetry={() => user.refetch()} />
  const selectedUser = user.data?.data ?? null

  return (
    <div className="mx-auto max-w-4xl">
      <PageHeader
        title={editing ? t('pages.editUser') : t('pages.newUser')}
        description={t('pages.userFormDescription')}
        actions={editing && selectedUser ? (
          <Button
            variant={selectedUser.is_active ? 'danger' : 'primary'}
            onClick={() => setStatusOpen(true)}
          >
            {selectedUser.is_active ? t('actions.deactivate') : t('actions.activate')}
          </Button>
        ) : undefined}
      />
      <div className="max-w-xl">
        <Card className="p-6">
          <form className="space-y-4" onSubmit={handleSubmit(onSubmit)} noValidate>
            <Field
              label={t('fields.email')}
              hint={t('fields.emailHint')}
              example={t('fields.emailExample')}
              htmlFor="email"
              error={errors.email?.message}
            >
              <Input id="email" type="email" disabled={editing} {...register('email')} />
            </Field>
            <Field
              label={t('fields.fullName')}
              example={t('fields.fullNameExample')}
              htmlFor="full_name"
              error={errors.full_name?.message}
            >
              <Input id="full_name" {...register('full_name')} />
            </Field>
            <Field
              label={t('fields.department')}
              hint={t('fields.departmentHint')}
              htmlFor="department_id"
              error={errors.department_id?.message}
            >
              <Select id="department_id" {...register('department_id')}>
                <option value="">{t('fields.departmentPlaceholder')}</option>
                {departments.data?.data.map((d) => <option key={d.id} value={d.id}>{d.code} — {d.name}</option>)}
              </Select>
            </Field>
            <Field
              label={editing ? t('fields.newPassword') : t('fields.password')}
              hint={editing ? t('fields.passwordKeepHint') : t('fields.passwordHint')}
              htmlFor="password"
              error={errors.password?.message}
            >
              <Input id="password" type="password" autoComplete="new-password" {...register('password')} />
            </Field>
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end [&>*]:w-full sm:[&>*]:w-auto">
              <Button type="button" variant="outline" onClick={() => navigate(-1)}>{t('common.cancel')}</Button>
              <Button type="submit" loading={submitting}>{editing ? t('actions.saveChanges') : t('actions.createUser')}</Button>
            </div>
          </form>
        </Card>
      </div>
      {editing && (
        <Card className="mt-4">
          <CardHeader title={t('audit.userHistory')} description={t('audit.userHistoryDescription')} />
          <AuditEventTable
            events={history.data?.items ?? []}
            loading={history.isLoading}
            error={history.error}
            onRetry={() => history.refetch()}
            showSubject={false}
          />
        </Card>
      )}
      <UserStatusDialog
        open={statusOpen}
        user={selectedUser}
        loading={statusSaving}
        onCancel={() => setStatusOpen(false)}
        onConfirm={changeStatus}
      />
    </div>
  )
}
