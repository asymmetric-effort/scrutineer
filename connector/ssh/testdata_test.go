package ssh

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"

	cryptossh "golang.org/x/crypto/ssh"
)

// testRSAKey is a PEM-encoded ECDSA private key used for testing.
// Generated fresh at init time.
var testRSAKey []byte

func init() {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("generate test key: %v", err))
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(fmt.Sprintf("marshal test key: %v", err))
	}
	testRSAKey = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	})
}

// mockSSHServer runs a minimal SSH server for integration testing.
// It accepts connections, performs authentication, and handles "exec" requests.
type mockSSHServer struct {
	listener net.Listener
	config   *cryptossh.ServerConfig
	wg       sync.WaitGroup
	handler  func(ch cryptossh.Channel, req *cryptossh.Request)
	mu       sync.Mutex
	closed   bool
}

func newMockSSHServer(t *testing.T, handler func(ch cryptossh.Channel, req *cryptossh.Request)) *mockSSHServer {
	t.Helper()

	// Generate server host key.
	hostKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := cryptossh.NewSignerFromKey(hostKey)
	if err != nil {
		t.Fatal(err)
	}

	config := &cryptossh.ServerConfig{
		PasswordCallback: func(c cryptossh.ConnMetadata, pass []byte) (*cryptossh.Permissions, error) {
			if c.User() == "testuser" && string(pass) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
		PublicKeyCallback: func(c cryptossh.ConnMetadata, pubKey cryptossh.PublicKey) (*cryptossh.Permissions, error) {
			// Accept any key for testing.
			return nil, nil
		},
	}
	config.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	s := &mockSSHServer{
		listener: ln,
		config:   config,
		handler:  handler,
	}

	s.wg.Add(1)
	go s.serve(t)

	return s
}

func (s *mockSSHServer) addr() string {
	return s.listener.Addr().String()
}

func (s *mockSSHServer) serve(t *testing.T) {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return
			}
			return
		}
		s.wg.Add(1)
		go s.handleConn(t, conn)
	}
}

func (s *mockSSHServer) handleConn(t *testing.T, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	sshConn, chans, reqs, err := cryptossh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	go cryptossh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() == "session" {
			ch, requests, err := newCh.Accept()
			if err != nil {
				return
			}
			go s.handleSession(ch, requests)
		} else if newCh.ChannelType() == "direct-tcpip" {
			ch, _, err := newCh.Accept()
			if err != nil {
				return
			}
			go s.handleDirectTCPIP(ch)
		} else {
			newCh.Reject(cryptossh.UnknownChannelType, "unsupported")
		}
	}
}

func (s *mockSSHServer) handleSession(ch cryptossh.Channel, reqs <-chan *cryptossh.Request) {
	defer ch.Close()

	for req := range reqs {
		if s.handler != nil {
			s.handler(ch, req)
		} else {
			if req.WantReply {
				req.Reply(true, nil)
			}
		}
	}
}

func (s *mockSSHServer) handleDirectTCPIP(ch cryptossh.Channel) {
	defer ch.Close()
	// Echo back any data received.
	io.Copy(ch, ch)
}

func (s *mockSSHServer) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.listener.Close()
	s.wg.Wait()
}
