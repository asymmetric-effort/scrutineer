import { test, expect } from '@playwright/test';

test.describe('Cross-Platform Section', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('section heading exists', async ({ page }) => {
    const heading = page.locator('h2').filter({ hasText: 'Cross-Platform' });
    await expect(heading).toBeVisible();
  });

  test('subtitle says build once, test everywhere', async ({ page }) => {
    const subtitle = page.locator('.subtitle').filter({ hasText: 'Build once' });
    await expect(subtitle).toContainText('Build once, test everywhere');
  });

  test('has exactly 3 platform cards', async ({ page }) => {
    await expect(page.locator('.platform-card')).toHaveCount(3);
  });

  test('Linux card with AMD64/ARM64', async ({ page }) => {
    const card = page.locator('.platform-card').nth(0);
    await expect(card.locator('h3')).toHaveText('Linux');
    await expect(card.locator('p')).toContainText('AMD64');
    await expect(card.locator('p')).toContainText('ARM64');
  });

  test('macOS card with AMD64/ARM64', async ({ page }) => {
    const card = page.locator('.platform-card').nth(1);
    await expect(card.locator('h3')).toHaveText('macOS');
    await expect(card.locator('p')).toContainText('AMD64');
  });

  test('Windows card with AMD64/ARM64', async ({ page }) => {
    const card = page.locator('.platform-card').nth(2);
    await expect(card.locator('h3')).toHaveText('Windows');
    await expect(card.locator('p')).toContainText('AMD64');
  });
});
