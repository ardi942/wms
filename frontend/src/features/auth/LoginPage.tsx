import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, Field, Input } from '../../components/ui'
import { ApiError } from '../../api/client'

const schema = z.object({
  email: z.string().email('Enter a valid email'),
  password: z.string().min(1, 'Password is required'),
})
type FormValues = z.infer<typeof schema>

export function LoginPage() {
  const { login, user } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [submitting, setSubmitting] = useState(false)
  const { register, handleSubmit, formState: { errors }, setError } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '', password: '' },
  })

  useEffect(() => {
    if (user) navigate('/dashboard', { replace: true })
  }, [user, navigate])
  if (user) return null

  const onSubmit = async (values: FormValues) => {
    setSubmitting(true)
    try {
      await login(values.email, values.password)
      toast.success('Welcome back')
      const next = params.get('next')
      navigate(next ? decodeURIComponent(next) : '/dashboard', { replace: true })
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'Login failed'
      if (err instanceof ApiError && err.details) {
        for (const [field, value] of Object.entries(err.details)) {
          if (field === 'email' || field === 'password') setError(field, { message: String(value) })
        }
      }
      toast.error(message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-full items-center justify-center bg-gradient-to-br from-slate-900 via-slate-800 to-brand-900 p-4">
      <div className="w-full max-w-md">
        <div className="mb-6 text-center text-white">
          <div className="mx-auto mb-3 grid h-12 w-12 place-items-center rounded-xl bg-brand-500 text-xl font-bold">W</div>
          <h1 className="text-2xl font-semibold">Warehouse Management</h1>
          <p className="mt-1 text-sm text-slate-300">Sign in to manage stock, requests and returns.</p>
        </div>
        <Card className="p-6">
          <form className="space-y-4" onSubmit={handleSubmit(onSubmit)} noValidate>
            <Field label="Email" htmlFor="email" error={errors.email?.message}>
              <Input id="email" type="email" autoComplete="email" placeholder="you@wms.local" {...register('email')} />
            </Field>
            <Field label="Password" htmlFor="password" error={errors.password?.message}>
              <Input id="password" type="password" autoComplete="current-password" placeholder="••••••••" {...register('password')} />
            </Field>
            <Button type="submit" className="w-full" loading={submitting}>Sign in</Button>
          </form>
        </Card>
        <div className="mt-4 rounded-lg bg-white/10 p-3 text-xs text-slate-200">
          <p className="font-medium text-white">Demo accounts</p>
          <p className="mt-1">supervisor@wms.local / Supervisor123!</p>
          <p>warehouse@wms.local / Warehouse123!</p>
          <p>staff@wms.local / Staff123!</p>
        </div>
      </div>
    </div>
  )
}
