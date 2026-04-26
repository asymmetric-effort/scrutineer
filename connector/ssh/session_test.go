package ssh

import (
	"testing"

	cryptossh "golang.org/x/crypto/ssh"
)

func TestNewSession_NilClient(t *testing.T) {
	c := New()
	_, err := c.newSession()
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestCloseSession_Nil(t *testing.T) {
	// Should not panic.
	closeSession(nil)
}

func TestNewSession_Integration(t *testing.T) {
	handler := func(ch cryptossh.Channel, req *cryptossh.Request) {
		if req.WantReply {
			req.Reply(true, nil)
		}
		ch.Close()
	}

	srv := newMockSSHServer(t, handler)
	defer srv.close()

	c := setupMockConnector(t, srv)
	defer c.Teardown(nil)

	session, err := c.newSession()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	closeSession(session)
}
