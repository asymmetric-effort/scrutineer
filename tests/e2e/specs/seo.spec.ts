import { test, expect } from '@playwright/test';

test.describe('SEO — robots.txt', () => {
  test('returns 200 OK', async ({ request }) => {
    const response = await request.get('/robots.txt');
    expect(response.status()).toBe(200);
  });

  test('serves plain text content type', async ({ request }) => {
    const response = await request.get('/robots.txt');
    expect(response.headers()['content-type']).toContain('text/plain');
  });

  test('allows all user agents', async ({ request }) => {
    const response = await request.get('/robots.txt');
    const body = await response.text();
    expect(body).toContain('User-agent: *');
    expect(body).toContain('Allow: /');
  });

  test('references sitemap URL', async ({ request }) => {
    const response = await request.get('/robots.txt');
    const body = await response.text();
    expect(body).toContain('Sitemap: https://scrutineer.asymmetric-effort.com/sitemap.xml');
  });
});

test.describe('SEO — sitemap.xml', () => {
  test('returns 200 OK', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    expect(response.status()).toBe(200);
  });

  test('serves XML content type', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    const contentType = response.headers()['content-type'];
    expect(contentType).toMatch(/xml/);
  });

  test('has valid XML declaration', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    const body = await response.text();
    expect(body).toMatch(/^<\?xml version="1\.0" encoding="UTF-8"\?>/);
  });

  test('uses sitemaps.org schema', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    const body = await response.text();
    expect(body).toContain('xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"');
  });

  test('contains site root URL', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    const body = await response.text();
    expect(body).toContain('<loc>https://scrutineer.asymmetric-effort.com/</loc>');
  });

  test('contains lastmod date in ISO format', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    const body = await response.text();
    expect(body).toMatch(/<lastmod>\d{4}-\d{2}-\d{2}<\/lastmod>/);
  });

  test('has valid urlset structure', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    const body = await response.text();
    expect(body).toContain('<urlset');
    expect(body).toContain('</urlset>');
    expect(body).toContain('<url>');
    expect(body).toContain('</url>');
  });
});
