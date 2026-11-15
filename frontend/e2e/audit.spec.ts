import { expect, test, type Browser, type Page } from '@playwright/test'
import {
  accounts,
  apiLogin,
  appPath,
  createProduct,
  createUser,
  loginUI,
  setUserActive,
  unique,
  updateProductCost,
} from './helpers'

async function pageFor(browser: Browser, email: string, password: string): Promise<Page> {
  const context = await browser.newContext()
  const page = await context.newPage()
  await loginUI(page, email, password)
  return page
}

test('E2E-8 user status changes require a reason and appear in audit history', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const suffix = unique('AUDIT')
  const email = `${suffix.toLowerCase()}@wms.local`
  const fullName = `Audit User ${suffix}`
  const deactivateReason = `Playwright deactivation ${suffix}`
  const activateReason = `Playwright reactivation ${suffix}`
  const user = await createUser(request, adminToken, email, fullName)

  await setUserActive(request, adminToken, user.id, false, deactivateReason)

  const blocked = await browser.newContext()
  const blockedPage = await blocked.newPage()
  await blockedPage.goto(appPath('/login'))
  await blockedPage.getByLabel('Email', { exact: true }).fill(email)
  await blockedPage.getByLabel('Password', { exact: true }).fill('Staff123!')
  await blockedPage.getByRole('button', { name: 'Sign in' }).click()
  await expect(blockedPage.getByText('invalid email or password')).toBeVisible()
  await expect(blockedPage).toHaveURL(/\/login/)
  await blocked.close()

  const supervisor = await pageFor(browser, accounts.supervisor.email, accounts.supervisor.password)
  await expect(supervisor.locator('aside').getByRole('link', { name: 'Audit' })).toBeVisible()
  await expect(supervisor.locator('aside').getByRole('link', { name: 'Users' })).toHaveCount(0)
  await supervisor.goto(appPath('/audit'))
  const deactivationRow = supervisor.getByRole('row').filter({ hasText: deactivateReason })
  await expect(deactivationRow).toContainText(fullName)
  await expect(deactivationRow).toContainText('Account deactivated')

  const admin = await pageFor(browser, accounts.admin.email, accounts.admin.password)
  await admin.goto(appPath(`/users/${user.id}/edit`))
  await expect(admin.getByText('Account history')).toBeVisible()
  await expect(admin.getByRole('row').filter({ hasText: deactivateReason })).toBeVisible()

  await setUserActive(request, adminToken, user.id, true, activateReason)
  await supervisor.reload()
  const activationRow = supervisor.getByRole('row').filter({ hasText: activateReason })
  await expect(activationRow).toContainText(fullName)
  await expect(activationRow).toContainText('Account activated')

  const staff = await browser.newContext()
  const staffPage = await staff.newPage()
  await loginUI(staffPage, email, 'Staff123!')
  await expect(staffPage).toHaveURL(/\/dashboard$/)
  await staff.close()
})

test('E2E-9 product cost history is redacted for admins and visible to supervisors', async ({ browser, request }) => {
  const adminToken = await apiLogin(request, accounts.admin.email, accounts.admin.password)
  const supervisorToken = await apiLogin(request, accounts.supervisor.email, accounts.supervisor.password)
  const sku = unique('E2E')
  const product = await createProduct(request, adminToken, sku)
  await updateProductCost(request, supervisorToken, product.id, 37.5)

  const supervisor = await pageFor(browser, accounts.supervisor.email, accounts.supervisor.password)
  await supervisor.goto(appPath(`/products/${product.id}`))
  const supervisorRow = supervisor.getByRole('row').filter({ hasText: 'Unit cost updated' })
  await expect(supervisorRow).toBeVisible()
  await supervisorRow.getByRole('button', { name: 'Expand' }).click()
  const supervisorDetail = supervisor.locator('tr.bg-slate-50').filter({ hasText: 'Unit cost' })
  await expect(supervisorDetail).toContainText('10')
  await expect(supervisorDetail).toContainText('37.5')

  const admin = await pageFor(browser, accounts.admin.email, accounts.admin.password)
  await admin.goto(appPath(`/products/${product.id}`))
  const adminRow = admin.getByRole('row').filter({ hasText: 'Unit cost updated' })
  await expect(adminRow).toContainText('Unit cost changed (value hidden)')
  await adminRow.getByRole('button', { name: 'Expand' }).click()
  const adminDetail = admin.locator('tr.bg-slate-50').filter({ hasText: 'Unit cost' })
  await expect(adminDetail).toContainText('Hidden')
  await expect(admin.getByText('37.5')).toHaveCount(0)
})
