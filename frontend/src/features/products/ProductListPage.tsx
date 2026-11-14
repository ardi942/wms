import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { productApi, referenceApi } from '../../api/endpoints'
import { useAuth } from '../../auth/AuthProvider'
import { Button, Card, Select } from '../../components/ui'
import { DataTable, FilterBar, Pagination, PageHeader, SearchInput, type Column } from '../../components/common'
import { formatCurrency, formatQty } from '../../lib/format'
import type { Product } from '../../types'

export function ProductListPage() {
  const { user } = useAuth()
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
    { key: 'sku', header: 'SKU', render: (p) => <span className="font-medium text-slate-900">{p.sku}</span> },
    { key: 'name', header: 'Product', render: (p) => p.name, tooltip: (p) => p.description ?? p.name },
    { key: 'category', header: 'Category', render: (p) => p.category.name },
    { key: 'stock', header: 'Stock', render: (p) => <span className={p.is_low_stock ? 'font-medium text-amber-700' : ''}>{formatQty(p.current_stock)} {p.unit.code}</span> },
    { key: 'min', header: 'Min', render: (p) => formatQty(p.min_stock) },
    ...(isSupervisor ? [{ key: 'cost', header: 'Unit cost', render: (p: Product) => formatCurrency(p.unit_cost) }] : []),
    { key: 'active', header: 'Status', render: (p) => p.is_active ? <span className="text-emerald-600">Active</span> : <span className="text-slate-400">Inactive</span> },
  ]

  return (
    <div>
      <PageHeader title="Products" description="Catalogue, pricing and current stock levels." actions={user?.role === 'WAREHOUSE_ADMIN' ? <Link to="/products/new"><Button><Plus className="h-4 w-4" /> New product</Button></Link> : undefined} />
      <Card>
        <FilterBar>
          <SearchInput value={search} onChange={(v) => { setSearch(v); setPage(1) }} placeholder="Search SKU or name…" />
          <Select className="h-9 w-48" value={categoryId} onChange={(e) => { setCategoryId(e.target.value); setPage(1) }}>
            <option value="">All categories</option>
            {categories.data?.data.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </Select>
          <Select className="h-9 w-36" value={active} onChange={(e) => { setActive(e.target.value); setPage(1) }}>
            <option value="">Any status</option>
            <option value="true">Active</option>
            <option value="false">Inactive</option>
          </Select>
          <label className="flex items-center gap-2 text-sm text-slate-600">
            <input type="checkbox" checked={lowStock} onChange={(e) => { setLowStock(e.target.checked); setPage(1) }} className="h-4 w-4 rounded border-slate-300" />
            Low stock only
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
              <div><dt className="text-xs uppercase text-slate-500">Description</dt><dd>{p.description ?? '—'}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">Category</dt><dd>{p.category.name}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">Unit</dt><dd>{p.unit.name} ({p.unit.code})</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">Stock</dt><dd>{formatQty(p.current_stock)} / min {formatQty(p.min_stock)}</dd></div>
              <div><dt className="text-xs uppercase text-slate-500">Status</dt><dd>{p.is_active ? 'Active' : 'Inactive'}</dd></div>
              {isSupervisor && (
                <div><dt className="text-xs uppercase text-slate-500">Unit cost</dt><dd>{formatCurrency(p.unit_cost)}</dd></div>
              )}
            </dl>
          )}
        />
        {products.data && <Pagination meta={products.data.meta} onPage={setPage} onPageSize={(s) => { setPageSize(s); setPage(1) }} />}
      </Card>
    </div>
  )
}
