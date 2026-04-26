# Browser Connector

The browser connector provides headless browser automation using the Chrome DevTools Protocol (CDP). It supports Chromium, Firefox, and WebKit, with capabilities for page navigation, element interaction, screenshots, JavaScript evaluation, and element waiting. It is identified as `browser` in YAML test definitions.

Source: `connector/browser/`, including sub-packages `cdp/`, `selector/`, `install/`

## Setup Configuration

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `browser` | `string` | No | `"chromium"` | Browser type. Supported values: `"chromium"`, `"firefox"`, `"webkit"`. |
| `headless` | `bool` | No | `true` | Run the browser in headless mode. |
| `args` | `[]string` | No | `[]` | Additional command-line arguments passed to the browser process. Accepts both `[]string` and `[]any` (from YAML parsing). |
| `browser_path` | `string` | No | Auto-detected | Override the browser binary path. If not set, the install manager resolves the path based on browser type and known revision. |

### Browser Installation

Browsers are managed by the `install.Manager`, which stores binaries under `~/.scrutineer/browsers/` by default. The expected path follows the pattern:

```
~/.scrutineer/browsers/<browser>-<revision>/<binary_name>
```

Known-good revisions (from Playwright's CDN):

| Browser | Revision | Base URL |
|---------|----------|----------|
| chromium | 1148 | `https://playwright.azureedge.net/builds/chromium` |
| firefox | 1467 | `https://playwright.azureedge.net/builds/firefox` |
| webkit | 2098 | `https://playwright.azureedge.net/builds/webkit` |

Install browsers with: `scrutineer browsers install`

### Platform-Specific Binary Names

| Browser | Linux | macOS | Windows |
|---------|-------|-------|---------|
| chromium | `chrome` | `Chromium.app/Contents/MacOS/Chromium` | `chrome.exe` |
| firefox | `firefox` | `Nightly.app/Contents/MacOS/firefox` | `firefox.exe` |
| webkit | `minibrowser` | `MiniBrowser.app/Contents/MacOS/MiniBrowser` | `MiniBrowser.exe` |

### Download URL Format

```
<base_url>/<revision>/<browser>-<platform_suffix>.zip
```

Platform suffixes: `linux`, `linux-arm64`, `mac`, `mac-arm64`, `win64`, `win64-arm64`.

### Browser Launch Arguments

#### Chromium

Chromium is launched with hardened flags for automation:

- `--no-first-run`, `--no-default-browser-check`
- `--disable-background-networking`, `--disable-background-timer-throttling`
- `--disable-backgrounding-occluded-windows`, `--disable-breakpad`
- `--disable-component-extensions-with-background-pages`, `--disable-component-update`
- `--disable-default-apps`, `--disable-dev-shm-usage`
- `--disable-extensions`, `--disable-hang-monitor`
- `--disable-ipc-flooding-protection`, `--disable-popup-blocking`
- `--disable-prompt-on-repost`, `--disable-renderer-backgrounding`
- `--disable-sync`, `--disable-translate`
- `--metrics-recording-only`, `--no-startup-window`
- `--password-store=basic`, `--use-mock-keychain`
- `--remote-debugging-port=0` (OS assigns a free port)
- `--headless=new` (when headless mode is enabled)

#### Firefox

- `--no-remote`, `--new-instance`, `-wait-for-browser`
- `--remote-debugging-port 0`
- `--headless` (when headless mode is enabled)

#### WebKit

- `--inspector-pipe`
- `--headless` (when headless mode is enabled)

### Connection Flow

1. The browser process is launched with appropriate arguments.
2. The WebSocket debugger URL is extracted from the browser's stderr output (searching for `ws://` in the output).
3. A CDP WebSocket client connects to the URL.
4. A new browser target (page) is created via `Target.createTarget`.
5. A CDP session is attached to the target via `Target.attachToTarget`.
6. Required CDP domains are enabled: `Page`, `Runtime`, `DOM`, `Network`.

### Setup Examples

#### Default Chromium headless

```yaml
connector: browser
config:
  browser: chromium
  headless: true
```

#### Firefox with custom args

```yaml
connector: browser
config:
  browser: firefox
  headless: true
  args:
    - "--width=1920"
    - "--height=1080"
```

#### Custom browser path

```yaml
connector: browser
config:
  browser: chromium
  browser_path: /usr/bin/google-chrome
  headless: true
```

## Actions

All actions operate on the current page session. Element-based actions use selectors to find target elements.

### Selector Types

Every action that targets an element accepts `selector` and `selector_type` parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | `string` | Yes (for element actions) | The selector value. |
| `selector_type` | `string` | No | Selector strategy. Default: `"css"`. |

Supported selector types:

#### CSS (`"css"`, default)

Uses `document.querySelector(selector)`. Standard CSS selector syntax.

```yaml
selector: "#login-button"
selector_type: css
```

```yaml
selector: "div.container > p.intro"
```

#### XPath (`"xpath"`)

Uses `document.evaluate()` with `FIRST_ORDERED_NODE_TYPE`. Full XPath expression syntax.

```yaml
selector: "//button[@type='submit']"
selector_type: xpath
```

#### Text (`"text"`)

Walks the DOM tree using `TreeWalker` with `NodeFilter.SHOW_TEXT`. Finds the first element whose trimmed text content matches exactly.

```yaml
selector: "Sign In"
selector_type: text
```

#### Role (`"role"`)

Uses `document.querySelector('[role="<value>"]')` to find elements by their ARIA role attribute.

```yaml
selector: "dialog"
selector_type: role
```

---

## Action: `navigate`

Navigates the page to a URL.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | `string` | Yes | The URL to navigate to. |

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `url` | `string` | The URL that was navigated to. |
| `frameId` | `string` | The CDP frame ID of the navigated frame. |

If the navigation produces an error (e.g., DNS failure), the `errorText` from the CDP response is returned as an error.

### Example

```yaml
steps:
  - connector: browser
    action: navigate
    parameters:
      url: https://example.com
    assert:
      - path: url
        equals: "https://example.com"
```

---

## Action: `click`

Clicks an element by computing its bounding box center and dispatching mouse events.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | `string` | Yes | Element selector. |
| `selector_type` | `string` | No | Selector type (default: `"css"`). |

### Behavior

1. The element's bounding rectangle is computed via `getBoundingClientRect()`.
2. The center point `(x + width/2, y + height/2)` is calculated.
3. Three `Input.dispatchMouseEvent` CDP calls are made: `mouseMoved`, `mousePressed` (left button, click count 1), `mouseReleased`.

### Result

Returns no data keys (action-only). Errors if the element is not found.

### Example

```yaml
steps:
  - connector: browser
    action: click
    parameters:
      selector: "#submit-btn"
```

---

## Action: `type`

Focuses an element and dispatches individual key events for each character.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | `string` | Yes | Element selector. |
| `selector_type` | `string` | No | Selector type. |
| `text` | `string` | No | The text to type, character by character. |

### Behavior

1. The element is focused via `DOM.focus`.
2. For each character in `text`, a `keyDown` and `keyUp` `Input.dispatchKeyEvent` is dispatched with the character as the `text` parameter.

### Example

```yaml
steps:
  - connector: browser
    action: type
    parameters:
      selector: "#search-input"
      text: "scrutineer testing"
```

---

## Action: `fill`

Clears an input element and sets its value directly via JavaScript, dispatching `input` and `change` events.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | `string` | Yes | Element selector. |
| `selector_type` | `string` | No | Selector type. |
| `value` | `string` | No | The value to set on the element. |

### Behavior

1. The element is found and focused.
2. `el.value` is set directly.
3. `input` and `change` events are dispatched (with `bubbles: true`).

This is faster than `type` and suitable for filling form fields where individual keystroke simulation is not needed.

### Example

```yaml
steps:
  - connector: browser
    action: fill
    parameters:
      selector: "input[name='email']"
      value: "user@example.com"
```

---

## Action: `select`

Selects an option in a `<select>` element by value.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | `string` | Yes | Element selector (must target a `<select>` element). |
| `selector_type` | `string` | No | Selector type. |
| `value` | `string` | No | The option value to select. |

### Behavior

1. Verifies the element is a `<select>` tag.
2. Sets `el.value` to the specified option value.
3. Dispatches `input` and `change` events.

### Example

```yaml
steps:
  - connector: browser
    action: select
    parameters:
      selector: "#country-select"
      value: "US"
```

---

## Action: `screenshot`

Captures a screenshot of the current page.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `format` | `string` | No | `"png"` | Image format: `"png"` or `"jpeg"`. |
| `quality` | `int` | No | 0 | JPEG quality (1-100). Only used when format is `"jpeg"`. |
| `full_page` | `bool` | No | `false` | Capture the full scrollable page. When true, uses `Page.getLayoutMetrics` to determine the full content size and sets a clip region. |
| `path` | `string` | No | -- | File path to save the screenshot. The base64 data is decoded and written to this path. |

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `data` | `string` | Base64-encoded screenshot image data. |
| `path` | `string` | The file path where the screenshot was saved (only present if `path` parameter was provided). |

### Example

```yaml
steps:
  - connector: browser
    action: screenshot
    parameters:
      format: png
      full_page: true
      path: /tmp/screenshots/homepage.png
    assert:
      - path: data
        not_empty: true
```

---

## Action: `evaluate`

Evaluates a JavaScript expression in the page context.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `expression` | `string` | Yes | The JavaScript expression to evaluate. Promises are automatically awaited (`awaitPromise: true`). |

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `value` | `any` | The return value of the expression. Returned by value (serialized). |

If the expression throws an exception, an error is returned with the exception text.

### Example

```yaml
steps:
  - connector: browser
    action: evaluate
    parameters:
      expression: "document.title"
    assert:
      - path: value
        equals: "Example Domain"
```

#### Evaluate with complex return value

```yaml
steps:
  - connector: browser
    action: evaluate
    parameters:
      expression: |
        (() => {
          const links = document.querySelectorAll('a');
          return Array.from(links).map(a => a.href);
        })()
    assert:
      - path: value
        not_empty: true
```

---

## Action: `wait_for_selector`

Polls until an element matching the selector appears in the DOM.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | `string` | Yes | Element selector to wait for. |
| `selector_type` | `string` | No | Selector type. |

### Timeout

Uses `step.Timeout` if set; otherwise defaults to **30 seconds**. The polling interval is **100 milliseconds**.

### Behavior

Repeatedly evaluates the selector expression until it returns a non-null, non-undefined result, or until the timeout expires. Respects context cancellation.

### Example

```yaml
steps:
  - connector: browser
    action: wait_for_selector
    timeout: 10s
    parameters:
      selector: ".results-loaded"
```

---

## Action: `get_text`

Returns the `innerText` of an element found by selector.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | `string` | Yes | Element selector. |
| `selector_type` | `string` | No | Selector type. |

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `text` | `string` | The `innerText` of the matched element. |

### Example

```yaml
steps:
  - connector: browser
    action: get_text
    parameters:
      selector: "h1.page-title"
    assert:
      - path: text
        equals: "Welcome"
```

---

## Action: `get_attribute`

Returns the value of a specific attribute on an element.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | `string` | Yes | Element selector. |
| `selector_type` | `string` | No | Selector type. |
| `attribute` | `string` | Yes | The attribute name to retrieve (e.g., `"href"`, `"class"`, `"data-id"`). |

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `value` | `string` | The attribute value. |

### Example

```yaml
steps:
  - connector: browser
    action: get_attribute
    parameters:
      selector: "a.main-link"
      attribute: href
    assert:
      - path: value
        contains: "example.com"
```

---

## CDP Internals Overview

The browser connector communicates with browsers using a custom CDP client built on the Go standard library (no external WebSocket or CDP libraries).

### Architecture

```
BrowserConnector
  |
  +-- browserProcess (manages the OS process)
  |     |-- launches browser with appropriate flags
  |     |-- extracts ws:// URL from stderr
  |     |-- provides kill() for cleanup
  |
  +-- cdp.Client (WebSocket JSON-RPC client)
  |     |-- wsConn (raw WebSocket over net.Conn)
  |     |-- readLoop goroutine for receiving messages
  |     |-- pending map for request/response correlation
  |     |-- event subscription system
  |     |-- Send(method, params) for browser-level commands
  |
  +-- cdp.Session (target-scoped CDP session)
        |-- attached to a specific page target
        |-- Send(method, params) for page-level commands
        |-- sessionId scoping for all messages
```

### Protocol

The CDP protocol uses JSON-RPC over WebSocket:

- **Request**: `{id, method, params, sessionId?}`
- **Response**: `{id, result?, error?}`
- **Event**: `{method, params, sessionId?}`

Responses are correlated by `id` using a concurrent-safe pending map. Events are dispatched to registered handlers asynchronously.

### Error Handling

CDP errors include a numeric `code`, a `message`, and optional `data`. They implement the Go `error` interface.

## Common Workflow Examples

### Login flow

```yaml
connector: browser
config:
  browser: chromium
  headless: true

steps:
  - action: navigate
    parameters:
      url: https://app.example.com/login

  - action: fill
    parameters:
      selector: "input[name='username']"
      value: "testuser"

  - action: fill
    parameters:
      selector: "input[name='password']"
      value: "testpass123"

  - action: click
    parameters:
      selector: "button[type='submit']"

  - action: wait_for_selector
    timeout: 10s
    parameters:
      selector: ".dashboard"

  - action: get_text
    parameters:
      selector: ".welcome-message"
    assert:
      - path: text
        contains: "Welcome, testuser"
```

### Form submission with validation

```yaml
steps:
  - action: navigate
    parameters:
      url: https://app.example.com/form

  - action: fill
    parameters:
      selector: "#name"
      value: "Jane Doe"

  - action: fill
    parameters:
      selector: "#email"
      value: "jane@example.com"

  - action: select
    parameters:
      selector: "#department"
      value: "engineering"

  - action: click
    parameters:
      selector: "#submit"

  - action: wait_for_selector
    parameters:
      selector: ".success-message"

  - action: screenshot
    parameters:
      path: /tmp/form-submitted.png
```

### Page content verification

```yaml
steps:
  - action: navigate
    parameters:
      url: https://example.com

  - action: evaluate
    parameters:
      expression: "document.title"
    assert:
      - path: value
        equals: "Example Domain"

  - action: get_text
    parameters:
      selector: "h1"
    assert:
      - path: text
        equals: "Example Domain"

  - action: get_attribute
    parameters:
      selector: "a"
      selector_type: css
      attribute: href
    assert:
      - path: value
        contains: "iana.org"
```

### XPath and text selectors

```yaml
steps:
  - action: navigate
    parameters:
      url: https://example.com

  - action: get_text
    parameters:
      selector: "//h1"
      selector_type: xpath
    assert:
      - path: text
        not_empty: true

  - action: click
    parameters:
      selector: "More information..."
      selector_type: text
```

## Teardown

The `Teardown` method performs cleanup in order:

1. Detaches the CDP session from the target (`Target.detachFromTarget`).
2. Closes the CDP WebSocket client connection.
3. Kills the browser process (sends cancel signal, then `Process.Kill`, then waits).

Errors from each step are collected and returned as a combined error.
