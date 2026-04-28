import { test, expect } from '@playwright/test';

test.describe('Code Example', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('code example block exists', async ({ page }) => {
    await expect(page.locator('.code-example').first()).toBeVisible();
  });

  test('code header shows example.test.yaml', async ({ page }) => {
    await expect(page.locator('.code-header').first()).toHaveText('example.test.yaml');
  });

  test('code block contains suite definition', async ({ page }) => {
    const code = page.locator('pre code').first();
    await expect(code).toContainText('suite:');
    await expect(code).toContainText('User API Tests');
  });

  test('code block shows HTTP POST method', async ({ page }) => {
    const code = page.locator('pre code').first();
    await expect(code).toContainText('method: POST');
    await expect(code).toContainText('path: /users');
  });

  test('code block shows assertions', async ({ page }) => {
    const code = page.locator('pre code').first();
    await expect(code).toContainText('assert:');
    await expect(code).toContainText('status: 201');
    await expect(code).toContainText('equals:');
  });

  test('code block shows capture and interpolation', async ({ page }) => {
    const code = page.locator('pre code').first();
    await expect(code).toContainText('capture:');
    await expect(code).toContainText('user_id: body.id');
    await expect(code).toContainText('${capture.user_id}');
  });

  test('code block shows timing assertion', async ({ page }) => {
    const code = page.locator('pre code').first();
    await expect(code).toContainText('elapsed:');
    await expect(code).toContainText('less_than:');
  });

  test('code block shows tags', async ({ page }) => {
    const code = page.locator('pre code').first();
    await expect(code).toContainText('tags:');
    await expect(code).toContainText('api');
    await expect(code).toContainText('smoke');
  });

  test('code block shows GET verification step', async ({ page }) => {
    const code = page.locator('pre code').first();
    await expect(code).toContainText('Verify user exists');
    await expect(code).toContainText('method: GET');
  });
});
