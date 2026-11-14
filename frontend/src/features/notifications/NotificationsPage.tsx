import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { notificationApi } from '../../api/endpoints'
import { Button, Card } from '../../components/ui'
import { DataTable, FilterBar, PageHeader, Pagination, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { formatDateTime } from '../../lib/format'
import type { Notification } from '../../types'

export function NotificationsPage() {
  const { t } = useT()
  const queryClient = useQueryClient()
  const [unreadOnly, setUnreadOnly] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const filters = { unread_only: unreadOnly || undefined, page, page_size: pageSize }
  const notifications = useQuery({ queryKey: ['notifications', filters], queryFn: () => notificationApi.list(filters) })

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: ['notifications'] })
  }

  const markRead = async (n: Notification) => {
    if (n.is_read) return
    try {
      await notificationApi.markRead(n.id)
      await refresh()
    } catch {
      toast.error(t('toast.markReadFailed'))
    }
  }

  const markAll = async () => {
    try {
      await notificationApi.markAllRead()
      await refresh()
      toast.success(t('toast.allMarkedRead'))
    } catch {
      toast.error(t('toast.markAllFailed'))
    }
  }

  const linkFor = (n: Notification): string | null => {
    if (n.related_type === 'stock_request' && n.related_id) return `/requests/${n.related_id}`
    if (n.related_type === 'stock_return' && n.related_id) return `/returns/${n.related_id}`
    return null
  }

  const columns: Column<Notification>[] = [
    { key: 'status', header: '', render: (n) => n.is_read ? <span className="block h-2 w-2 rounded-full bg-slate-200" /> : <span className="block h-2 w-2 rounded-full bg-brand-500" /> },
    { key: 'title', header: t('details.item'), render: (n) => (
      <div>
        <p className={n.is_read ? 'text-slate-600' : 'font-semibold text-slate-900'}>{n.title}</p>
        <p className="text-sm text-slate-500 line-clamp-1">{n.body}</p>
      </div>
    ), tooltip: (n) => n.body },
    { key: 'when', header: t('details.when'), render: (n) => formatDateTime(n.created_at) },
    { key: 'actions', header: '', render: (n) => (
      <div className="flex justify-end gap-2">
        {linkFor(n) && <Link to={linkFor(n)!}><Button variant="outline" size="sm">{t('actions.open')}</Button></Link>}
        {!n.is_read && <Button variant="ghost" size="sm" onClick={(e) => { e.stopPropagation(); markRead(n) }}>{t('actions.markRead')}</Button>}
      </div>
    ) },
  ]

  return (
    <div>
      <PageHeader
        title={t('pages.notifications')}
        description={t('pages.notificationsDescription')}
        actions={<Button variant="outline" onClick={markAll}>{t('actions.markAllRead')}</Button>}
      />
      <Card>
        <FilterBar>
          <label className="flex items-center gap-2 text-sm text-slate-600">
            <input type="checkbox" checked={unreadOnly} onChange={(e) => { setUnreadOnly(e.target.checked); setPage(1) }} className="h-4 w-4 rounded border-slate-300" /> {t('filters.unreadOnly')}
          </label>
        </FilterBar>
        <DataTable
          columns={columns}
          rows={notifications.data?.items ?? []}
          loading={notifications.isLoading}
          error={notifications.error}
          onRetry={() => notifications.refetch()}
          emptyTitle={t('empty.noNotifications')}
          onRowClick={(n) => { void markRead(n) }}
          getRowKey={(n) => n.id}
          renderExpanded={(n) => (
            <dl className="grid gap-2 text-sm sm:grid-cols-2">
              <div className="sm:col-span-2"><dt className="text-xs uppercase text-slate-500">{t('details.message')}</dt><dd>{n.body}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.type')}</dt><dd>{n.type}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.received')}</dt><dd>{formatDateTime(n.created_at)}</dd></div>
            </dl>
          )}
        />
        {notifications.data && <Pagination meta={notifications.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
