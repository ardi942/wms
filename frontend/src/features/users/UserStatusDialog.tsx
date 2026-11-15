import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { Button, Card, Label, Textarea } from '../../components/ui'
import { useT } from '../../i18n/I18nProvider'
import type { User } from '../../types'

export function UserStatusDialog({
  open,
  user,
  loading,
  onCancel,
  onConfirm,
}: {
  open: boolean
  user: User | null
  loading?: boolean
  onCancel: () => void
  onConfirm: (reason: string) => void
}) {
  const { t } = useT()
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (open) {
      setReason('')
      setError('')
    }
  }, [open, user?.id])

  if (!open || !user) return null
  const deactivating = user.is_active
  const title = deactivating ? t('dialogs.deactivateUser') : t('dialogs.activateUser')

  const submit = () => {
    const value = reason.trim()
    if (value.length < 5) {
      setError('validation.reasonRequired')
      return
    }
    if (value.length > 500) {
      setError('validation.tooLong')
      return
    }
    onConfirm(value)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center">
      <Card className="max-h-[90dvh] w-full max-w-md overflow-y-auto p-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))] sm:pb-5">
        <div className="flex items-start justify-between gap-4">
          <h3 className="text-lg font-semibold text-slate-900">{title}</h3>
          <button
            type="button"
            onClick={onCancel}
            className="grid h-10 w-10 place-items-center rounded-md text-slate-400 hover:bg-slate-100 hover:text-slate-600"
            aria-label={t('common.close')}
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <p className="mt-3 text-sm text-slate-600">
          {t(deactivating ? 'dialogs.deactivateUserBody' : 'dialogs.activateUserBody', { name: user.full_name })}
        </p>
        <div className="mt-4">
          <Label htmlFor="status-reason">{t('fields.statusReason')}</Label>
          <Textarea
            id="status-reason"
            value={reason}
            maxLength={500}
            onChange={(event) => {
              setReason(event.target.value)
              if (error) setError('')
            }}
            placeholder={t('dialogs.reasonPlaceholder')}
            aria-invalid={Boolean(error)}
            aria-describedby={error ? 'status-reason-error' : 'status-reason-hint'}
          />
          <div id="status-reason-hint" className="mt-1 text-xs text-slate-500">{t('fields.statusReasonHint')}</div>
          {error && <p id="status-reason-error" role="alert" className="mt-1 text-xs font-medium text-red-600">{t(error)}</p>}
        </div>
        <div className="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end [&>*]:w-full sm:[&>*]:w-auto">
          <Button variant="outline" onClick={onCancel}>{t('common.cancel')}</Button>
          <Button variant={deactivating ? 'danger' : 'primary'} loading={loading} onClick={submit}>
            {deactivating ? t('actions.deactivate') : t('actions.activate')}
          </Button>
        </div>
      </Card>
    </div>
  )
}
