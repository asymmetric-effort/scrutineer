# Scrutineer Website — E2E Demo

Smoke tests for [scrutineer.asymmetric-effort.com](https://scrutineer.asymmetric-effort.com/) demonstrating scrutineer's ability to run basic E2E web tests via HTTP and browser connectors.

This demo runs **after** Playwright PDV passes in CI, validating that scrutineer can perform the same class of tests.

## Running

```bash
scrutineer run --config demos/scrutineer-website/scrutineer.yaml
```

## Test Suites

| Suite | Tests | Connector | What It Proves |
|-------|-------|-----------|----------------|
| HTTP Smoke Tests | 4 | http | GET requests, status codes, headers, content type validation |
| Browser Smoke Tests | 8 | browser | Page load, DOM queries, element text, JS evaluation, screenshots |

**Total: 12 tests**

## What This Demo Validates

This is not an exhaustive site test — Playwright handles full PDV. This demo proves scrutineer can:

- Make HTTP requests and assert on status, headers, and body content
- Launch a headless browser and navigate to a URL
- Wait for DOM selectors to appear
- Extract element text and assert on it
- Evaluate JavaScript expressions in the page context
- Count DOM elements and verify structure
- Capture screenshots
- Validate link integrity via JS evaluation
