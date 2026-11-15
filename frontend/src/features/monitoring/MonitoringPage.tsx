import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { monitoringApi, referenceApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Card, Select } from '../../components/ui'
import {
  AutoRefreshIndicator,
  DataTable,
  FilterBar,
  PageHeader,
  Pagination,
  SearchInput,
  StatusBadge,
  type Column,
} from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { MONITORING_POLL_MS } from '../../lib/constants'
import { formatDateTime, formatQty } from '../../lib/format'
import { cn } from '../../lib/utils'
import type { MonitoringRequestRow, MonitoringReturnRow } from '../../types'

type Tab = 'requests' | 'returns'

const requestStatuses = ['PENDING', 'PARTIALLY_FULFILLED', 'FULFILLED', 'REJECTED', 'CANCELLED']
const returnStatuses = ['PENDING', 'CONFIRMED', 'REJECTED', 'CANCELLED']

export function MonitoringPage() {
  const { user } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const [tab, setTab] = useState<Tab>('requests')
  const [status, setStatus] = useState('')
  const [departmentId, setDepartmentId] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const filters = { status, department_id: departmentId, search, page, page_size: pageSize }
  const departments = useQuery({ queryKey: ['departments'], queryFn: () => referenceApi.departments() })

  const requests = useQuery({
    queryKey: ['monitoring', 'requests', filters],
    queryFn: () => monitoringApi.requests(filters),
    enabled: tab === 'requests',
    refetchInterval: MONITORING_POLL_MS,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    placeholderData: keepPreviousData,
  })

  const returns = useQuery({
    queryKey: ['monitoring', 'returns', filters],
    queryFn: () => monitoringApi.returns(filters),
    enabled: tab === 'returns',
    refetchInterval: MONITORING_POLL_MS,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    placeholderData: keepPreviousData,
  })

  const active = tab === 'requests' ? requests : returns
  const canOpenDetail = (ownerDepartmentId: string) =>
    user?.role !== 'DEPARTMENT_USER' || user.department?.id === ownerDepartmentId

  const requestColumns: Column<MonitoringRequestRow>[] = [
    { key: 'number', header: t('details.requestNumber'), render: (r) => <span className="font-medium text-brand-700">{r.request_number}</span> },
    { key: 'department', header: t('details.department'), render: (r) => r.department.name },
    { key: 'requester', header: t('details.requestedBy'), render: (r) => r.requested_by.full_name },
    { key: 'status', header: t('details.status'), render: (r) => <StatusBadge status={r.status} /> },
    { key: 'lines', header: t('details.lines'), render: (r) => formatQty(r.item_count) },
    { key: 'progress', header: t('details.issuedRequested'), render: (r) => `${formatQty(r.total_issued)} / ${formatQty(r.total_requested)}` },
    { key: 'updated', header: t('details.updated'), render: (r) => formatDateTime(r.updated_at) },
    { key: 'created', header: t('details.created'), render: (r) => formatDateTime(r.created_at) },
  ]

  const returnColumns: Column<MonitoringReturnRow>[] = [
    { key: 'number', header: t('details.returnNumber'), render: (r) => <span className="font-medium text-brand-700">{r.return_number}</span> },
    { key: 'department', header: t('details.department'), render: (r) => r.department.name },
    { key: 'requester', header: t('details.requestedBy'), render: (r) => r.requested_by.full_name },
    { key: 'status', header: t('details.status'), render: (r) => <StatusBadge status={r.status} /> },
    { key: 'lines', header: t('details.lines'), render: (r) => formatQty(r.item_count) },
    { key: 'progress', header: t('details.acceptedRequested'), render: (r) => `${formatQty(r.total_accepted)} / ${formatQty(r.total_requested)}` },
    { key: 'updated', header: t('details.updated'), render: (r) => formatDateTime(r.updated_at) },
    { key: 'created', header: t('details.created'), render: (r) => formatDateTime(r.created_at) },
  ]

  const selectTab = (next: Tab) => {
    setTab(next)
    setStatus('')
    setSearch('')
    setPage(1)
  }

  return (
    <div>
      <PageHeader
        title={t('pages.monitoring')}
        description={t('pages.monitoringDescription')}
        actions={
          <AutoRefreshIndicator
            updatedAt={active.dataUpdatedAt}
            intervalMs={MONITORING_POLL_MS}
            isFetching={active.isFetching}
            onRefresh={() => active.refetch()}
          />
        }
      />
      <p className="mb-4 text-sm text-slate-500">{t('monitoring.readOnlyNotice')}</p>
      <div className="mb-4 flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => selectTab('requests')}
          className={cn('rounded-full px-4 py-1.5 text-sm font-medium transition-colors', tab === 'requests' ? 'bg-brand-600 text-white' : 'bg-white text-slate-600 hover:bg-slate-200')}
        >
          {t('monitoring.tabRequests')}
        </button>
        <button
          type="button"
          onClick={() => selectTab('returns')}
          className={cn('rounded-full px-4 py-1.5 text-sm font-medium transition-colors', tab === 'returns' ? 'bg-brand-600 text-white' : 'bg-white text-slate-600 hover:bg-slate-200')}
        >
          {t('monitoring.tabReturns')}
        </button>
      </div>
      <Card>
        <FilterBar>
          <SearchInput value={search} onChange={(value) => { setSearch(value); setPage(1) }} placeholder={t('monitoring.searchPlaceholder')} />
          <Select className="h-11 w-full sm:h-9 sm:w-52" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
            <option value="">{t('common.allStatuses')}</option>
            {(tab === 'requests' ? requestStatuses : returnStatuses).map((value) => (
              <option key={value} value={value}>{t(`status.${value}`)}</option>
            ))}
          </Select>
          <Select className="h-11 w-full sm:h-9 sm:w-56" value={departmentId} onChange={(e) => { setDepartmentId(e.target.value); setPage(1) }}>
            <option value="">{t('monitoring.allDepartments')}</option>
            {departments.data?.data.map((department) => (
              <option key={department.id} value={department.id}>{department.name}</option>
            ))}
          </Select>
        </FilterBar>

        {tab === 'requests' ? (
          <DataTable
            columns={requestColumns}
            rows={requests.data?.items ?? []}
            loading={requests.isLoading}
            error={requests.error}
            onRetry={() => requests.refetch()}
            onRowClick={(row) => {
              if (canOpenDetail(row.department.id)) navigate(`/requests/${row.id}`)
            }}
            getRowKey={(row) => row.id}
            renderExpanded={(row) => (
              <dl className="grid gap-2 text-sm sm:grid-cols-2">
                <div><dt className="text-xs uppercase text-slate-500">{t('details.department')}</dt><dd>{row.department.name}</dd></div>
                <div><dt className="text-xs uppercase text-slate-500">{t('details.requestedBy')}</dt><dd>{row.requested_by.full_name}</dd></div>
                <div><dt className="text-xs uppercase text-slate-500">{t('details.lines')}</dt><dd>{formatQty(row.item_count)}</dd></div>
                <div><dt className="text-xs uppercase text-slate-500">{t('details.issuedRequested')}</dt><dd>{formatQty(row.total_issued)} / {formatQty(row.total_requested)}</dd></div>
                <div><dt className="text-xs uppercase text-slate-500">{t('details.created')}</dt><dd>{formatDateTime(row.created_at)}</dd></div>
                <div><dt className="text-xs uppercase text-slate-500">{t('details.updated')}</dt><dd>{formatDateTime(row.updated_at)}</dd></div>
              </dl>
            )}
          />
        ) : (
          <DataTable
            columns={returnColumns}
            rows={returns.data?.items ?? []}
            loading={returns.isLoading}
            error={returns.error}
            onRetry={() => returns.refetch()}
            onRowClick={(row) => {
              if (canOpenDetail(row.department.id)) navigate(`/returns/${row.id}`)
            }}
            getRowKey={(row) => row.id}
            renderExpanded={(row) => (
              <dl className="grid gap-2 text-sm sm:grid-cols-2">
                <div><dt className="text-xs uppercase text-slate-500">{t('details.department')}</dt><dd>{row.department.name}</dd></div>
                <div><dt className="text-xs uppercase text-slate-500">{t('details.requestedBy')}</dt><dd>{row.requested_by.full_name}</dd></div>
                <div><dt className="text-xs uppercase text-slate-500">{t('details.lines')}</dt><dd>{formatQty(row.item_count)}</dd></div>
                <div><dt className="text-xs uppercase text-slate-500">{t('details.acceptedRequested')}</dt><dd>{formatQty(row.total_accepted)} / {formatQty(row.total_requested)}</dd></div>
                <div><dt className="text-xs uppercase text-slate-500">{t('details.created')}</dt><dd>{formatDateTime(row.created_at)}</dd></div>
                <div><dt className="text-xs uppercase text-slate-500">{t('details.updated')}</dt><dd>{formatDateTime(row.updated_at)}</dd></div>
              </dl>
            )}
          />
        )}

        {tab === 'requests' && requests.data && (
          <Pagination meta={requests.data.meta} onPage={setPage} onPageSize={(size) => { setPageSize(size); setPage(1) }} />
        )}
        {tab === 'returns' && returns.data && (
          <Pagination meta={returns.data.meta} onPage={setPage} onPageSize={(size) => { setPageSize(size); setPage(1) }} />
        )}
      </Card>
    </div>
  )
}
