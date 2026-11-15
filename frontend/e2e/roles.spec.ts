import { expect, test } from '@playwright/test'
import { accounts, appPath, loginUI } from './helpers'

const allItems = ['Dashboard', 'Products', 'Inventory', 'Receiving', 'Stock Requests', 'Returns', 'Adjustments', 'Reports', 'Audit', 'Users', 'Notifications']

const cases = [
  {
    name: 'supervisor',
    account: accounts.supervisor,
    visible: ['Dashboard', 'Products', 'Inventory', 'Stock Requests', 'Returns', 'Reports', 'Audit', 'Notifications'],
    hidden: ['Receiving', 'Adjustments', 'Users'],
    forbiddenPath: '/adjustments',
  },
  {
    name: 'warehouse admin',
    account: accounts.admin,
    visible: allItems,
    hidden: [],
    forbiddenPath: null,
  },
  {
    name: 'department user',
    account: accounts.staff,
    visible: ['Dashboard', 'Products', 'Inventory', 'Stock Requests', 'Returns', 'Reports', 'Notifications'],
    hidden: ['Receiving', 'Adjustments', 'Audit', 'Users'],
    forbiddenPath: '/users',
  },
]

for (const role of cases) {
  test(`E2E-7 ${role.name} sees the correct navigation and is blocked from forbidden pages`, async ({ page }) => {
    await loginUI(page, role.account.email, role.account.password)
    const nav = page.locator('aside')
    for (const item of role.visible) {
      await expect(nav.getByRole('link', { name: item })).toBeVisible()
    }
    for (const item of role.hidden) {
      await expect(nav.getByRole('link', { name: item })).toHaveCount(0)
    }
    if (role.forbiddenPath) {
      await page.goto(appPath(role.forbiddenPath))
      await expect(page.getByText('Access denied')).toBeVisible()
    }
  })
}
