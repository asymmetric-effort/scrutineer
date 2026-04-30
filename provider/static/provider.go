// Package static implements a fleet provider that executes tests on
// pre-existing hosts via SSH with ed25519 key-based authentication.
package static

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/scrutineer/scrutineer/core/fleet"

	"golang.org/x/crypto/ssh"
)

// Config holds parsed configuration for the static provider.
type Config struct {
	SSH    SSHConfig
	Binary string
	Nodes  []string
}

// SSHConfig holds SSH connection parameters.
type SSHConfig struct {
	User    string
	KeyFile string
	Port    int
}

// Provider implements fleet.Provider using SSH to static hosts.
type Provider struct {
	config Config
	conns  map[string]*ssh.Client
	mu     sync.Mutex
}

// New creates a new static fleet provider.
func New() *Provider {
	return &Provider{
		conns: make(map[string]*ssh.Client),
	}
}

// Name returns "static".
func (p *Provider) Name() string { return "static" }

// Setup parses the provider-specific config map into a Config struct.
func (p *Provider) Setup(_ context.Context, config map[string]any) error {
	if config == nil {
		return fmt.Errorf("static: config is required")
	}

	// Parse SSH sub-config.
	if sshRaw, ok := config["ssh"].(map[string]any); ok {
		if v, ok := sshRaw["user"].(string); ok {
			p.config.SSH.User = v
		}
		if v, ok := sshRaw["key_file"].(string); ok {
			p.config.SSH.KeyFile = v
		}
		if v, ok := sshRaw["port"].(int); ok {
			p.config.SSH.Port = v
		}
	}

	if p.config.SSH.Port == 0 {
		p.config.SSH.Port = 22
	}

	if v, ok := config["binary"].(string); ok {
		p.config.Binary = v
	}

	if v, ok := config["nodes"].([]any); ok {
		for _, n := range v {
			if s, ok := n.(string); ok {
				p.config.Nodes = append(p.config.Nodes, s)
			}
		}
	}

	if len(p.config.Nodes) == 0 {
		return fmt.Errorf("static: at least one node is required")
	}

	if p.config.SSH.KeyFile == "" {
		return fmt.Errorf("static: ssh key_file is required")
	}

	if p.config.SSH.User == "" {
		return fmt.Errorf("static: ssh user is required")
	}

	return nil
}

// Acquire returns Host entries for the configured static nodes.
// n must be <= len(Nodes).
func (p *Provider) Acquire(_ context.Context, n int) ([]fleet.Host, error) {
	if n > len(p.config.Nodes) {
		return nil, fmt.Errorf("static: requested %d hosts but only %d configured", n, len(p.config.Nodes))
	}

	hosts := make([]fleet.Host, n)
	for i := 0; i < n; i++ {
		hosts[i] = fleet.Host{
			ID:       fmt.Sprintf("static-%d", i),
			Address:  p.config.Nodes[i],
			Provider: "static",
			BornAt:   time.Now(),
		}
	}
	return hosts, nil
}

// Push distributes artifacts to a host via SCP.
func (p *Provider) Push(ctx context.Context, host fleet.Host, artifacts []string) error {
	client, err := p.getOrDial(host.Address)
	if err != nil {
		return fmt.Errorf("static: push to %s: %w", host.Address, err)
	}

	for _, artifact := range artifacts {
		if err := scpFile(ctx, client, artifact, artifact); err != nil {
			return fmt.Errorf("static: push %s to %s: %w", artifact, host.Address, err)
		}
	}
	return nil
}

// Execute runs a command on the host via SSH.
func (p *Provider) Execute(ctx context.Context, host fleet.Host, cmd string) (*fleet.ExecResult, error) {
	client, err := p.getOrDial(host.Address)
	if err != nil {
		return nil, fmt.Errorf("static: connect to %s: %w", host.Address, err)
	}

	start := time.Now()
	stdout, stderr, exitCode, err := runCommand(ctx, client, cmd)
	elapsed := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("static: execute on %s: %w", host.Address, err)
	}

	return &fleet.ExecResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
		Elapsed:  elapsed,
	}, nil
}

// Release is a no-op for static nodes — they persist.
func (p *Provider) Release(_ context.Context, _ []fleet.Host) error {
	return nil
}

// Teardown closes all SSH connections.
func (p *Provider) Teardown(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for addr, client := range p.conns {
		if err := client.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("static: close %s: %w", addr, err)
		}
		delete(p.conns, addr)
	}
	return firstErr
}

// getOrDial returns an existing SSH connection or creates a new one.
func (p *Provider) getOrDial(address string) (*ssh.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, ok := p.conns[address]; ok {
		return client, nil
	}

	client, err := dialSSH(address, p.config.SSH)
	if err != nil {
		return nil, err
	}
	p.conns[address] = client
	return client, nil
}
