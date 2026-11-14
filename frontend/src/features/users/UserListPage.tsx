import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { userApi } from '../../api/endpoints'
import { Button, Card, Select } from '../../components/ui'
import { DataTable, FilterBar, Pagination, PageHeader, SearchInput, type Column } from '../../components/common'
import { formatDateTime } from '../../lib/format'
import type { User } from '../../types'

export function UserListPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [active, setActive] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const filters = { search, is_active: active, page, page_size: pageSize }
  const users = useQuery({ queryKey: ['users', filters], queryFn: () => userApi.list(filters) })

  const columns: Column<User>[] = [
    { key: 'name', header: 'Name', render: (u) => <span className="font-medium">{u.full_name}</span> },
    { key: 'email', header: 'Email', render: (u) => u.email },
    { key: 'department', header: 'Department', render: (u) => u.department?.name ?? '—' },
    { key: 'role', header: 'Role', render: (u) => u.role.replaceAll('_', ' ') },
    { key: 'status', header: 'Status', render: (u) => u.is_active ? <span className="text-emerald-600">Active</span> : <span className="text-slate-400">Inactive</span> },
    { key: 'created', header: 'Created', render: (u) => formatDateTime(u.created_at) },
  ]

  return (
    <div>
      <PageHeader title="Users" description="Manage department users. Supervisors and warehouse admins are provisioned by seed." actions={<Link to="/users/new"><Button><Plus className="h-4 w-4" /> New user</Button></Link>} />
      <Card>
        <FilterBar>
          <SearchInput value={search} onChange={(v) => { setSearch(v); setPage(1) }} placeholder="Search name or email…" />
          <Select className="h-9 w-36" value={active} onChange={(e) => { setActive(e.target.value); setPage(1) }}>
            <option value="">Any status</option>
            <option value="true">Active</option>
            <option value="false">Inactive</option>
          </Select>
        </FilterBar>
        <DataTable
          columns={columns}
          rows={users.data?.items ?? []}
          loading={users.isLoading}
          error={users.error}
          onRetry={() => users.refetch()}
          onRowClick={(u) => navigate(`/users/${u.id}/edit`)}
          getRowKey={(u) => u.id}
          renderExpanded={(u) => (
            <dl className="grid gap-2 text-sm sm:grid-cols-2">
              <div><dt className="text-xs uppercase text-slate-500">Email</dt><dd>{u.email}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">Department</dt><dd>{u.department?.name ?? '—'}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">Role</dt><dd>{u.role.replaceAll('_', ' ')}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">Status</dt><dd>{u.is_active ? 'Active' : 'Inactive'}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">Updated</dt><dd>{formatDateTime(u.updated_at)}</dd></div>
            </dl>
          )}
        />
        {users.data && <Pagination meta={users.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
