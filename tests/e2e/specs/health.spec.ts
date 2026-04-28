import { test, expect } from '@playwright/test';

test.describe('Site Health', () => {
  test('returns 200 OK', async ({ request }) => {
    const response = await request.get('/');
    expect(response.status()).toBe(200);
  });

  test('serves HTML content type', async ({ request }) => {
    const response = await request.get('/');
    expect(response.headers()['content-type']).toContain('text/html');
  });

  test('responds under 5 seconds', async ({ request }) => {
    const start = Date.now();
    await request.get('/');
    expect(Date.now() - start).toBeLessThan(5000);
  });

  test('returns 404 for nonexistent page', async ({ request }) => {
    const response = await request.get('/this-page-does-not-exist');
    expect(response.status()).toBe(404);
  });

  test('CSS loads correctly', async ({ request }) => {
    const response = await request.get('/css/style.css');
    expect(response.status()).toBe(200);
    expect(response.headers()['content-type']).toContain('text/css');
  });

  test('logo image loads correctly', async ({ request }) => {
    const response = await request.get('/img/logo.png');
    expect(response.status()).toBe(200);
    expect(response.headers()['content-type']).toContain('image/png');
  });

  test('CNAME file serves custom domain', async ({ request }) => {
    const response = await request.get('/CNAME');
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('scrutineer.asymmetric-effort.com');
  });
});
