# Browser Testing Guide

Scrutineer provides headless browser automation through the Chrome DevTools Protocol (CDP). The browser connector supports Chromium, Firefox, and WebKit using Playwright's open-source patched browser builds. The CDP wire protocol client is implemented from scratch in Go -- no Playwright or Selenium libraries are used.

## Installing Browsers

Before running browser tests, install the browser binaries:

```bash
scrutineer browsers install
```

This downloads patched builds of Chromium, Firefox, and WebKit from vendor CDNs. List installed browsers:

```bash
scrutineer browsers list
```

Enable browsers in your `scrutineer.yaml`:

```yaml
browsers:
  chromium: true
  firefox: false
  webkit: false
```

## Writing Browser Tests

### Basic Structure

Browser tests use `connector: browser` and support actions for navigation, element interaction, assertions, and screenshots.

```yaml
suite: "Homepage Tests"

tests:
  - name: "Page loads successfully"
    connector: browser
    steps:
      - action: navigate
        url: "https://example.com"

      - action: get_text
        selector: "h1"
        assert:
          - field: text
            operator: equal
            expected: "Example Domain"
```

### Browser Configuration

Configure the browser connector in your `scrutineer.yaml` or per-step:

```yaml
connectors:
  browser:
    browser: chromium              # chromium, firefox, or webkit
    headless: true                 # run without visible window
    args:                          # additional browser launch arguments
      - "--disable-gpu"
      - "--no-sandbox"
```

You can also override the browser binary path:

```yaml
connectors:
  browser:
    browser_path: /usr/bin/chromium-browser
```

## Available Actions

### navigate

Navigate the page to a URL.

```yaml
- action: navigate
  url: "https://example.com/login"
```

**Result data:** `url` (string), `frameId` (string)

### click

Click an element.

```yaml
- action: click
  selector: "#submit-button"
```

### type

Type text into an element character by character (simulates keyboard input).

```yaml
- action: type
  selector: "#search-input"
  text: "search query"
```

### fill

Set the value of an input element directly (faster than `type`, but does not fire keyboard events).

```yaml
- action: fill
  selector: "#email"
  value: "user@example.com"
```

### select

Select an option from a `<select>` element.

```yaml
- action: select
  selector: "#country"
  value: "US"
```

### screenshot

Capture a screenshot of the page.

```yaml
- action: screenshot
  path: "screenshots/login-page.png"    # save to file (optional)
  format: png                            # png or jpeg
  quality: 80                            # jpeg quality (1-100)
  full_page: true                        # capture full scrollable area
```

**Result data:** `data` (base64-encoded image string), `path` (string, if saved)

### evaluate

Execute JavaScript in the page context.

```yaml
- action: evaluate
  expression: "document.title"
```

**Result data:** `value` (the JavaScript return value)

For more complex expressions:

```yaml
- action: evaluate
  expression: |
    (function() {
      var items = document.querySelectorAll('.item');
      return items.length;
    })()
```

### wait_for_selector

Wait until an element matching the selector appears in the DOM.

```yaml
- action: wait_for_selector
  selector: ".loaded-indicator"
  timeout: 10s
```

The default timeout is 30 seconds. Polling occurs every 100 milliseconds.

### get_text

Get the inner text of an element.

```yaml
- action: get_text
  selector: ".welcome-message"
```

**Result data:** `text` (string)

### get_attribute

Get the value of an element attribute.

```yaml
- action: get_attribute
  selector: "img.hero"
  attribute: "src"
```

**Result data:** `value` (string)

## Selectors

Scrutineer supports four selector types. The default is CSS.

### CSS Selectors (Default)

```yaml
- action: click
  selector: "#login-button"           # by ID
  selector_type: css                   # optional, css is the default

- action: click
  selector: ".nav-item.active"         # by class

- action: click
  selector: "button[type='submit']"    # by attribute

- action: click
  selector: "form > input:first-child" # combinators and pseudo-selectors
```

### XPath Selectors

```yaml
- action: click
  selector: "//button[@id='submit']"
  selector_type: xpath

- action: get_text
  selector: "//div[@class='content']/p[1]"
  selector_type: xpath
```

### Text Selectors

Find elements by their visible text content (exact match).

```yaml
- action: click
  selector: "Sign In"
  selector_type: text

- action: get_text
  selector: "Welcome back"
  selector_type: text
```

### Role Selectors

Find elements by their ARIA role attribute.

```yaml
- action: click
  selector: "button"
  selector_type: role

- action: get_text
  selector: "alert"
  selector_type: role
```

## Waiting Strategies

### Explicit Wait

Use `wait_for_selector` to wait for an element to appear:

```yaml
- action: wait_for_selector
  selector: "#results-loaded"
  timeout: 15s
```

### Step Timeout

Set a timeout on any step to limit how long it can take:

```yaml
- action: navigate
  url: "https://slow-site.example.com"
  timeout: 30s
```

### JavaScript-Based Waiting

For complex wait conditions, use `evaluate`:

```yaml
- action: evaluate
  expression: |
    new Promise(resolve => {
      const check = () => {
        if (document.readyState === 'complete') {
          resolve(true);
        } else {
          setTimeout(check, 100);
        }
      };
      check();
    })
  timeout: 10s
```

## Screenshots

### Capture to File

```yaml
- action: screenshot
  path: "screenshots/current-state.png"
```

### Full-Page Screenshot

Captures the entire scrollable area, not just the visible viewport:

```yaml
- action: screenshot
  path: "screenshots/full-page.png"
  full_page: true
```

### JPEG with Quality

```yaml
- action: screenshot
  path: "screenshots/compressed.jpg"
  format: jpeg
  quality: 60
```

### Screenshot as Data

If `path` is omitted, the base64-encoded image data is available in the result for assertions or captures:

```yaml
- action: screenshot
  format: png
  assert:
    - field: data
      operator: not_empty
  capture:
    screenshot_data: data
```

## JavaScript Evaluation

### Simple Expressions

```yaml
- action: evaluate
  expression: "document.title"
  assert:
    - field: value
      operator: equal
      expected: "My App"
```

### DOM Manipulation

```yaml
- action: evaluate
  expression: "document.querySelector('#hidden-field').value"
  capture:
    hidden_value: value
```

### Async Expressions

The evaluate action uses `awaitPromise: true`, so you can return promises:

```yaml
- action: evaluate
  expression: |
    fetch('/api/status').then(r => r.json()).then(data => data.healthy)
  assert:
    - field: value
      operator: equal
      expected: true
```

## Multi-Page Workflows

Chain multiple steps to test complete user flows. Use captures to pass data between steps.

### Complete Example: Login Flow

```yaml
suite: "Authentication"

fixtures:
  credentials:
    username: "testuser@example.com"
    password: "TestPassword123!"

tests:
  - name: "Successful login redirects to dashboard"
    connector: browser
    steps:
      # Navigate to login page
      - action: navigate
        url: "https://app.example.com/login"

      # Wait for the login form to load
      - action: wait_for_selector
        selector: "#login-form"
        timeout: 10s

      # Fill in credentials
      - action: fill
        selector: "#email"
        value: ${fixture.credentials.username}

      - action: fill
        selector: "#password"
        value: ${fixture.credentials.password}

      # Click the login button
      - action: click
        selector: "button[type='submit']"

      # Wait for redirect to dashboard
      - action: wait_for_selector
        selector: ".dashboard-header"
        timeout: 15s

      # Verify we are on the dashboard
      - action: evaluate
        expression: "window.location.pathname"
        assert:
          - field: value
            operator: equal
            expected: "/dashboard"

      # Verify welcome message
      - action: get_text
        selector: ".welcome-message"
        assert:
          - field: text
            operator: contains
            expected: "Welcome"

      # Take a screenshot for the test report
      - action: screenshot
        path: "screenshots/dashboard-after-login.png"

  - name: "Invalid login shows error message"
    connector: browser
    steps:
      - action: navigate
        url: "https://app.example.com/login"

      - action: wait_for_selector
        selector: "#login-form"

      - action: fill
        selector: "#email"
        value: "wrong@example.com"

      - action: fill
        selector: "#password"
        value: "wrong-password"

      - action: click
        selector: "button[type='submit']"

      # Wait for error message to appear
      - action: wait_for_selector
        selector: ".error-message"
        timeout: 10s

      - action: get_text
        selector: ".error-message"
        assert:
          - field: text
            operator: contains
            expected: "Invalid credentials"

      # Verify we are still on the login page
      - action: evaluate
        expression: "window.location.pathname"
        assert:
          - field: value
            operator: equal
            expected: "/login"

  - name: "Logout returns to login page"
    connector: browser
    steps:
      # Login first
      - action: navigate
        url: "https://app.example.com/login"

      - action: wait_for_selector
        selector: "#login-form"

      - action: fill
        selector: "#email"
        value: ${fixture.credentials.username}

      - action: fill
        selector: "#password"
        value: ${fixture.credentials.password}

      - action: click
        selector: "button[type='submit']"

      - action: wait_for_selector
        selector: ".dashboard-header"

      # Click logout
      - action: click
        selector: "#logout-button"

      # Verify redirect to login
      - action: wait_for_selector
        selector: "#login-form"
        timeout: 10s

      - action: evaluate
        expression: "window.location.pathname"
        assert:
          - field: value
            operator: equal
            expected: "/login"
```

## Best Practices

1. **Always use `wait_for_selector`** before interacting with elements that load dynamically. This prevents flaky tests caused by race conditions between navigation and DOM updates.

2. **Prefer `fill` over `type`** for most form inputs. `fill` sets the value directly and is faster. Use `type` only when you need to test keyboard event handling.

3. **Use CSS selectors by default.** They are the most familiar and widely supported. Switch to XPath for complex structural queries, text selectors for content-based lookups, and role selectors for accessibility testing.

4. **Take screenshots at failure points.** Screenshots captured before assertions make debugging much easier when tests fail.

5. **Use fixtures for test data.** Do not hardcode credentials, URLs, or test data in steps. Put them in the fixtures section for easy maintenance.

6. **Keep tests independent.** Each test gets a fresh browser context. Do not rely on state from previous tests -- perform setup (like login) within each test that needs it.

## Next Steps

- [Writing Tests](writing-tests.md) -- general test writing guide
- [CI Integration](ci-integration.md) -- running browser tests in CI
- [Installation](installation.md) -- installing browser binaries
