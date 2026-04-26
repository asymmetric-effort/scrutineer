// Package ssh implements an SSH connector for the scrutineer test framework.
//
// It supports remote command execution and SSH tunneling over authenticated
// SSH connections. Authentication can be key-based (file or raw PEM) or
// password-based. Host key verification is configurable.
package ssh

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/scrutineer/scrutineer/core/connector"
	cryptossh "golang.org/x/crypto/ssh"
)

// SSHConnector implements connector.Connector for SSH-based test steps.
type SSHConnector struct {
	client *cryptossh.Client
	config *cryptossh.ClientConfig
	addr   string

	mu        sync.Mutex
	listeners []net.Listener
	tunnelWg  sync.WaitGroup
	closed    chan struct{}
}

// New creates a new SSHConnector instance.
func New() *SSHConnector {
	return &SSHConnector{
		closed: make(chan struct{}),
	}
}

// Name returns the connector identifier used in YAML definitions.
func (c *SSHConnector) Name() string {
	return "ssh"
}

// Setup initializes the SSH connection using the provided configuration.
//
// Accepted configuration keys:
//   - "host" (string, required): remote host
//   - "port" (int, default 22): remote port
//   - "user" (string, required): SSH username
//   - "key_file" (string): path to PEM private key file
//   - "key" (string): raw PEM private key
//   - "password" (string): password for password-based auth
//   - "host_key_check" (bool, default true): verify remote host key
func (c *SSHConnector) Setup(ctx context.Context, config map[string]any) error {
	host, err := requireString(config, "host")
	if err != nil {
		return err
	}

	port := 22
	if v, ok := config["port"]; ok {
		p, err := toInt(v)
		if err != nil {
			return fmt.Errorf("ssh: invalid port: %w", err)
		}
		port = p
	}

	user, err := requireString(config, "user")
	if err != nil {
		return err
	}

	hostKeyCheck := true
	if v, ok := config["host_key_check"]; ok {
		b, isBool := v.(bool)
		if !isBool {
			return fmt.Errorf("ssh: host_key_check must be a bool")
		}
		hostKeyCheck = b
	}

	authMethods, err := buildAuthMethods(config)
	if err != nil {
		return err
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("ssh: no authentication method provided (need key_file, key, or password)")
	}

	clientConfig := &cryptossh.ClientConfig{
		User: user,
		Auth: authMethods,
	}

	if hostKeyCheck {
		clientConfig.HostKeyCallback = cryptossh.FixedHostKey(nil)
	} else {
		clientConfig.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()
	}

	c.config = clientConfig
	c.addr = net.JoinHostPort(host, strconv.Itoa(port))

	// Check for context cancellation before dialing.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	client, err := dialSSH(ctx, "tcp", c.addr, c.config)
	if err != nil {
		return fmt.Errorf("ssh: dial %s: %w", c.addr, err)
	}
	c.client = client

	return nil
}

// Execute runs a single test step over the SSH connection.
//
// Supported actions:
//   - "exec": run a remote command
//   - "tunnel": create a local-to-remote port forward
func (c *SSHConnector) Execute(ctx context.Context, step connector.Step) (*connector.Result, error) {
	if c.client == nil {
		return nil, fmt.Errorf("ssh: not connected (call Setup first)")
	}

	switch step.Action {
	case "exec":
		return c.executeCommand(ctx, step)
	case "tunnel":
		return c.executeTunnel(ctx, step)
	default:
		return nil, fmt.Errorf("ssh: unsupported action %q (supported: exec, tunnel)", step.Action)
	}
}

// Teardown closes the SSH connection and cleans up all resources.
func (c *SSHConnector) Teardown(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Signal tunnels to stop.
	select {
	case <-c.closed:
		// Already closed.
	default:
		close(c.closed)
	}

	// Close all tunnel listeners.
	for _, l := range c.listeners {
		l.Close()
	}
	c.listeners = nil

	// Wait for tunnel goroutines to finish.
	c.tunnelWg.Wait()

	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

// dialSSH is a variable so tests can replace it.
var dialSSH = defaultDialSSH

func defaultDialSSH(_ context.Context, network, addr string, config *cryptossh.ClientConfig) (*cryptossh.Client, error) {
	return cryptossh.Dial(network, addr, config)
}

// requireString extracts a required string from the config map.
func requireString(config map[string]any, key string) (string, error) {
	v, ok := config[key]
	if !ok {
		return "", fmt.Errorf("ssh: missing required config key %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("ssh: config key %q must be a string", key)
	}
	if s == "" {
		return "", fmt.Errorf("ssh: config key %q must not be empty", key)
	}
	return s, nil
}

// toInt converts various numeric types to int.
func toInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case float64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}
