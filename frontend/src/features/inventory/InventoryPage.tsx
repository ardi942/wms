import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { inventoryApi, referenceApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Card, Select } from '../../components/ui'
import { DataTable, FilterBar, Pagination, PageHeader, SearchInput, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { formatCurrency, formatQty } from '../../lib/format'
import type { InventoryBalance } from '../../types'

export function InventoryPage() {
  const { user } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [lowStock, setLowStock] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const filters = { search, category_id: categoryId, low_stock: lowStock || undefined, page, page_size: pageSize }
  const balances = useQuery({ queryKey: ['inventory', filters], queryFn: () => inventoryApi.balances(filters) })
  const categories = useQuery({ queryKey: ['categories'], queryFn: () => referenceApi.categories() })

  const columns: Column<InventoryBalance>[] = [
    { key: 'sku', header: t('fields.sku'), render: (r) => <span className="font-medium">{r.sku}</span> },
    { key: 'name', header: t('details.product'), render: (r) => r.name, tooltip: (r) => r.name },
    { key: 'category', header: t('details.category'), render: (r) => r.category },
    { key: 'qty', header: t('details.quantity'), render: (r) => <span className={r.is_low_stock ? 'font-semibold text-amber-700' : ''}>{formatQty(r.quantity)} {r.unit}</span> },
    { key: 'min', header: t('details.minimum'), render: (r) => formatQty(r.min_stock) },
    ...(user?.role === 'SUPERVISOR' ? [{ key: 'value', header: t('details.stockValue'), render: (r: InventoryBalance) => formatCurrency(r.stock_value) }] : []),
  ]

  return (
    <div>
      <PageHeader title={t('pages.inventory')} description={t('pages.inventoryDescription')} />
      <Card>
        <FilterBar>
          <SearchInput value={search} onChange={(v) => { setSearch(v); setPage(1) }} placeholder={t('common.searchProduct')} />
          <Select className="h-9 w-48" value={categoryId} onChange={(e) => { setCategoryId(e.target.value); setPage(1) }}>
            <option value="">{t('common.allCategories')}</option>
            {categories.data?.data.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </Select>
          <label className="flex items-center gap-2 text-sm text-slate-600">
            <input type="checkbox" checked={lowStock} onChange={(e) => { setLowStock(e.target.checked); setPage(1) }} className="h-4 w-4 rounded border-slate-300" /> {t('filters.lowStockOnly')}
          </label>
        </FilterBar>
        <DataTable
          columns={columns}
          rows={balances.data?.items ?? []}
          loading={balances.isLoading}
          error={balances.error}
          onRetry={() => balances.refetch()}
          onRowClick={(r) => navigate(`/inventory/${r.product_id}`)}
          getRowKey={(r) => r.product_id}
          renderExpanded={(r) => (
            <dl className="grid gap-2 text-sm sm:grid-cols-2">
              <div><dt className="text-xs uppercase text-slate-500">{t('fields.sku')}</dt><dd>{r.sku}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.category')}</dt><dd>{r.category}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.quantity')}</dt><dd>{formatQty(r.quantity)} {r.unit}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.minimum')}</dt><dd>{formatQty(r.min_stock)}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.status')}</dt><dd>{r.is_low_stock ? t('common.lowStock') : t('common.ok')}</dd></div>
              {user?.role === 'SUPERVISOR' && (
                <div><dt className="text-xs uppercase text-slate-500">{t('details.stockValue')}</dt><dd>{formatCurrency(r.stock_value)}</dd></div>
              )}
            </dl>
          )}
        />
        {balances.data && <Pagination meta={balances.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
