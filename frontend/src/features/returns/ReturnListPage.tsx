import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { returnApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, Select, Tooltip } from '../../components/ui'
import { DataTable, FilterBar, Pagination, PageHeader, SearchInput, StatusBadge, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { LIST_POLL_MS } from '../../lib/constants'
import { formatDateTime, formatQty } from '../../lib/format'
import type { StockReturnList } from '../../types'

export function ReturnListPage() {
  const { user } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const filters = { status, search, page, page_size: pageSize }
  const returns = useQuery({
    queryKey: ['returns', filters],
    queryFn: () => returnApi.list(filters),
    refetchInterval: LIST_POLL_MS,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    placeholderData: keepPreviousData,
  })

  const columns: Column<StockReturnList>[] = [
    { key: 'number', header: t('details.returnNumber'), render: (r) => <span className="font-medium text-brand-700">{r.return_number}</span> },
    { key: 'department', header: t('details.department'), render: (r) => r.department.name },
    { key: 'reason', header: t('details.reason'), render: (r) => <span className="line-clamp-1">{r.reason}</span>, tooltip: (r) => r.reason },
    { key: 'items', header: t('details.lines'), render: (r) => formatQty(r.item_count) },
    { key: 'qty', header: t('details.acceptedRequested'), render: (r) => `${formatQty(r.total_accepted)} / ${formatQty(r.total_requested)}` },
    { key: 'status', header: t('details.status'), render: (r) => <StatusBadge status={r.status} /> },
    {
      key: 'actions',
      header: '',
      className: 'text-right',
      render: (r) =>
        (r.allowed_actions?.length ?? 0) > 0 ? (
          <Link
            to={`/returns/${r.id}`}
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
        title={t('pages.returns')}
        description={user?.role === 'DEPARTMENT_USER' ? t('pages.returnsDescriptionDept') : t('pages.returnsDescriptionAdmin')}
        actions={user?.role === 'DEPARTMENT_USER' ? <Link to="/returns/new"><Button><Plus className="h-4 w-4" /> {t('actions.newReturn')}</Button></Link> : undefined}
      />
      <Card>
        <FilterBar>
          <SearchInput value={search} onChange={(v) => { setSearch(v); setPage(1) }} placeholder={t('common.searchNumberOrReason')} />
          <Select className="h-11 w-full sm:h-9 sm:w-48" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
            <option value="">{t('common.allStatuses')}</option>
            {['PENDING', 'CONFIRMED', 'REJECTED', 'CANCELLED'].map((s) => <option key={s} value={s}>{t(`status.${s}`)}</option>)}
          </Select>
        </FilterBar>
        <DataTable
          columns={columns}
          rows={returns.data?.items ?? []}
          loading={returns.isLoading}
          error={returns.error}
          onRetry={() => returns.refetch()}
          onRowClick={(r) => navigate(`/returns/${r.id}`)}
          getRowKey={(r) => r.id}
          renderExpanded={(r) => (
            <dl className="grid gap-2 text-sm sm:grid-cols-2">
              <div><dt className="text-xs uppercase text-slate-500">{t('details.reason')}</dt><dd>{r.reason}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.requestedBy')}</dt><dd>{r.requested_by.full_name}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.acceptedRequested')}</dt><dd>{formatQty(r.total_accepted)} / {formatQty(r.total_requested)}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.updated')}</dt><dd>{formatDateTime(r.updated_at)}</dd></div>
            </dl>
          )}
        />
        {returns.data && <Pagination meta={returns.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
