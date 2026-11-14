import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { productApi, referenceApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, Select } from '../../components/ui'
import { DataTable, FilterBar, Pagination, PageHeader, SearchInput, type Column } from '../../components/common'
import { useT } from '../../i18n/I18nProvider'
import { formatCurrency, formatQty } from '../../lib/format'
import type { Product } from '../../types'

export function ProductListPage() {
  const { user } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [active, setActive] = useState('')
  const [lowStock, setLowStock] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const filters = { search, category_id: categoryId, is_active: active, low_stock: lowStock || undefined, page, page_size: pageSize }
  const products = useQuery({ queryKey: ['products', filters], queryFn: () => productApi.list(filters) })
  const categories = useQuery({ queryKey: ['categories'], queryFn: () => referenceApi.categories() })

  const isSupervisor = user?.role === 'SUPERVISOR'
  const columns: Column<Product>[] = [
    { key: 'sku', header: t('fields.sku'), render: (p) => <span className="font-medium text-slate-900">{p.sku}</span> },
    { key: 'name', header: t('details.product'), render: (p) => p.name, tooltip: (p) => p.description ?? p.name },
    { key: 'category', header: t('details.category'), render: (p) => p.category.name },
    { key: 'stock', header: t('details.stock'), render: (p) => <span className={p.is_low_stock ? 'font-medium text-amber-700' : ''}>{formatQty(p.current_stock)} {p.unit.code}</span> },
    { key: 'min', header: t('details.minimum'), render: (p) => formatQty(p.min_stock) },
    ...(isSupervisor ? [{ key: 'cost', header: t('details.unitCost'), render: (p: Product) => formatCurrency(p.unit_cost) }] : []),
    { key: 'active', header: t('details.status'), render: (p) => p.is_active ? <span className="text-emerald-600">{t('common.active')}</span> : <span className="text-slate-400">{t('common.inactive')}</span> },
  ]

  return (
    <div>
      <PageHeader
        title={t('pages.products')}
        description={t('pages.productsDescription')}
        actions={user?.role === 'WAREHOUSE_ADMIN' ? <Link to="/products/new"><Button><Plus className="h-4 w-4" /> {t('actions.newProduct')}</Button></Link> : undefined}
      />
      <Card>
        <FilterBar>
          <SearchInput value={search} onChange={(v) => { setSearch(v); setPage(1) }} placeholder={t('common.searchSkuOrName')} />
          <Select className="h-11 w-full sm:h-9 sm:w-48" value={categoryId} onChange={(e) => { setCategoryId(e.target.value); setPage(1) }}>
            <option value="">{t('common.allCategories')}</option>
            {categories.data?.data.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </Select>
          <Select className="h-11 w-full sm:h-9 sm:w-36" value={active} onChange={(e) => { setActive(e.target.value); setPage(1) }}>
            <option value="">{t('common.anyStatus')}</option>
            <option value="true">{t('common.active')}</option>
            <option value="false">{t('common.inactive')}</option>
          </Select>
          <label className="flex min-h-11 items-center gap-2 text-sm text-slate-600">
            <input type="checkbox" checked={lowStock} onChange={(e) => { setLowStock(e.target.checked); setPage(1) }} className="h-4 w-4 rounded border-slate-300" />
            {t('filters.lowStockOnly')}
          </label>
        </FilterBar>
        <DataTable
          columns={columns}
          rows={products.data?.items ?? []}
          loading={products.isLoading}
          error={products.error}
          onRetry={() => products.refetch()}
          onRowClick={(p) => navigate(`/products/${p.id}`)}
          getRowKey={(p) => p.id}
          renderExpanded={(p) => (
            <dl className="grid gap-2 text-sm sm:grid-cols-2">
              <div><dt className="text-xs uppercase text-slate-500">{t('details.description')}</dt><dd>{p.description ?? '—'}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.category')}</dt><dd>{p.category.name}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.unit')}</dt><dd>{p.unit.name} ({p.unit.code})</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.stock')}</dt><dd>{formatQty(p.current_stock)} / {t('details.minimum').toLowerCase()} {formatQty(p.min_stock)}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">{t('details.status')}</dt><dd>{p.is_active ? t('common.active') : t('common.inactive')}</dd></div>
              {isSupervisor && (
                <div><dt className="text-xs uppercase text-slate-500">{t('details.unitCost')}</dt><dd>{formatCurrency(p.unit_cost)}</dd></div>
              )}
            </dl>
          )}
        />
        {products.data && <Pagination meta={products.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
