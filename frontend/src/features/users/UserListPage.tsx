import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { userApi } from '../../api/endpoints'
import { Button, Card, Select } from '../../components/ui'
import { DataTable, FilterBar, Pagination, PageHeader, SearchInput, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { formatDateTime } from '../../lib/format'
import type { User } from '../../types'

export function UserListPage() {
  const { t } = useT()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [active, setActive] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const filters = { search, is_active: active, page, page_size: pageSize }
  const users = useQuery({ queryKey: ['users', filters], queryFn: () => userApi.list(filters) })

  const columns: Column<User>[] = [
    { key: 'name', header: t('fields.fullName'), render: (u) => <span className="font-medium">{u.full_name}</span> },
    { key: 'email', header: t('fields.email'), render: (u) => u.email },
    { key: 'department', header: t('details.department'), render: (u) => u.department?.name ?? '—' },
    { key: 'role', header: t('details.role'), render: (u) => t(`role.${u.role}`) },
    { key: 'status', header: t('details.status'), render: (u) => u.is_active ? <span className="text-emerald-600">{t('common.active')}</span> : <span className="text-slate-400">{t('common.inactive')}</span> },
    { key: 'created', header: t('details.created'), render: (u) => formatDateTime(u.created_at) },
  ]

  return (
    <div>
      <PageHeader
        title={t('pages.users')}
        description={t('pages.usersDescription')}
        actions={<Link to="/users/new"><Button><Plus className="h-4 w-4" /> {t('actions.newUser')}</Button></Link>}
      />
      <Card>
        <FilterBar>
          <SearchInput value={search} onChange={(v) => { setSearch(v); setPage(1) }} placeholder={t('common.searchNameOrEmail')} />
          <Select className="h-9 w-36" value={active} onChange={(e) => { setActive(e.target.value); setPage(1) }}>
            <option value="">{t('common.anyStatus')}</option>
            <option value="true">{t('common.active')}</option>
            <option value="false">{t('common.inactive')}</option>
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
              <div><dt className="text-xs uppercase text-slate-500">{t('fields.email')}</dt><dd>{u.email}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.department')}</dt><dd>{u.department?.name ?? '—'}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.role')}</dt><dd>{t(`role.${u.role}`)}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.status')}</dt><dd>{u.is_active ? t('common.active') : t('common.inactive')}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.updated')}</dt><dd>{formatDateTime(u.updated_at)}</dd></div>
            </dl>
          )}
        />
        {users.data && <Pagination meta={users.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
