package static

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

// dialSSH is the package-level SSH dial function, replaceable for testing.
var dialSSH = defaultDialSSH

// runCommand is the package-level command execution function, replaceable for testing.
var runCommand = defaultRunCommand

// scpFile is the package-level SCP function, replaceable for testing.
var scpFile = defaultScpFile

func defaultDialSSH(address string, cfg SSHConfig) (*ssh.Client, error) {
	key, err := loadEd25519Key(cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("ssh: load key: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", address, cfg.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh: dial %s: %w", addr, err)
	}
	return client, nil
}

// loadEd25519Key loads an ed25519 private key from a file.
// Returns an error if the key is not ed25519.
func loadEd25519Key(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key %s: %w", path, err)
	}

	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse key %s: %w", path, err)
	}

	// Verify the key is ed25519.
	pubKey := signer.PublicKey()
	if _, ok := pubKey.(ssh.CryptoPublicKey).CryptoPublicKey().(ed25519.PublicKey); !ok {
		return nil, fmt.Errorf("key %s: only ed25519 keys are supported, got %s", path, pubKey.Type())
	}

	return signer, nil
}

func defaultRunCommand(_ context.Context, client *ssh.Client, cmd string) (stdout, stderr string, exitCode int, err error) {
	session, err := client.NewSession()
	if err != nil {
		return "", "", -1, fmt.Errorf("ssh: new session: %w", err)
	}
	defer session.Close()

	var outBuf, errBuf bytes.Buffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf

	exitCode = 0
	if err := session.Run(cmd); err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		} else {
			return "", "", -1, err
		}
	}

	return outBuf.String(), errBuf.String(), exitCode, nil
}

func defaultScpFile(_ context.Context, client *ssh.Client, localPath, remotePath string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("ssh: new session: %w", err)
	}
	defer session.Close()

	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("scp: read %s: %w", localPath, err)
	}

	// Use cat-based SCP approach.
	session.Stdin = bytes.NewReader(data)
	cmd := fmt.Sprintf("cat > %s", remotePath)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("scp: write %s: %w", remotePath, err)
	}

	return nil
}

// isEd25519Key checks if a parsed SSH key is ed25519.
// Exported for testing.
func isEd25519Key(key ssh.Signer) bool {
	pubKey := key.PublicKey()
	cryptoPub, ok := pubKey.(ssh.CryptoPublicKey)
	if !ok {
		return false
	}
	_, isEd := cryptoPub.CryptoPublicKey().(ed25519.PublicKey)
	return isEd
}

// For testing: net.Listener-based mock SSH server helper.
func listenOnFreePort() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}
