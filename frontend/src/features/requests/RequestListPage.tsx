import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { requestApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, Select } from '../../components/ui'
import { DataTable, FilterBar, Pagination, PageHeader, SearchInput, StatusBadge, type Column } from '../../components/common'
import { formatDateTime, formatQty } from '../../lib/format'
import type { StockRequestList } from '../../types'

export function RequestListPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [status, setStatus] = useState(params.get('status') ?? '')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const filters = { status, search, page, page_size: pageSize }
  const requests = useQuery({ queryKey: ['requests', filters], queryFn: () => requestApi.list(filters) })

  const columns: Column<StockRequestList>[] = [
    { key: 'number', header: 'Request', render: (r) => <span className="font-medium text-brand-700">{r.request_number}</span> },
    { key: 'department', header: 'Department', render: (r) => r.department.name },
    { key: 'purpose', header: 'Purpose', render: (r) => <span className="line-clamp-1">{r.purpose}</span> },
    { key: 'items', header: 'Lines', render: (r) => formatQty(r.item_count) },
    { key: 'qty', header: 'Issued / requested', render: (r) => `${formatQty(r.total_issued)} / ${formatQty(r.total_requested)}` },
    { key: 'status', header: 'Status', render: (r) => <StatusBadge status={r.status} /> },
    { key: 'created', header: 'Created', render: (r) => formatDateTime(r.created_at) },
  ]

  return (
    <div>
      <PageHeader
        title="Stock Requests"
        description={user?.role === 'DEPARTMENT_USER' ? 'Requests submitted by your department.' : 'Review and fulfil requests from departments.'}
        actions={user?.role === 'DEPARTMENT_USER' ? <Link to="/requests/new"><Button><Plus className="h-4 w-4" /> New request</Button></Link> : undefined}
      />
      <Card>
        <FilterBar>
          <SearchInput value={search} onChange={(v) => { setSearch(v); setPage(1) }} placeholder="Search number or purpose…" />
          <Select className="h-9 w-48" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
            <option value="">All statuses</option>
            {['PENDING', 'PARTIALLY_FULFILLED', 'FULFILLED', 'REJECTED', 'CANCELLED'].map((s) => <option key={s} value={s}>{s.replaceAll('_', ' ')}</option>)}
          </Select>
        </FilterBar>
        <DataTable columns={columns} rows={requests.data?.items ?? []} loading={requests.isLoading} error={requests.error} onRetry={() => requests.refetch()} onRowClick={(r) => navigate(`/requests/${r.id}`)} />
        {requests.data && <Pagination meta={requests.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
