import { test, expect } from '@playwright/test';

test.describe('Link Integrity', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('all external links use HTTPS', async ({ page }) => {
    const hrefs = await page.locator('a[href^="http"]').evaluateAll(
      els => els.map(el => (el as HTMLAnchorElement).href)
    );
    for (const href of hrefs) {
      expect(href).toMatch(/^https:\/\//);
    }
  });

  test('no anchor links point to missing IDs', async ({ page }) => {
    const broken = await page.evaluate(() => {
      const anchors = document.querySelectorAll('a[href^="#"]');
      const broken: string[] = [];
      anchors.forEach(a => {
        const id = a.getAttribute('href')!.slice(1);
        if (id && !document.getElementById(id)) {
          broken.push(id);
        }
      });
      return broken;
    });
    expect(broken).toHaveLength(0);
  });

  test('no links have empty href', async ({ page }) => {
    await expect(page.locator('a[href=""]')).toHaveCount(0);
  });

  test('no links use javascript: protocol', async ({ page }) => {
    await expect(page.locator('a[href^="javascript:"]')).toHaveCount(0);
  });

  test('GitHub repo links are consistent', async ({ page }) => {
    const ghLinks = page.locator('a[href*="github.com/asymmetric-effort/scrutineer"]');
    expect(await ghLinks.count()).toBeGreaterThan(2);
  });
});
