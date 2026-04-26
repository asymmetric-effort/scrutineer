package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
)

// executeTunnel creates a local TCP listener that forwards connections
// through the SSH tunnel to a remote host:port.
//
// Parameters:
//   - "local_port" (int, required): local port to listen on (0 for auto)
//   - "remote_host" (string, required): remote host to connect to
//   - "remote_port" (int, required): remote port to connect to
//
// Result data:
//   - "local_addr" (string): the local address the tunnel is listening on
//
// The tunnel stays open until Teardown is called.
func (c *SSHConnector) executeTunnel(ctx context.Context, step connector.Step) (*connector.Result, error) {
	localPort, err := requireIntParam(step.Parameters, "local_port")
	if err != nil {
		return nil, err
	}

	remoteHost, err := requireParam(step.Parameters, "remote_host")
	if err != nil {
		return nil, err
	}

	remotePort, err := requireIntParam(step.Parameters, "remote_port")
	if err != nil {
		return nil, err
	}

	remoteAddr := net.JoinHostPort(remoteHost, strconv.Itoa(remotePort))
	localAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(localPort))

	start := time.Now()

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("ssh: listen on %s: %w", localAddr, err)
	}

	// Record the listener for cleanup during Teardown.
	c.mu.Lock()
	c.listeners = append(c.listeners, listener)
	c.mu.Unlock()

	// Start accepting connections in the background.
	c.tunnelWg.Add(1)
	go c.acceptTunnelConns(listener, remoteAddr)

	elapsed := time.Since(start)

	return &connector.Result{
		Data: map[string]any{
			"local_addr": listener.Addr().String(),
		},
		Elapsed: elapsed,
		Meta: map[string]string{
			"connector":   "ssh",
			"action":      "tunnel",
			"remote_addr": remoteAddr,
		},
	}, nil
}

// acceptTunnelConns accepts incoming connections on the local listener and
// forwards them through the SSH tunnel to the remote address.
func (c *SSHConnector) acceptTunnelConns(listener net.Listener, remoteAddr string) {
	defer c.tunnelWg.Done()

	for {
		localConn, err := listener.Accept()
		if err != nil {
			// Check if we've been closed.
			select {
			case <-c.closed:
				return
			default:
			}
			// Transient error; continue or break depending on listener state.
			return
		}

		go c.forwardTunnelConn(localConn, remoteAddr)
	}
}

// forwardTunnelConn forwards a single connection through the SSH tunnel.
func (c *SSHConnector) forwardTunnelConn(localConn net.Conn, remoteAddr string) {
	defer localConn.Close()

	if c.client == nil {
		return
	}

	remoteConn, err := c.client.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	// Wait for one direction to finish, then return (defers close both).
	select {
	case <-done:
	case <-c.closed:
	}
}

// requireIntParam extracts a required integer parameter from the step parameters.
func requireIntParam(params map[string]any, key string) (int, error) {
	v, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("ssh: missing required parameter %q", key)
	}
	n, err := toInt(v)
	if err != nil {
		return 0, fmt.Errorf("ssh: parameter %q: %w", key, err)
	}
	return n, nil
}
