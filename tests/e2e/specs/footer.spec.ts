import { test, expect } from '@playwright/test';

test.describe('Footer', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('footer element exists', async ({ page }) => {
    await expect(page.locator('footer')).toBeVisible();
  });

  test('footer has contentinfo role', async ({ page }) => {
    await expect(page.locator('footer')).toHaveAttribute('role', 'contentinfo');
  });

  test('footer has ARIA label', async ({ page }) => {
    await expect(page.locator('footer')).toHaveAttribute('aria-label', 'Site footer');
  });

  test('displays project name and version', async ({ page }) => {
    await expect(page.locator('footer')).toContainText(/Scrutineer v?\d+\.\d+/);
  });

  test('displays copyright with correct year range', async ({ page }) => {
    const year = new Date().getFullYear();
    await expect(page.locator('footer')).toContainText(`© 2022-${year}`);
  });

  test('displays company name', async ({ page }) => {
    await expect(page.locator('footer')).toContainText('Asymmetric Effort, LLC');
  });

  test('displays MIT License', async ({ page }) => {
    await expect(page.locator('footer')).toContainText('MIT License');
  });

  test('company name links to asymmetric-effort.com', async ({ page }) => {
    const link = page.locator('footer a[href*="asymmetric-effort.com"]').first();
    await expect(link).toHaveText('Asymmetric Effort, LLC');
  });

  test('GitHub Repository link points to correct repo', async ({ page }) => {
    const link = page.locator('footer a[href*="github.com/asymmetric-effort/scrutineer"]');
    await expect(link).toHaveText('GitHub Repository');
  });

  test('footer uses three-column layout', async ({ page }) => {
    const columns = page.locator('footer > div > div');
    await expect(columns).toHaveCount(3);
  });
});
