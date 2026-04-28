import { test, expect } from '@playwright/test';

test.describe('HTML Document Structure', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('has correct title', async ({ page }) => {
    await expect(page).toHaveTitle(/Scrutineer/);
  });

  test('has meta description', async ({ page }) => {
    const description = await page.getAttribute('meta[name="description"]', 'content');
    expect(description).toContain('extensible test framework');
  });

  test('has viewport meta tag', async ({ page }) => {
    const viewport = await page.getAttribute('meta[name="viewport"]', 'content');
    expect(viewport).toContain('width=device-width');
  });

  test('has charset declaration', async ({ page }) => {
    const charset = await page.locator('meta[charset]').getAttribute('charset');
    expect(charset).toBe('UTF-8');
  });

  test('has favicon', async ({ page }) => {
    const favicon = await page.locator('link[rel="icon"]').getAttribute('href');
    expect(favicon).toContain('logo.png');
  });

  test('has stylesheet', async ({ page }) => {
    const stylesheet = await page.locator('link[rel="stylesheet"]').getAttribute('href');
    expect(stylesheet).toContain('style.css');
  });

  test('uses HTML5 doctype', async ({ page }) => {
    const html = await page.content();
    expect(html).toContain('<!DOCTYPE html>');
  });

  test('has lang attribute set to en', async ({ page }) => {
    const lang = await page.locator('html').getAttribute('lang');
    expect(lang).toBe('en');
  });
});
