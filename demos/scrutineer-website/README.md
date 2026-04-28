# Scrutineer Website E2E Tests

End-to-end test suite for [scrutineer.asymmetric-effort.com](https://scrutineer.asymmetric-effort.com/) using scrutineer itself.

## Running

```bash
scrutineer run --config demos/scrutineer-website/scrutineer.yaml
```

## Test Suites

| Suite | File | Tests | Connector | Description |
|-------|------|-------|-----------|-------------|
| HTTP Health Checks | `http-health.test.yaml` | 5 | http | Status codes, response times, HTTPS enforcement |
| HTTP Security Headers | `http-headers.test.yaml` | 5 | http | Content-Type, CSS, logo serving |
| Static Asset Delivery | `http-assets.test.yaml` | 4 | http | HTML, CSS, logo, CNAME file integrity |
| HTML Meta Tags | `html-meta.test.yaml` | 7 | http | Title, description, viewport, charset, favicon, stylesheet, HTML5 validity |
| Navigation Bar | `browser-navigation.test.yaml` | 8 | browser | Logo, nav links, anchor hrefs, link count |
| Hero Section | `browser-hero.test.yaml` | 10 | browser | Heading, tagline, badges, CTA buttons |
| Features Section | `browser-features.test.yaml` | 14 | browser | All 9 feature cards, icons, descriptions |
| Supported Configurations | `browser-configurations.test.yaml` | 17 | browser | Table headers, all 10 rows, status styling |
| Install Section | `browser-install.test.yaml` | 11 | browser | Go install command, quick start, all CLI commands |
| Cross-Platform | `browser-cross-platform.test.yaml` | 7 | browser | Linux/macOS/Windows cards, arch support |
| Footer | `browser-footer.test.yaml` | 8 | browser | Copyright, version, ARIA, three-column layout |
| External Links | `browser-links.test.yaml` | 6 | browser | HTTPS-only, no broken anchors, no empty hrefs |
| Code Example | `browser-code-example.test.yaml` | 10 | browser | YAML syntax, assertions, captures, interpolation |

**Total: 13 test suites, 112 tests**

## What's Validated

- **HTTP layer**: Status codes, headers, content types, TLS, response times
- **Document structure**: HTML5 validity, meta tags, SEO elements, favicon
- **Navigation**: All nav links, anchor targets, logo, link count
- **Hero**: Heading, tagline content, all 3 badges, both CTA buttons
- **Features**: All 9 feature cards with correct titles, descriptions, icons
- **Configurations table**: All 10 rows, column headers, status styling, planned vs shipped
- **Install section**: Go install command, quick start commands, platform availability
- **Cross-platform**: All 3 platform cards with correct OS/arch info
- **Code example**: Complete YAML syntax demonstration (suite, steps, assertions, captures, interpolation)
- **Footer**: Copyright format, version display, company link, GitHub link, ARIA accessibility, layout
- **Link integrity**: All external links use HTTPS, no broken anchors, no empty/javascript hrefs
