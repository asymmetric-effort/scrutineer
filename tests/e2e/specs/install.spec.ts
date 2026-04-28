import { test, expect } from '@playwright/test';

test.describe('Install Section', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('section has correct heading', async ({ page }) => {
    await expect(page.locator('#install h2')).toHaveText('Install');
  });

  test('subtitle says one command, no dependencies', async ({ page }) => {
    const subtitle = page.locator('#install .subtitle');
    await expect(subtitle).toContainText('One command');
    await expect(subtitle).toContainText('No dependencies');
  });

  test('go install command is displayed', async ({ page }) => {
    const installCode = page.locator('.install-block code');
    await expect(installCode).toContainText('go install');
    await expect(installCode).toContainText('github.com/asymmetric-effort/scrutineer');
    await expect(installCode).toContainText('@latest');
  });

  test('releases link points to GitHub releases', async ({ page }) => {
    const link = page.locator('#install a[href*="releases"]');
    await expect(link).toHaveText('Releases');
    await expect(link).toHaveAttribute('href', /github\.com\/asymmetric-effort\/scrutineer\/releases/);
  });

  test('platform availability mentioned', async ({ page }) => {
    const section = page.locator('#install');
    await expect(section).toContainText('Linux');
    await expect(section).toContainText('macOS');
    await expect(section).toContainText('Windows');
    await expect(section).toContainText('AMD64');
    await expect(section).toContainText('ARM64');
  });

  test('quick start code header is present', async ({ page }) => {
    const headers = page.locator('.code-header');
    const quickStart = headers.filter({ hasText: 'Quick start' });
    await expect(quickStart).toBeVisible();
  });

  test('quick start shows all key commands', async ({ page }) => {
    const quickStartPre = page.locator('#install pre').first();
    await expect(quickStartPre).toContainText('scrutineer browsers install');
    await expect(quickStartPre).toContainText('scrutineer run');
    await expect(quickStartPre).toContainText('--format json');
    await expect(quickStartPre).toContainText('scrutineer log-dump');
  });
});
