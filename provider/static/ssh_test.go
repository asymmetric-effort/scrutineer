package static

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestLoadEd25519KeyValid(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}

	pemBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	pemBytes := pem.EncodeToMemory(pemBlock)
	if err := os.WriteFile(keyPath, pemBytes, 0600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	signer, err := loadEd25519Key(keyPath)
	if err != nil {
		t.Fatalf("loadEd25519Key returned error: %v", err)
	}
	if signer == nil {
		t.Fatal("loadEd25519Key returned nil signer")
	}
	if signer.PublicKey().Type() != "ssh-ed25519" {
		t.Errorf("key type = %q, want ssh-ed25519", signer.PublicKey().Type())
	}
}

func TestLoadEd25519KeyRSA(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	pemBlock, err := ssh.MarshalPrivateKey(rsaKey, "")
	if err != nil {
		t.Fatalf("marshal RSA key: %v", err)
	}

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_rsa")
	pemBytes := pem.EncodeToMemory(pemBlock)
	if err := os.WriteFile(keyPath, pemBytes, 0600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	_, err = loadEd25519Key(keyPath)
	if err == nil {
		t.Fatal("expected error for RSA key, got nil")
	}
	if !strings.Contains(err.Error(), "only ed25519") {
		t.Errorf("error = %q, want it to contain 'only ed25519'", err.Error())
	}
}

func TestLoadEd25519KeyMissing(t *testing.T) {
	_, err := loadEd25519Key("/nonexistent/path/to/key")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadEd25519KeyInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "garbage_key")
	if err := os.WriteFile(keyPath, []byte("this is not a valid key"), 0600); err != nil {
		t.Fatalf("write garbage file: %v", err)
	}

	_, err := loadEd25519Key(keyPath)
	if err == nil {
		t.Fatal("expected error for invalid key format, got nil")
	}
	if !strings.Contains(err.Error(), "parse key") {
		t.Errorf("error = %q, want it to contain 'parse key'", err.Error())
	}
}

func TestIsEd25519Key(t *testing.T) {
	// Test with ed25519 signer.
	_, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	edSigner, err := ssh.NewSignerFromKey(edPriv)
	if err != nil {
		t.Fatalf("new signer from ed25519: %v", err)
	}
	if !isEd25519Key(edSigner) {
		t.Error("isEd25519Key returned false for ed25519 key")
	}

	// Test with RSA signer.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	rsaSigner, err := ssh.NewSignerFromKey(rsaKey)
	if err != nil {
		t.Fatalf("new signer from RSA: %v", err)
	}
	if isEd25519Key(rsaSigner) {
		t.Error("isEd25519Key returned true for RSA key")
	}
}
