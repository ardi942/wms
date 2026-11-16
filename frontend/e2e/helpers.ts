import { Buffer } from 'node:buffer'
import { expect, type APIRequestContext, type Page } from '@playwright/test'
import type { ImageFile } from './imageFixtures'

const API = process.env.E2E_API_URL || 'http://localhost:8080/api/v1'

export const accounts = {
  supervisor: { email: 'supervisor@wms.local', password: 'Supervisor123!' },
  admin: { email: 'warehouse@wms.local', password: 'Warehouse123!' },
  staff: { email: 'staff@wms.local', password: 'Staff123!' },
  staffFin: { email: 'staff.fin@wms.local', password: 'Staff123!' },
}

export function appPath(path: string): string {
  const base = new URL(process.env.E2E_BASE_URL || 'http://localhost:3000')
  const prefix = base.pathname.replace(/\/$/, '')
  return `${base.origin}${prefix}${path.startsWith('/') ? path : `/${path}`}`
}

interface Envelope<T> { data: T }
interface Ref { id: string }
interface Product { id: string; sku: string }
interface User { id: string; email: string; full_name: string; is_active: boolean }
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
  await page.goto(appPath('/login'))
  await page.getByLabel('Email', { exact: true }).fill(email)
  await page.getByLabel('Password', { exact: true }).fill(password)
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

export async function createUser(
  request: APIRequestContext,
  token: string,
  email: string,
  fullName: string,
): Promise<User> {
  const departments = await authed<Envelope<Ref[]>>(request, token, 'GET', '/departments')
  const body = await authed<Envelope<User>>(request, token, 'POST', '/users', {
    email,
    full_name: fullName,
    password: 'Staff123!',
    department_id: departments.data[0].id,
  })
  return body.data
}

export async function setUserActive(
  request: APIRequestContext,
  token: string,
  userId: string,
  active: boolean,
  reason: string,
): Promise<void> {
  await authed(request, token, 'POST', `/users/${userId}/${active ? 'activate' : 'deactivate'}`, { reason })
}

export async function updateProductCost(
  request: APIRequestContext,
  token: string,
  productId: string,
  unitCost: number,
): Promise<void> {
  await authed(request, token, 'PATCH', `/products/${productId}`, { unit_cost: unitCost })
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

interface Me {
  id: string
  full_name: string
}

export async function apiMe(request: APIRequestContext, token: string): Promise<Me> {
  const res = await request.get(`${API}/auth/me`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok()) throw new Error(`GET /auth/me failed: ${res.status()} ${await res.text()}`)
  const body = (await res.json()) as Envelope<{ user: Me }>
  return body.data.user
}

async function uploadImage(
  request: APIRequestContext,
  token: string,
  path: string,
  image: ImageFile,
): Promise<void> {
  const res = await request.post(`${API}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
    multipart: { image },
  })
  if (!res.ok()) throw new Error(`POST ${path} failed: ${res.status()} ${await res.text()}`)
}

async function deleteImage(request: APIRequestContext, token: string, path: string): Promise<void> {
  const res = await request.delete(`${API}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok()) throw new Error(`DELETE ${path} failed: ${res.status()} ${await res.text()}`)
}

export function uploadProductImage(
  request: APIRequestContext,
  token: string,
  productId: string,
  image: ImageFile,
): Promise<void> {
  return uploadImage(request, token, `/products/${productId}/image`, image)
}

export function deleteProductImage(
  request: APIRequestContext,
  token: string,
  productId: string,
): Promise<void> {
  return deleteImage(request, token, `/products/${productId}/image`)
}

export function uploadUserImage(
  request: APIRequestContext,
  token: string,
  userId: string,
  image: ImageFile,
): Promise<void> {
  return uploadImage(request, token, `/users/${userId}/image`, image)
}

export function deleteUserImage(
  request: APIRequestContext,
  token: string,
  userId: string,
): Promise<void> {
  return deleteImage(request, token, `/users/${userId}/image`)
}

export async function getImage(
  request: APIRequestContext,
  token: string,
  path: string,
): Promise<{ contentType: string; body: Buffer }> {
  const res = await request.get(`${API}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok()) throw new Error(`GET ${path} failed: ${res.status()} ${await res.text()}`)
  return {
    contentType: res.headers()['content-type'] ?? '',
    body: Buffer.from(await res.body()),
  }
}

export async function imageStatus(
  request: APIRequestContext,
  token: string,
  path: string,
): Promise<number> {
  const res = await request.get(`${API}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  return res.status()
}
