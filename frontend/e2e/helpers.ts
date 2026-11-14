import { expect, type APIRequestContext, type Page } from '@playwright/test'

const API = process.env.E2E_API_URL || 'http://localhost:8080/api/v1'

export const accounts = {
  supervisor: { email: 'supervisor@wms.local', password: 'Supervisor123!' },
  admin: { email: 'warehouse@wms.local', password: 'Warehouse123!' },
  staff: { email: 'staff@wms.local', password: 'Staff123!' },
  staffFin: { email: 'staff.fin@wms.local', password: 'Staff123!' },
}

interface Envelope<T> { data: T }
interface Ref { id: string }
interface Product { id: string; sku: string }
interface StockRequest { id: string; request_number: string }
interface StockReturn { id: string; return_number: string }

let counter = 0
export function unique(prefix: string): string {
  counter += 1
  return `${prefix}${Date.now().toString(36).toUpperCase()}${counter}`.slice(0, 32)
}

export function idempotencyKey(): string {
  counter += 1
  return `e2e-${Date.now()}-${counter}`
}

export async function loginUI(page: Page, email: string, password: string) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await page.waitForURL('**/dashboard')
  await expect(page.getByRole('heading', { name: /^Welcome,/ })).toBeVisible()
}

async function authed<T>(
  request: APIRequestContext,
  token: string,
  method: 'GET' | 'POST' | 'PATCH',
  path: string,
  data?: unknown,
): Promise<T> {
  const res = await request.fetch(`${API}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(method === 'GET' ? {} : { 'Idempotency-Key': idempotencyKey() }),
    },
    data,
  })
  if (!res.ok()) {
    throw new Error(`${method} ${path} failed: ${res.status()} ${await res.text()}`)
  }
  return (await res.json()) as T
}

export async function apiLogin(request: APIRequestContext, email: string, password: string): Promise<string> {
  const res = await request.post(`${API}/auth/login`, { data: { email, password } })
  if (!res.ok()) throw new Error(`login failed: ${res.status()} ${await res.text()}`)
  const body = (await res.json()) as Envelope<{ token: string }>
  return body.data.token
}

export async function createProduct(
  request: APIRequestContext,
  token: string,
  sku: string,
  name = `E2E product ${sku}`,
): Promise<Product> {
  const [categories, units] = await Promise.all([
    authed<Envelope<Ref[]>>(request, token, 'GET', '/categories'),
    authed<Envelope<Ref[]>>(request, token, 'GET', '/units'),
  ])
  const body = await authed<Envelope<Product>>(request, token, 'POST', '/products', {
    sku,
    name,
    description: 'Created by Playwright',
    category_id: categories.data[0].id,
    unit_id: units.data[0].id,
    min_stock: 0,
    unit_cost: 10,
  })
  return body.data
}

export async function receiveStock(
  request: APIRequestContext,
  token: string,
  productId: string,
  quantity: number,
): Promise<void> {
  await authed(request, token, 'POST', '/inventory/receipts', {
    product_id: productId,
    quantity,
    supplier: 'Playwright Supplies',
    reference_number: unique('PO-'),
    notes: null,
  })
}

export async function createRequest(
  request: APIRequestContext,
  token: string,
  productId: string,
  quantity: number,
): Promise<StockRequest> {
  const body = await authed<Envelope<StockRequest>>(request, token, 'POST', '/stock-requests', {
    purpose: 'Playwright request',
    items: [{ product_id: productId, requested_qty: quantity }],
  })
  return body.data
}

export async function fulfillRequest(
  request: APIRequestContext,
  token: string,
  requestId: string,
  itemId: string,
  quantity: number,
): Promise<void> {
  await authed(request, token, 'POST', `/stock-requests/${requestId}/fulfill`, {
    items: [{ item_id: itemId, issued_quantity: quantity }],
  })
}

export async function createReturn(
  request: APIRequestContext,
  token: string,
  productId: string,
  quantity: number,
): Promise<StockReturn> {
  const body = await authed<Envelope<StockReturn>>(request, token, 'POST', '/stock-returns', {
    reason: 'Playwright return',
    items: [{ product_id: productId, requested_qty: quantity }],
  })
  return body.data
}

export async function requestItemId(
  request: APIRequestContext,
  token: string,
  requestId: string,
): Promise<string> {
  const body = await authed<Envelope<{ items: { id: string }[] }>>(
    request,
    token,
    'GET',
    `/stock-requests/${requestId}`,
  )
  return body.data.items[0].id
}

export async function pickProduct(page: Page, sku: string) {
  const input = page.getByPlaceholder('Search by SKU or product name…')
  await input.fill(sku)
  await page.getByRole('button', { name: new RegExp(sku) }).click()
}

export async function getBalance(
  request: APIRequestContext,
  token: string,
  productId: string,
): Promise<number> {
  const body = await authed<Envelope<{ quantity: number }>>(
    request,
    token,
    'GET',
    `/inventory/${productId}`,
  )
  return body.data.quantity
}
