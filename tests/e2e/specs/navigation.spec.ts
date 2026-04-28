import { test, expect } from '@playwright/test';

test.describe('Navigation Bar', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('nav bar is visible', async ({ page }) => {
    await expect(page.locator('nav')).toBeVisible();
  });

  test('logo links to home', async ({ page }) => {
    const logo = page.locator('.logo');
    await expect(logo).toBeVisible();
    await expect(logo).toHaveAttribute('href', '/');
  });

  test('logo image is displayed', async ({ page }) => {
    const img = page.locator('.logo-icon');
    await expect(img).toBeVisible();
    await expect(img).toHaveAttribute('src', /logo\.png/);
  });

  test('logo text says scrutineer', async ({ page }) => {
    await expect(page.locator('.logo')).toContainText('scrutineer');
  });

  test('has exactly 4 nav links', async ({ page }) => {
    const links = page.locator('.nav-links a');
    await expect(links).toHaveCount(4);
  });

  test('Features link anchors to #features', async ({ page }) => {
    const link = page.locator('.nav-links a[href="#features"]');
    await expect(link).toHaveText('Features');
  });

  test('Protocols link anchors to #protocols', async ({ page }) => {
    const link = page.locator('.nav-links a[href="#protocols"]');
    await expect(link).toHaveText('Protocols');
  });

  test('Install link anchors to #install', async ({ page }) => {
    const link = page.locator('.nav-links a[href="#install"]');
    await expect(link).toHaveText('Install');
  });

  test('GitHub link points to correct repo', async ({ page }) => {
    const link = page.locator('.nav-links a[href*="github.com"]');
    await expect(link).toHaveText('GitHub');
    await expect(link).toHaveAttribute('href', /github\.com\/asymmetric-effort\/scrutineer/);
  });

  test('nav is fixed at top of page', async ({ page }) => {
    const position = await page.locator('nav').evaluate(el => getComputedStyle(el).position);
    expect(position).toBe('fixed');
  });
});
