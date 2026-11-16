import { expect, test, type Browser, type Page } from '@playwright/test'
import {
  accounts,
  apiLogin,
  apiMe,
  appPath,
  createProduct,
  deleteProductImage,
  deleteUserImage,
  getImage,
  imageStatus,
  loginUI,
  unique,
  uploadProductImage,
} from './helpers'
import { invalidTextFile, oversizedPng, tinyJpeg, tinyPng } from './imageFixtures'

async function pageFor(browser: Browser, email: string, password: string): Promise<Page> {
  const context = await browser.newContext()
  const page = await context.newPage()
  await loginUI(page, email, password)
  return page
}

test('E2E-11 admin uploads, replaces, and removes a product image through the UI', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const product = await createProduct(request, adminToken, unique('E2EIMG'))
  const page = await pageFor(browser, accounts.admin.email, accounts.admin.password)
  try {
    await page.goto(appPath(`/products/${product.id}/edit`))
    await expect(page.getByRole('button', { name: 'Save changes' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Choose image' })).toBeVisible()

    await page.locator('input[type="file"]').setInputFiles(tinyPng)
    await expect(page.getByRole('button', { name: 'Replace image' })).toBeVisible()
    await page.getByRole('button', { name: 'Save changes' }).click()
    await page.waitForURL(new RegExp(`/products/${product.id}$`))
    await expect(page.getByRole('img', { name: 'Product image' })).toBeVisible()

    let stored = await getImage(request, adminToken, `/products/${product.id}/image`)
    expect(stored.contentType).toContain('image/png')
    expect(stored.body.equals(tinyPng.buffer)).toBe(true)

    await page.goto(appPath(`/products/${product.id}/edit`))
    await expect(page.getByRole('button', { name: 'Replace image' })).toBeVisible()
    await page.locator('input[type="file"]').setInputFiles(tinyJpeg)
    await page.getByRole('button', { name: 'Save changes' }).click()
    await page.waitForURL(new RegExp(`/products/${product.id}$`))
    await expect(page.getByRole('img', { name: 'Product image' })).toBeVisible()

    stored = await getImage(request, adminToken, `/products/${product.id}/image`)
    expect(stored.contentType).toContain('image/jpeg')
    expect(stored.body.equals(tinyJpeg.buffer)).toBe(true)

    await page.goto(appPath(`/products/${product.id}/edit`))
    await page.getByRole('button', { name: 'Remove image' }).click()
    await page.getByRole('button', { name: 'Save changes' }).click()
    await page.waitForURL(new RegExp(`/products/${product.id}$`))
    await expect(page.getByRole('img', { name: 'Product image' })).toHaveCount(0)
    expect(await imageStatus(request, adminToken, `/products/${product.id}/image`)).toBe(404)
  } finally {
    await page.context().close()
  }
})

test('E2E-12 admin profile photo appears in the header', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const me = await apiMe(request, adminToken)
  const page = await pageFor(browser, accounts.admin.email, accounts.admin.password)
  try {
    await page.goto(appPath(`/users/${me.id}/edit`))
    await page.locator('input[type="file"]').setInputFiles(tinyPng)
    await expect(page.getByRole('img', { name: 'Profile photo' })).toBeVisible()
    await page.getByRole('button', { name: 'Save changes' }).click()
    await page.waitForURL('**/users')

    const header = page.locator('header')
    await expect(header.getByRole('img', { name: me.full_name })).toBeVisible()
    await page.reload()
    await expect(header.getByRole('img', { name: me.full_name })).toBeVisible()

    await page.goto(appPath(`/users/${me.id}/edit`))
    await page.getByRole('button', { name: 'Remove image' }).click()
    await page.getByRole('button', { name: 'Save changes' }).click()
    await page.waitForURL('**/users')
    await expect(header.getByRole('img', { name: me.full_name })).toHaveCount(0)
  } finally {
    await deleteUserImage(request, adminToken, me.id).catch(() => undefined)
    await page.context().close()
  }
})

test('E2E-13 image controls respect supervisor and department roles', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const product = await createProduct(request, adminToken, unique('E2EIMG'))
  await uploadProductImage(request, adminToken, product.id, tinyPng)
  const supervisor = await pageFor(browser, accounts.supervisor.email, accounts.supervisor.password)
  const department = await pageFor(browser, accounts.staff.email, accounts.staff.password)
  try {
    await supervisor.goto(appPath(`/products/${product.id}/edit`))
    await expect(supervisor.getByRole('img', { name: 'Product image' })).toBeVisible()
    await expect(supervisor.locator('input[type="file"]')).toHaveCount(0)
    await expect(supervisor.getByRole('button', { name: 'Replace image' })).toHaveCount(0)
    await expect(supervisor.getByRole('button', { name: 'Remove image' })).toHaveCount(0)

    await department.goto(appPath(`/products/${product.id}`))
    await expect(department.getByRole('img', { name: 'Product image' })).toBeVisible()
    await department.goto(appPath(`/products/${product.id}/edit`))
    await expect(department.getByText('Access denied')).toBeVisible()
  } finally {
    await deleteProductImage(request, adminToken, product.id).catch(() => undefined)
    await supervisor.context().close()
    await department.context().close()
  }
})

test('E2E-14 image picker rejects wrong file types and oversized images', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const product = await createProduct(request, adminToken, unique('E2EIMG'))
  const page = await pageFor(browser, accounts.admin.email, accounts.admin.password)
  try {
    await page.goto(appPath(`/products/${product.id}/edit`))
    const input = page.locator('input[type="file"]')

    await input.setInputFiles(invalidTextFile)
    await expect(page.getByRole('alert')).toHaveText('Choose a JPEG or PNG image.')

    await input.setInputFiles(oversizedPng)
    await expect(page.getByRole('alert')).toHaveText('Image must be 2 MiB or smaller.')

    await expect(page.getByRole('button', { name: 'Choose image' })).toBeVisible()
    await expect(page.getByRole('img', { name: 'Product image' })).toHaveCount(0)
  } finally {
    await page.context().close()
  }
})
