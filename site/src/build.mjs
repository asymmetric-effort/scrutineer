import { createElement as h, Fragment } from '@asymmetric-effort/specifyjs';
import { renderToStaticMarkup } from '@asymmetric-effort/specifyjs/server';
import { Footer as SpecFooter } from '@asymmetric-effort/specifyjs/components';
import { specifyJsSeoPlugin } from '@asymmetric-effort/specifyjs/build';
import { writeFileSync, mkdirSync, cpSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';

const currentYear = new Date().getFullYear();
const projectVersion = execSync('git describe --tags --abbrev=0 2>/dev/null || echo "0.0.1-dev"', { encoding: 'utf-8', cwd: join(dirname(fileURLToPath(import.meta.url)), '../..') }).trim();

const __dirname = dirname(fileURLToPath(import.meta.url));
const outDir = join(__dirname, '..', 'dist');

// --- Components ---

function Nav() {
    return h('nav', null,
        h('div', { className: 'nav-inner' },
            h('a', { href: '/', className: 'logo' },
                h('img', { src: 'img/logo.png', alt: 'Scrutineer', className: 'logo-icon', width: '28', height: '28' }),
                ' scrutineer'
            ),
            h('ul', { className: 'nav-links' },
                h('li', null, h('a', { href: '#features' }, 'Features')),
                h('li', null, h('a', { href: '#protocols' }, 'Protocols')),
                h('li', null, h('a', { href: '#install' }, 'Install')),
                h('li', null, h('a', { href: 'https://github.com/asymmetric-effort/scrutineer' }, 'GitHub'))
            )
        )
    );
}

function Badge({ text, color }) {
    return h('span', { className: `badge badge-${color}` }, text);
}

function Hero() {
    return h('div', { className: 'hero' },
        h('h1', null, 'scrutineer'),
        h('p', { className: 'tagline' },
            'An extensible test framework for CLI programs, REST APIs, GraphQL, gRPC, browsers, and network protocols. Declarative YAML tests. Zero third-party dependencies.'
        ),
        h('div', { className: 'badges' },
            h(Badge, { text: 'Go 1.26+', color: 'green' }),
            h(Badge, { text: 'MIT License', color: 'blue' }),
            h(Badge, { text: 'v0.0.1-dev', color: 'orange' })
        ),
        h('div', { className: 'cta-group' },
            h('a', { href: '#install', className: 'btn btn-primary' }, 'Get Started'),
            h('a', { href: 'https://github.com/asymmetric-effort/scrutineer', className: 'btn btn-secondary' }, 'View Source')
        ),
        h(CodeExample, null)
    );
}

function CodeExample() {
    const code = `suite: "User API Tests"
tags: [api, smoke]

tests:
  - name: "Create user returns 201"
    connector: http
    steps:
      - action: request
        method: POST
        path: /users
        body:
          name: "Alice"
          email: "alice@example.com"
        assert:
          - status: 201
          - body.name: {equals: "Alice"}
          - elapsed: {less_than: 2s}
        capture:
          user_id: body.id

  - name: "Verify user exists"
    connector: http
    steps:
      - action: request
        method: GET
        path: /users/\${capture.user_id}
        assert:
          - status: 200
          - body.email: {equals: "alice@example.com"}`;

    return h('div', { className: 'code-example' },
        h('div', { className: 'code-header' }, 'example.test.yaml'),
        h('pre', null, h('code', null, code))
    );
}

function FeatureCard({ icon, title, description }) {
    return h('div', { className: 'feature-card' },
        h('span', { className: 'feature-icon' }, icon),
        h('h3', null, title),
        h('p', null, description)
    );
}

function Features() {
    const features = [
        { icon: '\u25B7', title: 'Declarative YAML Tests', description: 'Define tests as data, not code. Describe what to assert, not how to execute. Familiar to Playwright and assertion-based test users.' },
        { icon: '\u2699', title: 'Modular Connectors', description: 'CLI, HTTP, SSH, gRPC, GraphQL, and browser connectors. Add new protocols by implementing a single Go interface.' },
        { icon: '\u2637', title: 'Browser Automation', description: 'Headless Chromium, Firefox, and WebKit via Chrome DevTools Protocol. Selectors, interactions, screenshots, network interception.' },
        { icon: '\u21B1', title: 'Load Testing', description: 'Parallel test execution distributed across nodes via SSH. Configurable concurrency, ramp-up, and duration. Locust-style scaling.' },
        { icon: '\u23F1', title: 'Nanosecond Telemetry', description: 'Every test captures timing data automatically. Structured binary TLV logs with nanosecond timestamps for benchmark analysis.' },
        { icon: '\u25A0', title: 'Zero Dependencies', description: 'Built with the Go standard library. No node_modules, no pip packages, no dependency hell. One binary, every platform.' },
        { icon: '\u2696', title: 'Coverage as a Feature', description: 'Built-in test coverage measurement with configurable thresholds. Know exactly which tests ran, which steps executed, which assertions fired.' },
        { icon: '\u2726', title: 'Fuzz Testing', description: 'Declarative fuzz targets integrated with Go\'s built-in fuzzing. Corpus management and automated edge-case discovery.' },
        { icon: '\u270E', title: 'Rich Assertions', description: 'Equality, contains, regex, JSON path, HTTP status, headers, timing, collections. Extensible assertion library with clear error messages.' },
    ];

    return h('section', { id: 'features' },
        h('h2', null, 'Features'),
        h('p', { className: 'subtitle' }, 'Everything you need for comprehensive testing, in a single binary with zero dependencies.'),
        h('div', { className: 'feature-grid' },
            ...features.map(f => h(FeatureCard, f))
        )
    );
}

function ProtocolRow({ protocol, connector, status, features }) {
    return h('tr', null,
        h('td', null, protocol),
        h('td', null, connector),
        h('td', { className: status === 'v0.0.1' ? 'status-yes' : 'status-planned' }, status),
        h('td', null, features)
    );
}

function Protocols() {
    const rows = [
        { protocol: 'HTTP/1.1, HTTP/2', connector: 'http', status: 'v0.0.1', features: 'TLS 1.2/1.3, self-signed certs, request/response assertions' },
        { protocol: 'REST APIs', connector: 'http', status: 'v0.0.1', features: 'CRUD, auth (Bearer, Basic, API key), JSON/XML, pagination, HATEOAS' },
        { protocol: 'GraphQL', connector: 'http', status: 'v0.0.1', features: 'Queries, mutations, subscriptions, introspection, variables' },
        { protocol: 'gRPC / Protobuf', connector: 'grpc', status: 'v0.0.1', features: 'Unary, client/server/bidi streaming, .proto + reflection' },
        { protocol: 'SSH', connector: 'ssh', status: 'v0.0.1', features: 'Key-based auth, command execution, tunneling' },
        { protocol: 'CLI Programs', connector: 'cli', status: 'v0.0.1', features: 'stdin/stdout/stderr, exit codes, filesystem side-effects' },
        { protocol: 'Chromium / Firefox / WebKit', connector: 'browser', status: 'v0.0.1', features: 'CDP, selectors, interactions, screenshots, network mocking' },
        { protocol: 'HTTP/3 (QUIC)', connector: 'http', status: 'planned', features: 'Pending Go stdlib or from-scratch QUIC' },
        { protocol: 'SMTP', connector: 'smtp', status: 'planned', features: 'Send, auth, envelope validation' },
        { protocol: 'IMAP', connector: 'imap', status: 'planned', features: 'Mailbox access, search, fetch' },
    ];

    return h('section', { id: 'protocols' },
        h('h2', null, 'Supported Configurations'),
        h('p', { className: 'subtitle' }, 'Test anything that speaks a protocol.'),
        h('table', { className: 'protocol-table' },
            h('thead', null,
                h('tr', null,
                    h('th', null, 'Feature'),
                    h('th', null, 'Connector'),
                    h('th', null, 'Status'),
                    h('th', null, 'Features')
                )
            ),
            h('tbody', null, ...rows.map(r => h(ProtocolRow, r)))
        )
    );
}

function Install() {
    return h('section', { id: 'install' },
        h('h2', null, 'Install'),
        h('p', { className: 'subtitle' }, 'One command. No dependencies.'),
        h('div', { className: 'install-block' },
            h('code', null, '$ go install github.com/asymmetric-effort/scrutineer/cmd/scrutineer@latest')
        ),
        h('br', null),
        h('p', { style: 'color: var(--text-muted); font-size: 0.9rem;' },
            'Or download a pre-built binary from ',
            h('a', { href: 'https://github.com/asymmetric-effort/scrutineer/releases' }, 'Releases'),
            ' for Linux, macOS, or Windows (AMD64 / ARM64).'
        ),
        h('div', { className: 'code-example', style: 'margin-top: 2rem;' },
            h('div', { className: 'code-header' }, 'Quick start'),
            h('pre', null, h('code', null,
`# Install scrutineer
$ go install github.com/asymmetric-effort/scrutineer/cmd/scrutineer@latest

# Install browsers (for browser testing)
$ scrutineer browsers install

# Run tests
$ scrutineer run

# Run with JSON output
$ scrutineer run --format json

# Dump binary telemetry logs
$ scrutineer log-dump scrutineer.log`
            ))
        )
    );
}

function CrossPlatform() {
    return h('section', null,
        h('h2', null, 'Cross-Platform'),
        h('p', { className: 'subtitle' }, 'Build once, test everywhere.'),
        h('div', { className: 'platform-grid' },
            h('div', { className: 'platform-card' }, h('h3', null, 'Linux'), h('p', null, 'AMD64 / ARM64')),
            h('div', { className: 'platform-card' }, h('h3', null, 'macOS'), h('p', null, 'AMD64 / ARM64')),
            h('div', { className: 'platform-card' }, h('h3', null, 'Windows'), h('p', null, 'AMD64 / ARM64'))
        )
    );
}

function Footer() {
    const left = h('span', null, `Scrutineer ${projectVersion}`);

    const center = h('span', null,
        `\u00A9 2022-${currentYear} `,
        h('a', {
            href: 'https://asymmetric-effort.com',
            style: { color: '#3b82f6', textDecoration: 'none' }
        }, 'Asymmetric Effort, LLC'),
        '. MIT License.'
    );

    const right = h('a', {
        href: 'https://github.com/asymmetric-effort/scrutineer',
        style: { color: '#3b82f6', textDecoration: 'none' }
    }, 'GitHub Repository');

    return h(SpecFooter, { left, center, right });
}

function Page() {
    return h(Fragment, null,
        h(Nav, null),
        h(Hero, null),
        h(Features, null),
        h(Protocols, null),
        h(Install, null),
        h(CrossPlatform, null),
        h(Footer, null)
    );
}

// --- Build ---

const html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Scrutineer — Extensible Test Framework</title>
    <meta name="description" content="An extensible test framework for automating tests against CLI programs, APIs, and web applications. Zero third-party dependencies. Built in Go.">
    <link rel="icon" type="image/png" href="img/logo.png">
    <link rel="stylesheet" href="css/style.css">
</head>
<body>
${renderToStaticMarkup(h(Page, null))}
</body>
</html>`;

// Write output
mkdirSync(outDir, { recursive: true });
mkdirSync(join(outDir, 'css'), { recursive: true });
mkdirSync(join(outDir, 'img'), { recursive: true });
writeFileSync(join(outDir, 'index.html'), html);
cpSync(join(__dirname, '..', 'public', 'css', 'style.css'), join(outDir, 'css', 'style.css'));
cpSync(join(__dirname, '..', 'public', 'img', 'logo.png'), join(outDir, 'img', 'logo.png'));
cpSync(join(__dirname, '..', 'CNAME'), join(outDir, 'CNAME'));

// Generate SEO files (sitemap.xml, robots.txt, llms.txt)
const seoPlugin = specifyJsSeoPlugin({
    siteUrl: 'https://scrutineer.asymmetric-effort.com',
    routes: ['/'],
    title: 'Scrutineer',
    description: 'An extensible test framework for automating tests against CLI programs, APIs, and web applications. Zero third-party dependencies. Built in Go.',
    repository: 'https://github.com/asymmetric-effort/scrutineer',
    license: 'MIT',
    author: 'Asymmetric Effort, LLC',
});
seoPlugin.closeBundle();

console.log('Site built to', outDir);
console.log('  index.html:', html.length, 'bytes');
