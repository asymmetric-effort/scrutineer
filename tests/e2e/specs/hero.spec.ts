import { test, expect } from '@playwright/test';

test.describe('Hero Section', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('heading displays project name', async ({ page }) => {
    await expect(page.locator('.hero h1')).toHaveText('scrutineer');
  });

  test('tagline describes the framework', async ({ page }) => {
    const tagline = page.locator('.tagline');
    await expect(tagline).toContainText('extensible test framework');
    await expect(tagline).toContainText('CLI programs');
    await expect(tagline).toContainText('REST APIs');
    await expect(tagline).toContainText('GraphQL');
    await expect(tagline).toContainText('gRPC');
    await expect(tagline).toContainText('Declarative YAML tests');
    await expect(tagline).toContainText('Zero third-party dependencies');
  });

  test('has exactly 3 badges', async ({ page }) => {
    await expect(page.locator('.badges .badge')).toHaveCount(3);
  });

  test('Go version badge is displayed', async ({ page }) => {
    await expect(page.locator('.badge-green')).toContainText(/Go 1\.26/);
  });

  test('MIT License badge is displayed', async ({ page }) => {
    await expect(page.locator('.badge-blue')).toHaveText('MIT License');
  });

  test('version badge is displayed', async ({ page }) => {
    await expect(page.locator('.badge-orange')).toContainText(/^v\d+\.\d+/);
  });

  test('Get Started button links to install section', async ({ page }) => {
    const btn = page.locator('.btn-primary');
    await expect(btn).toHaveText('Get Started');
    await expect(btn).toHaveAttribute('href', /#install/);
  });

  test('View Source button links to GitHub', async ({ page }) => {
    const btn = page.locator('.btn-secondary');
    await expect(btn).toHaveText('View Source');
    await expect(btn).toHaveAttribute('href', /github\.com\/asymmetric-effort\/scrutineer/);
  });
});
