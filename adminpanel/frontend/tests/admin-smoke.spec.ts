import { expect, test, type Page } from '@playwright/test';

async function goToSection(page: Page, section: RegExp | string) {
  await page.locator('aside').getByRole('button', { name: section }).click();
}

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('header').getByRole('heading', { level: 2, name: 'Overview' })).toBeVisible();
});

test('renders overview dashboard widgets', async ({ page }) => {
  await expect(page.getByText('Growth Overview')).toBeVisible();
  await expect(page.getByText('Recent system alerts')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Retention Rate' })).toBeVisible();
});

test('opens users section and shows user records', async ({ page }) => {
  await goToSection(page, 'Users');

  await expect(page.locator('header').getByRole('heading', { level: 2, name: 'Users' })).toBeVisible();
  await expect(page.getByText('Emma Wilson')).toBeVisible();
  await expect(page.getByText('@jamesc')).toBeVisible();
});

test('opens moderation section and shows queue actions', async ({ page }) => {
  await goToSection(page, /^Moderation\b/);

  await expect(page.locator('header').getByRole('heading', { level: 2, name: 'Moderation' })).toBeVisible();
  await expect(page.getByText('Review and moderate reported content')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Approve' })).toBeVisible();
});

test('opens roles section and shows role management controls', async ({ page }) => {
  await goToSection(page, 'Roles & Access');

  await expect(page.locator('header').getByRole('heading', { level: 2, name: 'Roles & Access' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Create Role' })).toBeVisible();
  await expect(page.getByPlaceholder('Search roles...')).toBeVisible();
});

test('opens settings and switches tabs', async ({ page }) => {
  await goToSection(page, 'Settings');

  await expect(page.locator('header').getByRole('heading', { level: 2, name: 'Settings' })).toBeVisible();
  await page.getByRole('button', { name: 'Notifications', exact: true }).click();
  await expect(page.getByText('Notification Channels')).toBeVisible();

  await page.getByRole('button', { name: 'Integrations' }).click();
  await expect(page.getByPlaceholder('https://your-app.com/webhook')).toBeVisible();
});
