package ssh

import (
	"fmt"

	cryptossh "golang.org/x/crypto/ssh"
)

// newSession creates a new SSH session on the active client connection.
// Each command execution should use its own session.
func (c *SSHConnector) newSession() (*cryptossh.Session, error) {
	if c.client == nil {
		return nil, fmt.Errorf("ssh: no active connection")
	}
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ssh: create session: %w", err)
	}
	return session, nil
}

// closeSession safely closes an SSH session, ignoring errors from
// already-closed sessions.
func closeSession(session *cryptossh.Session) {
	if session != nil {
		session.Close()
	}
}
