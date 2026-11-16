import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { requestApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, Select, Tooltip } from '../../components/ui'
import { DataTable, FilterBar, Pagination, PageHeader, SearchInput, StatusBadge, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { LIST_POLL_MS } from '../../lib/constants'
import { formatDateTime, formatQty } from '../../lib/format'
import type { StockRequestList } from '../../types'

export function RequestListPage() {
  const { user } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [status, setStatus] = useState(params.get('status') ?? '')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const filters = { status, search, page, page_size: pageSize }
  const requests = useQuery({
    queryKey: ['requests', filters],
    queryFn: () => requestApi.list(filters),
    refetchInterval: LIST_POLL_MS,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    placeholderData: keepPreviousData,
  })

  const columns: Column<StockRequestList>[] = [
    { key: 'number', header: t('details.requestNumber'), render: (r) => <span className="font-medium text-brand-700">{r.request_number}</span> },
    { key: 'department', header: t('details.department'), render: (r) => r.department.name },
    { key: 'purpose', header: t('details.purpose'), render: (r) => <span className="line-clamp-1">{r.purpose}</span>, tooltip: (r) => r.purpose },
    { key: 'items', header: t('details.lines'), render: (r) => formatQty(r.item_count) },
    { key: 'qty', header: t('details.issuedRequested'), render: (r) => `${formatQty(r.total_issued)} / ${formatQty(r.total_requested)}` },
    { key: 'status', header: t('details.status'), render: (r) => <StatusBadge status={r.status} /> },
    {
      key: 'actions',
      header: '',
      className: 'text-right',
      render: (r) =>
        (r.allowed_actions?.length ?? 0) > 0 ? (
          <Link
            to={`/requests/${r.id}`}
            onClick={(e) => e.stopPropagation()}
            className="font-medium text-brand-600 hover:underline"
          >
            {t('actions.review')}
          </Link>
        ) : (
          <Tooltip content={t('actions.none')}>
            <span className="text-xs text-slate-400">—</span>
          </Tooltip>
        ),
    },
    { key: 'created', header: t('details.created'), render: (r) => formatDateTime(r.created_at) },
  ]

  return (
    <div>
      <PageHeader
        title={t('pages.requests')}
        description={user?.role === 'DEPARTMENT_USER' ? t('pages.requestsDescriptionDept') : t('pages.requestsDescriptionAdmin')}
        actions={user?.role === 'DEPARTMENT_USER' ? <Link to="/requests/new"><Button><Plus className="h-4 w-4" /> {t('actions.newRequest')}</Button></Link> : undefined}
      />
      <Card>
        <FilterBar>
          <SearchInput value={search} onChange={(v) => { setSearch(v); setPage(1) }} placeholder={t('common.searchNumberOrPurpose')} />
          <Select className="h-11 w-full sm:h-9 sm:w-48" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
            <option value="">{t('common.allStatuses')}</option>
            {['PENDING', 'PARTIALLY_FULFILLED', 'FULFILLED', 'REJECTED', 'CANCELLED'].map((s) => <option key={s} value={s}>{t(`status.${s}`)}</option>)}
          </Select>
        </FilterBar>
        <DataTable
          columns={columns}
          rows={requests.data?.items ?? []}
          loading={requests.isLoading}
          error={requests.error}
          onRetry={() => requests.refetch()}
          onRowClick={(r) => navigate(`/requests/${r.id}`)}
          getRowKey={(r) => r.id}
          renderExpanded={(r) => (
            <dl className="grid gap-2 text-sm sm:grid-cols-2">
              <div><dt className="text-xs uppercase text-slate-500">{t('details.purpose')}</dt><dd>{r.purpose}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.requestedBy')}</dt><dd>{r.requested_by.full_name}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.issuedRequested')}</dt><dd>{formatQty(r.total_issued)} / {formatQty(r.total_requested)}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.updated')}</dt><dd>{formatDateTime(r.updated_at)}</dd></div>
            </dl>
          )}
        />
        {requests.data && <Pagination meta={requests.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
