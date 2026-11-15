import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { History } from 'lucide-react'
import { auditApi } from '../../api/endpoints'
import { Card, Select } from '../../components/ui'
import {
  FilterBar,
  Pagination,
  PageHeader,
  RangeDatePicker,
  SearchInput,
} from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { auditActions } from '../../lib/constants'
import { AuditEventTable, auditActionLabel } from './AuditEventTable'

type EntityFilter = 'USER' | 'PRODUCT'

export function AuditHistoryPage() {
  const { t } = useT()
  const [entityType, setEntityType] = useState<EntityFilter>('USER')
  const [action, setAction] = useState('')
  const [actor, setActor] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const filters = {
    entity_type: entityType,
    action,
    actor,
    from,
    to,
    page,
    page_size: pageSize,
  }
  const events = useQuery({ queryKey: ['audit', filters], queryFn: () => auditApi.list(filters) })

  const resetPage = () => setPage(1)

  return (
    <div>
      <PageHeader title={t('pages.auditHistory')} description={t('pages.auditHistoryDescription')} />
      <Card>
        <FilterBar>
          <Select
            aria-label={t('audit.filterEntity')}
            className="h-11 w-full sm:h-9 sm:w-36"
            value={entityType}
            onChange={(event) => {
              setEntityType(event.target.value as EntityFilter)
              resetPage()
            }}
          >
            <option value="USER">{t('audit.entity.USER')}</option>
            <option value="PRODUCT">{t('audit.entity.PRODUCT')}</option>
          </Select>
          <Select
            aria-label={t('audit.filterAction')}
            className="h-11 w-full sm:h-9 sm:w-56"
            value={action}
            onChange={(event) => {
              setAction(event.target.value)
              resetPage()
            }}
          >
            <option value="">{t('common.allTypes')}</option>
            {auditActions.map((value) => (
              <option key={value} value={value}>{auditActionLabel(value, t)}</option>
            ))}
          </Select>
          <SearchInput
            value={actor}
            onChange={(value) => {
              setActor(value)
              resetPage()
            }}
            placeholder={t('audit.actorSearch')}
          />
          <RangeDatePicker
            from={from}
            to={to}
            onFrom={(value) => {
              setFrom(value)
              resetPage()
            }}
            onTo={(value) => {
              setTo(value)
              resetPage()
            }}
          />
        </FilterBar>
        <AuditEventTable
          events={events.data?.items ?? []}
          loading={events.isLoading}
          error={events.error}
          onRetry={() => events.refetch()}
        />
        {events.data && (
          <Pagination
            meta={events.data.meta}
            onPage={setPage}
            onPageSize={(size) => {
              setPageSize(size)
              setPage(1)
            }}
          />
        )}
      </Card>
      <div className="mt-3 flex items-center gap-2 text-xs text-slate-500">
        <History className="h-3.5 w-3.5" />
        {t('audit.readOnlyNotice')}
      </div>
    </div>
  )
}
