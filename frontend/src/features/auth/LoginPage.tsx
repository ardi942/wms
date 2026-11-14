import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuth } from '../../auth/AuthProvider'
import { useT } from '../../i18n/I18nProvider'
import { Button, Card, Field, Input } from '../../components/ui'
import { ApiError } from '../../api/client'
import { Mail, Lock, Eye, EyeOff } from 'lucide-react'
import { LanguageSwitcher } from '../../i18n/I18nProvider'

const REMEMBER_KEY = 'wms.login.email'

const schema = z.object({
  email: z.string().email('validation.email'),
  password: z.string().min(1, 'validation.passwordRequired'),
})
type FormValues = z.infer<typeof schema>

export function LoginPage() {
  const { login, user } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [submitting, setSubmitting] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const remembered = typeof localStorage !== 'undefined' ? localStorage.getItem(REMEMBER_KEY) ?? '' : ''
  const [remember, setRemember] = useState(Boolean(remembered))

  const { register, handleSubmit, formState: { errors }, setError } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: remembered, password: '' },
  })

  useEffect(() => {
    if (user) navigate('/dashboard', { replace: true })
  }, [user, navigate])
  if (user) return null

  const onSubmit = async (values: FormValues) => {
    setSubmitting(true)
    try {
      await login(values.email, values.password)
      if (remember) localStorage.setItem(REMEMBER_KEY, values.email)
      else localStorage.removeItem(REMEMBER_KEY)
      toast.success(t('login.welcome'))
      const next = params.get('next')
      navigate(next ? decodeURIComponent(next) : '/dashboard', { replace: true })
    } catch (err) {
      const message = err instanceof ApiError ? err.message : t('login.failed')
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
    <div className="grid min-h-dvh lg:grid-cols-2">
      <div className="hidden lg:flex-col justify-between bg-gradient-to-br from-slate-900 via-slate-800 to-brand-900 p-10 lg:text-white">
        <div className="flex items-center gap-3">
          <div className="grid h-10 w-10 place-items-center rounded-xl bg-brand-500 text-xl font-bold">W</div>
          <div>
            <p className="text-lg font-semibold">{t('login.brand')}</p>
            <p className="mt-1 max-w-sm text-sm text-slate-300">{t('login.tagline')}</p>
          </div>
        </div>
        <div>
          <p className="text-xs text-slate-400">{t('common.copyright', { year: new Date().getFullYear() })}</p>
          <ul className="mt-6 space-y-2 text-sm text-slate-300">
            <li>{t('nav.inventory')}</li>
            <li>{t('nav.requests')}</li>
            <li>{t('nav.reports')}</li>
          </ul>
        </div>
      </div>

      <div className="flex items-center justify-center bg-slate-100 p-4">
        <div className="w-full max-w-md">
          <div className="mb-4 flex justify-end">
            <LanguageSwitcher />
          </div>
          <Card className="p-5 sm:p-6">
            <div className="mb-5">
              <h2 className="text-xl font-semibold text-slate-900">{t('login.signIn')}</h2>
              <p className="mt-1 text-sm text-slate-500">{t('login.tagline')}</p>
            </div>
            <form className="space-y-4" onSubmit={handleSubmit(onSubmit)} noValidate>
              <Field label={t('login.email')} htmlFor="email" error={errors.email?.message}>
                <div className="relative">
                  <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                  <Input
                    id="email"
                    type="email"
                    autoComplete="email"
                    placeholder={t('login.email')}
                    className="pl-9"
                    {...register('email')}
                  />
                </div>
              </Field>
              <Field label={t('login.password')} htmlFor="password" error={errors.password?.message}>
                <div className="relative">
                  <Lock className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                  <Input
                    id="password"
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="current-password"
                    placeholder="••••••••"
                    className="pl-9 pr-10"
                    {...register('password')}
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword((v) => !v)}
                    className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-slate-400 hover:text-slate-600"
                    aria-label={showPassword ? t('login.hidePassword') : t('login.showPassword')}
                  >
                    {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
              </Field>
              <label className="flex items-center gap-2 text-sm text-slate-600">
                <input
                  type="checkbox"
                  checked={remember}
                  onChange={(e) => setRemember(e.target.checked)}
                  className="h-4 w-4 rounded border-slate-300"
                />
                {t('login.remember')}
              </label>
              <Button type="submit" className="w-full" loading={submitting}>{t('login.signIn')}</Button>
            </form>
          </Card>
          <details className="mt-4 rounded-lg bg-white/70 p-3 text-xs text-slate-600 shadow-sm">
            <summary className="cursor-pointer font-medium text-slate-700">{t('login.demo')}</summary>
            <div className="mt-2 space-y-1">
              <p>supervisor@wms.local / Supervisor123!</p>
              <p>warehouse@wms.local / Warehouse123!</p>
              <p>staff@wms.local / Staff123!</p>
            </div>
          </details>
        </div>
      </div>
    </div>
  )
}