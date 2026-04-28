import { test, expect } from '@playwright/test';

test.describe('Supported Configurations Table', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('section heading is Supported Configurations', async ({ page }) => {
    await expect(page.locator('#protocols h2')).toHaveText('Supported Configurations');
  });

  test('section has subtitle', async ({ page }) => {
    await expect(page.locator('#protocols .subtitle')).toContainText('Test anything that speaks a protocol');
  });

  test('table has 4 column headers', async ({ page }) => {
    const headers = page.locator('.protocol-table th');
    await expect(headers).toHaveCount(4);
    await expect(headers.nth(0)).toHaveText('Feature');
    await expect(headers.nth(1)).toHaveText('Connector');
    await expect(headers.nth(2)).toHaveText('Status');
    await expect(headers.nth(3)).toHaveText('Features');
  });

  test('table has 10 data rows', async ({ page }) => {
    await expect(page.locator('.protocol-table tbody tr')).toHaveCount(10);
  });

  const v001Features = [
    { feature: 'HTTP/1.1, HTTP/2', connector: 'http', detail: 'TLS 1.2/1.3' },
    { feature: 'REST APIs', connector: 'http', detail: 'Bearer' },
    { feature: 'GraphQL', connector: 'http', detail: 'Queries' },
    { feature: 'gRPC', connector: 'grpc', detail: 'streaming' },
    { feature: 'SSH', connector: 'ssh', detail: 'tunneling' },
    { feature: 'CLI Programs', connector: 'cli', detail: 'stdin/stdout/stderr' },
    { feature: 'Chromium', connector: 'browser', detail: 'CDP' },
  ];

  for (let i = 0; i < v001Features.length; i++) {
    test(`row ${i + 1}: ${v001Features[i].feature}`, async ({ page }) => {
      const row = page.locator('.protocol-table tbody tr').nth(i);
      await expect(row).toContainText(v001Features[i].feature);
      await expect(row).toContainText(v001Features[i].connector);
      await expect(row).toContainText('v0.0.1');
      await expect(row).toContainText(v001Features[i].detail);
    });
  }

  test('HTTP/3 QUIC is listed as planned', async ({ page }) => {
    const row = page.locator('.protocol-table tbody tr').nth(7);
    await expect(row).toContainText('HTTP/3');
    await expect(row).toContainText('planned');
  });

  test('SMTP is listed as planned', async ({ page }) => {
    const row = page.locator('.protocol-table tbody tr').nth(8);
    await expect(row).toContainText('SMTP');
    await expect(row).toContainText('planned');
  });

  test('IMAP is listed as planned', async ({ page }) => {
    const row = page.locator('.protocol-table tbody tr').nth(9);
    await expect(row).toContainText('IMAP');
    await expect(row).toContainText('planned');
  });

  test('v0.0.1 items use green status class', async ({ page }) => {
    const greenItems = page.locator('.status-yes');
    expect(await greenItems.count()).toBeGreaterThan(0);
  });

  test('planned items use orange status class', async ({ page }) => {
    await expect(page.locator('.status-planned')).toHaveCount(3);
  });
});
