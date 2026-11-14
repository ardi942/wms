import { expect, test, type Browser, type Page } from '@playwright/test'
import {
  accounts,
  apiLogin,
  appPath,
  createProduct,
  createRequest,
  fulfillRequest,
  getBalance,
  loginUI,
  pickProduct,
  receiveStock,
  requestItemId,
  unique,
} from './helpers'

async function adminPage(browser: Browser): Promise<Page> {
  const context = await browser.newContext()
  const page = await context.newPage()
  await loginUI(page, accounts.admin.email, accounts.admin.password)
  return page
}

async function staffPage(browser: Browser): Promise<Page> {
  const context = await browser.newContext()
  const page = await context.newPage()
  await loginUI(page, accounts.staff.email, accounts.staff.password)
  return page
}

test('E2E-1 admin creates a product and receives stock through the UI', async ({ browser }) => {
  const sku = unique('E2E')
  const page = await adminPage(browser)

  await page.goto(appPath('/products/new'))
  await page.locator('#sku').fill(sku)
  await page.locator('#name').fill(`Playwright product ${sku}`)
  await page.locator('#category_id').selectOption({ index: 1 })
  await page.locator('#unit_id').selectOption({ index: 1 })
  await page.locator('#min_stock').fill('0')
  await page.getByRole('button', { name: 'Create product' }).click()
  await page.waitForURL('**/products')

  await page.goto(appPath('/receiving'))
  await pickProduct(page, sku)
  await page.locator('#quantity').fill('100')
  await page.locator('#supplier').fill('Playwright Supplies')
  await page.locator('#reference_number').fill(unique('PO-'))
  await page.getByRole('button', { name: 'Review receipt' }).click()
  await page.getByRole('button', { name: 'Receive', exact: true }).click()
  await expect(page.getByText('Stock received')).toBeVisible()

  await page.goto(appPath('/inventory'))
  await page.getByPlaceholder('Search product…').fill(sku)
  const row = page.getByRole('row', { name: new RegExp(sku) })
  await expect(row).toBeVisible()
  await expect(row).toContainText('100')
})

test('E2E-2 department request is fulfilled by the admin', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const sku = unique('E2E')
  const product = await createProduct(request, adminToken, sku)
  await receiveStock(request, adminToken, product.id, 100)

  const dept = await staffPage(browser)
  await dept.goto(appPath('/requests/new'))
  await dept.locator('#purpose').fill('Playwright fulfilment flow')
  await pickProduct(dept, sku)
  await dept.locator('input[type="number"]').first().fill('20')
  await dept.getByRole('button', { name: 'Submit request' }).click()
  await dept.waitForURL(/\/requests\/[0-9a-f-]+$/)
  const requestPath = new URL(dept.url()).pathname

  const admin = await adminPage(browser)
  await admin.goto(requestPath)
  await admin.getByRole('button', { name: 'Fulfil', exact: true }).click()
  await admin.getByRole('button', { name: 'Fulfil request', exact: true }).click()
  await expect(admin.locator('dl').getByText('Fulfilled', { exact: true })).toBeVisible()

  await dept.reload()
  await expect(dept.locator('dl').getByText('Fulfilled', { exact: true })).toBeVisible()
})

test('E2E-3 partial fulfilment updates status and drains stock', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const sku = unique('E2E')
  const product = await createProduct(request, adminToken, sku)
  await receiveStock(request, adminToken, product.id, 15)

  const dept = await staffPage(browser)
  await dept.goto(appPath('/requests/new'))
  await dept.locator('#purpose').fill('Playwright partial flow')
  await pickProduct(dept, sku)
  await dept.locator('input[type="number"]').first().fill('30')
  await dept.getByRole('button', { name: 'Submit request' }).click()
  await dept.waitForURL(/\/requests\/[0-9a-f-]+$/)
  const requestPath = new URL(dept.url()).pathname

  const admin = await adminPage(browser)
  await admin.goto(requestPath)
  await admin.getByRole('button', { name: 'Fulfil', exact: true }).click()
  await admin.locator('.fixed.inset-0 input[type="number"]').first().fill('15')
  await admin.getByRole('button', { name: 'Fulfil request', exact: true }).click()
  await expect(admin.locator('dl').getByText('Partially Fulfilled', { exact: true })).toBeVisible()
  await expect.poll(() => getBalance(request, adminToken, product.id)).toBe(0)
})

test('E2E-4 fulfilment beyond available stock is rejected', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const sku = unique('E2E')
  const product = await createProduct(request, adminToken, sku)
  await receiveStock(request, adminToken, product.id, 5)

  const dept = await staffPage(browser)
  await dept.goto(appPath('/requests/new'))
  await dept.locator('#purpose').fill('Playwright insufficient flow')
  await pickProduct(dept, sku)
  await dept.locator('input[type="number"]').first().fill('10')
  await dept.getByRole('button', { name: 'Submit request' }).click()
  await dept.waitForURL(/\/requests\/[0-9a-f-]+$/)
  const requestPath = new URL(dept.url()).pathname

  const admin = await adminPage(browser)
  await admin.goto(requestPath)
  await admin.getByRole('button', { name: 'Fulfil', exact: true }).click()
  await admin.locator('.fixed.inset-0 input[type="number"]').first().fill('10')
  await admin.getByRole('button', { name: 'Fulfil request', exact: true }).click()
  await expect(admin.getByText(/insufficient stock/i).first()).toBeVisible()
  await expect(admin.locator('dl').getByText('Pending', { exact: true })).toBeVisible()
  await expect.poll(() => getBalance(request, adminToken, product.id)).toBe(5)
})

test('E2E-5 department isolation hides another department request', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const staffToken = await apiLogin(request, accounts.staff.email, accounts.staff.password)
  const product = await createProduct(request, adminToken, unique('E2E'))
  const created = await createRequest(request, staffToken, product.id, 1)

  const fin = await browser.newContext()
  const page = await fin.newPage()
  await loginUI(page, accounts.staffFin.email, accounts.staffFin.password)
  await page.goto(appPath(`/requests/${created.id}`))
  await expect(page.getByText('stock request not found')).toBeVisible()
})

test('E2E-6 accepted return restores stock', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const staffToken = await apiLogin(request, accounts.staff.email, accounts.staff.password)
  const sku = unique('E2E')
  const product = await createProduct(request, adminToken, sku)
  await receiveStock(request, adminToken, product.id, 40)
  const created = await createRequest(request, staffToken, product.id, 20)
  const itemId = await requestItemId(request, staffToken, created.id)
  await fulfillRequest(request, adminToken, created.id, itemId, 20)
  expect(await getBalance(request, adminToken, product.id)).toBe(20)

  const dept = await staffPage(browser)
  await dept.goto(appPath('/returns/new'))
  await dept.locator('#reason').fill('Playwright return flow')
  await pickProduct(dept, sku)
  await dept.locator('input[type="number"]').first().fill('10')
  await dept.getByRole('button', { name: 'Submit return' }).click()
  await dept.waitForURL(/\/returns\/[0-9a-f-]+$/)
  const returnPath = new URL(dept.url()).pathname

  const admin = await adminPage(browser)
  await admin.goto(returnPath)
  await admin.getByRole('button', { name: 'Confirm return', exact: true }).click()
  await admin.locator('.fixed.inset-0 input[type="number"]').first().fill('10')
  await admin.getByRole('button', { name: 'Confirm return', exact: true }).last().click()
  await expect(admin.getByText('Return confirmed')).toBeVisible()
  await expect.poll(() => getBalance(request, adminToken, product.id)).toBe(30)
})
