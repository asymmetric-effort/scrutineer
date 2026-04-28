import { test, expect } from '@playwright/test';

const FEATURE_TITLES = [
  'Declarative YAML Tests',
  'Modular Connectors',
  'Browser Automation',
  'Load Testing',
  'Nanosecond Telemetry',
  'Zero Dependencies',
  'Coverage as a Feature',
  'Fuzz Testing',
  'Rich Assertions',
];

test.describe('Features Section', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('section has correct heading', async ({ page }) => {
    await expect(page.locator('#features h2')).toHaveText('Features');
  });

  test('section has subtitle', async ({ page }) => {
    await expect(page.locator('#features .subtitle')).toContainText(
      'Everything you need for comprehensive testing'
    );
  });

  test('has exactly 9 feature cards', async ({ page }) => {
    await expect(page.locator('.feature-card')).toHaveCount(9);
  });

  for (let i = 0; i < FEATURE_TITLES.length; i++) {
    test(`card ${i + 1}: "${FEATURE_TITLES[i]}" exists`, async ({ page }) => {
      const title = page.locator('.feature-card h3').nth(i);
      await expect(title).toHaveText(FEATURE_TITLES[i]);
    });
  }

  test('each feature card has a description', async ({ page }) => {
    const descriptions = page.locator('.feature-card p');
    const count = await descriptions.count();
    expect(count).toBe(9);
    for (let i = 0; i < count; i++) {
      const text = await descriptions.nth(i).textContent();
      expect(text!.length).toBeGreaterThan(10);
    }
  });

  test('each feature card has an icon', async ({ page }) => {
    await expect(page.locator('.feature-icon')).toHaveCount(9);
  });
});
