package ssh

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/scrutineer/scrutineer/core/connector"
	cryptossh "golang.org/x/crypto/ssh"
)

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if c.closed == nil {
		t.Fatal("closed channel not initialized")
	}
}

func TestName(t *testing.T) {
	c := New()
	if got := c.Name(); got != "ssh" {
		t.Fatalf("Name() = %q, want %q", got, "ssh")
	}
}

func TestRequireString(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]any
		key     string
		want    string
		wantErr bool
	}{
		{"valid", map[string]any{"host": "example.com"}, "host", "example.com", false},
		{"missing", map[string]any{}, "host", "", true},
		{"not_string", map[string]any{"host": 123}, "host", "", true},
		{"empty", map[string]any{"host": ""}, "host", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := requireString(tt.config, tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("requireString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("requireString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name    string
		val     any
		want    int
		wantErr bool
	}{
		{"int", 42, 42, false},
		{"int64", int64(42), 42, false},
		{"float64", float64(42), 42, false},
		{"string", "42", 0, true},
		{"bool", true, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toInt(tt.val)
			if (err != nil) != tt.wantErr {
				t.Fatalf("toInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("toInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSetup_MissingHost(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"user":     "test",
		"password": "pass",
	})
	if err == nil {
		t.Fatal("expected error for missing host")
	}
}

func TestSetup_MissingUser(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":     "example.com",
		"password": "pass",
	})
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestSetup_InvalidPort(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":     "example.com",
		"user":     "test",
		"port":     "not-a-number",
		"password": "pass",
	})
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestSetup_NoAuth(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":           "example.com",
		"user":           "test",
		"host_key_check": false,
	})
	if err == nil {
		t.Fatal("expected error for no auth method")
	}
}

func TestSetup_InvalidHostKeyCheck(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":           "example.com",
		"user":           "test",
		"password":       "pass",
		"host_key_check": "not-bool",
	})
	if err == nil {
		t.Fatal("expected error for invalid host_key_check")
	}
}

func TestSetup_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := New()
	err := c.Setup(ctx, map[string]any{
		"host":           "example.com",
		"user":           "test",
		"password":       "pass",
		"host_key_check": false,
	})
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestSetup_DialError(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()

	dialSSH = func(_ context.Context, _, _ string, _ *cryptossh.ClientConfig) (*cryptossh.Client, error) {
		return nil, fmt.Errorf("connection refused")
	}

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":           "example.com",
		"user":           "test",
		"password":       "pass",
		"host_key_check": false,
	})
	if err == nil {
		t.Fatal("expected error for dial failure")
	}
}

func TestSetup_Success(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()

	dialSSH = func(_ context.Context, _, _ string, _ *cryptossh.ClientConfig) (*cryptossh.Client, error) {
		return &cryptossh.Client{}, nil
	}

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":           "example.com",
		"port":           22,
		"user":           "test",
		"password":       "pass",
		"host_key_check": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.addr != "example.com:22" {
		t.Fatalf("addr = %q, want %q", c.addr, "example.com:22")
	}
}

func TestSetup_HostKeyCheckEnabled(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()

	dialSSH = func(_ context.Context, _, _ string, _ *cryptossh.ClientConfig) (*cryptossh.Client, error) {
		return nil, fmt.Errorf("host key mismatch")
	}

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":           "example.com",
		"user":           "test",
		"password":       "pass",
		"host_key_check": true,
	})
	if err == nil {
		t.Fatal("expected error with host key check enabled")
	}
}

func TestSetup_DefaultPort(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()

	dialSSH = func(_ context.Context, _, addr string, _ *cryptossh.ClientConfig) (*cryptossh.Client, error) {
		if addr != "example.com:22" {
			t.Fatalf("addr = %q, want default port 22", addr)
		}
		return &cryptossh.Client{}, nil
	}

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":           "example.com",
		"user":           "test",
		"password":       "pass",
		"host_key_check": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetup_CustomPort(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()

	dialSSH = func(_ context.Context, _, addr string, _ *cryptossh.ClientConfig) (*cryptossh.Client, error) {
		if addr != "example.com:2222" {
			t.Fatalf("addr = %q, want custom port 2222", addr)
		}
		return &cryptossh.Client{}, nil
	}

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":           "example.com",
		"port":           2222,
		"user":           "test",
		"password":       "pass",
		"host_key_check": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetup_PortFloat64(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()

	dialSSH = func(_ context.Context, _, addr string, _ *cryptossh.ClientConfig) (*cryptossh.Client, error) {
		if addr != "example.com:2222" {
			t.Fatalf("addr = %q, want port from float64", addr)
		}
		return &cryptossh.Client{}, nil
	}

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":           "example.com",
		"port":           float64(2222),
		"user":           "test",
		"password":       "pass",
		"host_key_check": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetup_PortInt64(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()

	dialSSH = func(_ context.Context, _, addr string, _ *cryptossh.ClientConfig) (*cryptossh.Client, error) {
		if addr != "example.com:2222" {
			t.Fatalf("addr = %q, want port from int64", addr)
		}
		return &cryptossh.Client{}, nil
	}

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":           "example.com",
		"port":           int64(2222),
		"user":           "test",
		"password":       "pass",
		"host_key_check": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetup_WithKeyAndPassword(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()

	dialSSH = func(_ context.Context, _, _ string, config *cryptossh.ClientConfig) (*cryptossh.Client, error) {
		if len(config.Auth) != 2 {
			t.Fatalf("expected 2 auth methods, got %d", len(config.Auth))
		}
		return &cryptossh.Client{}, nil
	}

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"host":           "example.com",
		"user":           "test",
		"key":            string(testRSAKey),
		"password":       "pass",
		"host_key_check": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_NotConnected(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{Action: "exec"})
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestExecute_UnsupportedAction(t *testing.T) {
	c := New()
	c.client = &cryptossh.Client{}
	_, err := c.Execute(context.Background(), connector.Step{Action: "unknown"})
	if err == nil {
		t.Fatal("expected error for unsupported action")
	}
}

func TestTeardown_NilClient(t *testing.T) {
	c := New()
	err := c.Teardown(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTeardown_DoubleTeardown(t *testing.T) {
	c := New()
	_ = c.Teardown(context.Background())
	err := c.Teardown(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on double teardown: %v", err)
	}
}

func TestTeardown_ClosesListeners(t *testing.T) {
	c := New()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	c.listeners = append(c.listeners, ln)

	err = c.Teardown(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the listener is closed.
	_, err = ln.Accept()
	if err == nil {
		t.Fatal("expected error from closed listener")
	}
}

func TestConnectorInterface(t *testing.T) {
	var _ connector.Connector = (*SSHConnector)(nil)
}
