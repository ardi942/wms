import { devices, expect, test } from '@playwright/test'
import { accounts, loginUI } from './helpers'

test.use({ ...devices['Pixel 7'] })

test('E2E-M1 mobile navigation, responsive lists and logout work', async ({ page }) => {
  await loginUI(page, accounts.admin.email, accounts.admin.password)

  const bottomNav = page.getByRole('navigation', { name: 'Main navigation' })
  await expect(bottomNav).toBeVisible()

  await bottomNav.getByRole('link', { name: 'Products' }).click()
  await expect(page).toHaveURL(/\/products$/)
  await expect(page.getByTestId('data-table-mobile')).toBeVisible()

  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
  expect(overflow).toBeLessThanOrEqual(1)

  await page.getByRole('button', { name: 'More' }).click()
  const drawer = page.getByRole('dialog', { name: 'Main navigation' })
  await expect(drawer).toBeVisible()
  await drawer.getByRole('link', { name: 'Inventory' }).click()
  await expect(page).toHaveURL(/\/inventory$/)
  await expect(page.getByTestId('data-table-mobile')).toBeVisible()

  await page.getByRole('button', { name: 'More' }).click()
  await page.getByRole('dialog', { name: 'Main navigation' }).getByRole('button', { name: 'Log out' }).click()
  await expect(page).toHaveURL(/\/login(\?|$)/)
})
