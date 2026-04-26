package ssh

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
	cryptossh "golang.org/x/crypto/ssh"
)

func TestRequireParam(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		key     string
		want    string
		wantErr bool
	}{
		{"valid", map[string]any{"command": "ls"}, "command", "ls", false},
		{"missing", map[string]any{}, "command", "", true},
		{"not_string", map[string]any{"command": 42}, "command", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := requireParam(tt.params, tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"exit_error", &cryptossh.ExitError{Waitmsg: cryptossh.Waitmsg{}}, 0},
		{"generic_error", fmt.Errorf("something"), -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractExitCode(tt.err)
			if got != tt.want {
				t.Fatalf("extractExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExecuteCommand_MissingCommand(t *testing.T) {
	c := New()
	c.client = &cryptossh.Client{}

	_, err := c.executeCommand(context.Background(), connector.Step{
		Parameters: map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error for missing command parameter")
	}
}

func TestExecuteCommand_InvalidStdin(t *testing.T) {
	// We need a server to test stdin validation since it happens after session creation.
	handler := func(ch cryptossh.Channel, req *cryptossh.Request) {
		if req.Type == "exec" {
			if req.WantReply {
				req.Reply(true, nil)
			}
			ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
			ch.Close()
		}
	}

	srv := newMockSSHServer(t, handler)
	defer srv.close()

	c := setupMockConnector(t, srv)
	defer c.Teardown(context.Background())

	_, err := c.executeCommand(context.Background(), connector.Step{
		Parameters: map[string]any{
			"command": "echo hi",
			"stdin":   123, // not a string
		},
	})
	if err == nil {
		t.Fatal("expected error for non-string stdin")
	}
}

func TestExecuteCommand_Integration(t *testing.T) {
	handler := func(ch cryptossh.Channel, req *cryptossh.Request) {
		if req.Type == "exec" {
			// Extract command from the request payload.
			// Payload format: uint32 length + command string
			if len(req.Payload) > 4 {
				cmdLen := binary.BigEndian.Uint32(req.Payload[:4])
				cmd := string(req.Payload[4 : 4+cmdLen])

				if req.WantReply {
					req.Reply(true, nil)
				}

				switch cmd {
				case "echo hello":
					ch.Write([]byte("hello\n"))
				case "fail":
					ch.Stderr().Write([]byte("error occurred\n"))
					// Send exit status 1
					exitStatus := make([]byte, 4)
					binary.BigEndian.PutUint32(exitStatus, 1)
					ch.SendRequest("exit-status", false, exitStatus)
					ch.Close()
					return
				case "cat":
					// Read stdin and echo it.
					buf := make([]byte, 1024)
					n, _ := ch.Read(buf)
					ch.Write(buf[:n])
				}
				// Send exit status 0
				exitStatus := make([]byte, 4)
				binary.BigEndian.PutUint32(exitStatus, 0)
				ch.SendRequest("exit-status", false, exitStatus)
				ch.Close()
			} else {
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}
	}

	srv := newMockSSHServer(t, handler)
	defer srv.close()

	c := setupMockConnector(t, srv)
	defer c.Teardown(context.Background())

	t.Run("echo", func(t *testing.T) {
		result, err := c.Execute(context.Background(), connector.Step{
			Action:     "exec",
			Parameters: map[string]any{"command": "echo hello"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		stdout := result.Data["stdout"].(string)
		if stdout != "hello\n" {
			t.Fatalf("stdout = %q, want %q", stdout, "hello\n")
		}
		exitCode := result.Data["exit_code"].(int)
		if exitCode != 0 {
			t.Fatalf("exit_code = %d, want 0", exitCode)
		}
		if result.Meta["connector"] != "ssh" {
			t.Fatalf("meta connector = %q, want %q", result.Meta["connector"], "ssh")
		}
		if result.Meta["action"] != "exec" {
			t.Fatalf("meta action = %q, want %q", result.Meta["action"], "exec")
		}
		if result.Elapsed <= 0 {
			t.Fatal("elapsed should be positive")
		}
	})

	t.Run("fail_command", func(t *testing.T) {
		result, err := c.Execute(context.Background(), connector.Step{
			Action:     "exec",
			Parameters: map[string]any{"command": "fail"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		exitCode := result.Data["exit_code"].(int)
		if exitCode != 1 {
			t.Fatalf("exit_code = %d, want 1", exitCode)
		}
		stderr := result.Data["stderr"].(string)
		if stderr != "error occurred\n" {
			t.Fatalf("stderr = %q, want %q", stderr, "error occurred\n")
		}
	})

	t.Run("with_stdin", func(t *testing.T) {
		result, err := c.Execute(context.Background(), connector.Step{
			Action: "exec",
			Parameters: map[string]any{
				"command": "cat",
				"stdin":   "input data",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		stdout := result.Data["stdout"].(string)
		if stdout != "input data" {
			t.Fatalf("stdout = %q, want %q", stdout, "input data")
		}
	})
}

func TestExecuteCommand_ContextCancel(t *testing.T) {
	handler := func(ch cryptossh.Channel, req *cryptossh.Request) {
		if req.Type == "exec" {
			if req.WantReply {
				req.Reply(true, nil)
			}
			// Simulate long-running command: don't close channel.
			time.Sleep(5 * time.Second)
			ch.Close()
		}
	}

	srv := newMockSSHServer(t, handler)
	defer srv.close()

	c := setupMockConnector(t, srv)
	defer c.Teardown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := c.Execute(ctx, connector.Step{
		Action:     "exec",
		Parameters: map[string]any{"command": "sleep 100"},
	})
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestExecuteCommand_SessionError(t *testing.T) {
	c := New()
	// nil client triggers session creation error
	_, err := c.executeCommand(context.Background(), connector.Step{
		Parameters: map[string]any{"command": "ls"},
	})
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

// setupMockConnector creates an SSHConnector connected to the mock server.
func setupMockConnector(t *testing.T, srv *mockSSHServer) *SSHConnector {
	t.Helper()

	host, portStr, err := net.SplitHostPort(srv.addr())
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(portStr)

	origDial := dialSSH
	t.Cleanup(func() { dialSSH = origDial })

	// Use the real dial since we have a real mock server.
	dialSSH = defaultDialSSH

	c := New()
	err = c.Setup(context.Background(), map[string]any{
		"host":           host,
		"port":           port,
		"user":           "testuser",
		"password":       "testpass",
		"host_key_check": false,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	return c
}
