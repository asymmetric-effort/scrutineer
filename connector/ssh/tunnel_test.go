package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
	cryptossh "golang.org/x/crypto/ssh"
)

func TestRequireIntParam(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		key     string
		want    int
		wantErr bool
	}{
		{"valid_int", map[string]any{"port": 8080}, "port", 8080, false},
		{"valid_float", map[string]any{"port": float64(8080)}, "port", 8080, false},
		{"valid_int64", map[string]any{"port": int64(8080)}, "port", 8080, false},
		{"missing", map[string]any{}, "port", 0, true},
		{"not_number", map[string]any{"port": "abc"}, "port", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := requireIntParam(tt.params, tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExecuteTunnel_MissingParams(t *testing.T) {
	handler := func(ch cryptossh.Channel, req *cryptossh.Request) {
		if req.WantReply {
			req.Reply(true, nil)
		}
		ch.Close()
	}
	srv := newMockSSHServer(t, handler)
	defer srv.close()

	c := setupMockConnector(t, srv)
	defer c.Teardown(context.Background())

	tests := []struct {
		name   string
		params map[string]any
	}{
		{"missing_local_port", map[string]any{"remote_host": "localhost", "remote_port": 80}},
		{"missing_remote_host", map[string]any{"local_port": 0, "remote_port": 80}},
		{"missing_remote_port", map[string]any{"local_port": 0, "remote_host": "localhost"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.executeTunnel(context.Background(), connector.Step{
				Parameters: tt.params,
			})
			if err == nil {
				t.Fatal("expected error for missing parameter")
			}
		})
	}
}

func TestExecuteTunnel_InvalidParamTypes(t *testing.T) {
	handler := func(ch cryptossh.Channel, req *cryptossh.Request) {
		if req.WantReply {
			req.Reply(true, nil)
		}
		ch.Close()
	}
	srv := newMockSSHServer(t, handler)
	defer srv.close()

	c := setupMockConnector(t, srv)
	defer c.Teardown(context.Background())

	tests := []struct {
		name   string
		params map[string]any
	}{
		{"local_port_string", map[string]any{"local_port": "abc", "remote_host": "localhost", "remote_port": 80}},
		{"remote_host_int", map[string]any{"local_port": 0, "remote_host": 123, "remote_port": 80}},
		{"remote_port_string", map[string]any{"local_port": 0, "remote_host": "localhost", "remote_port": "abc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.executeTunnel(context.Background(), connector.Step{
				Parameters: tt.params,
			})
			if err == nil {
				t.Fatal("expected error for invalid param type")
			}
		})
	}
}

func TestExecuteTunnel_Integration(t *testing.T) {
	// Start an echo TCP server that the tunnel will forward to.
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoLn.Close()

	go func() {
		for {
			conn, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()

	_, echoPortStr, _ := net.SplitHostPort(echoLn.Addr().String())

	handler := func(ch cryptossh.Channel, req *cryptossh.Request) {
		if req.WantReply {
			req.Reply(true, nil)
		}
		ch.Close()
	}
	srv := newMockSSHServer(t, handler)
	defer srv.close()

	c := setupMockConnector(t, srv)
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "tunnel",
		Parameters: map[string]any{
			"local_port":  0, // auto-assign
			"remote_host": "127.0.0.1",
			"remote_port": mustAtoi(echoPortStr),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	localAddr, ok := result.Data["local_addr"].(string)
	if !ok || localAddr == "" {
		t.Fatal("expected non-empty local_addr in result")
	}

	if result.Meta["connector"] != "ssh" {
		t.Fatalf("meta connector = %q, want %q", result.Meta["connector"], "ssh")
	}
	if result.Meta["action"] != "tunnel" {
		t.Fatalf("meta action = %q, want %q", result.Meta["action"], "tunnel")
	}
	if result.Elapsed < 0 {
		t.Fatal("elapsed should be non-negative")
	}

	// Try connecting through the tunnel. The SSH mock server handles
	// direct-tcpip channels with echo behavior.
	conn, err := net.DialTimeout("tcp", localAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial tunnel: %v", err)
	}
	defer conn.Close()

	testData := []byte("hello through tunnel")
	_, err = conn.Write(testData)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// Set a read deadline so we don't hang.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, len(testData))
	n, err := io.ReadFull(conn, buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf[:n]) != string(testData) {
		t.Fatalf("got %q, want %q", string(buf[:n]), string(testData))
	}
}

func TestExecuteTunnel_ListenError(t *testing.T) {
	handler := func(ch cryptossh.Channel, req *cryptossh.Request) {
		if req.WantReply {
			req.Reply(true, nil)
		}
		ch.Close()
	}
	srv := newMockSSHServer(t, handler)
	defer srv.close()

	c := setupMockConnector(t, srv)
	defer c.Teardown(context.Background())

	// Bind a port first so the tunnel can't use it.
	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Close()
	_, blockerPort, _ := net.SplitHostPort(blocker.Addr().String())

	_, err = c.executeTunnel(context.Background(), connector.Step{
		Parameters: map[string]any{
			"local_port":  mustAtoi(blockerPort),
			"remote_host": "127.0.0.1",
			"remote_port": 80,
		},
	})
	if err == nil {
		t.Fatal("expected error for port already in use")
	}
}

func TestForwardTunnelConn_NilClient(t *testing.T) {
	c := New()
	// Create a pipe to simulate a connection.
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Should not panic, just return.
	c.forwardTunnelConn(server, "127.0.0.1:80")
}

func TestAcceptTunnelConns_ClosedListener(t *testing.T) {
	c := New()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ln.Close() // Close immediately.

	c.tunnelWg.Add(1)
	c.acceptTunnelConns(ln, "127.0.0.1:80")
	// Should return without hanging.
}

func mustAtoi(s string) int {
	n := 0
	fmt.Sscanf(s, "%d", &n)
	return n
}
