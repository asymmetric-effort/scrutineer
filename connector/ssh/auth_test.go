package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildAuthMethods_Password(t *testing.T) {
	methods, err := buildAuthMethods(map[string]any{
		"password": "secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

func TestBuildAuthMethods_PasswordNotString(t *testing.T) {
	_, err := buildAuthMethods(map[string]any{
		"password": 123,
	})
	if err == nil {
		t.Fatal("expected error for non-string password")
	}
}

func TestBuildAuthMethods_KeyPEM(t *testing.T) {
	methods, err := buildAuthMethods(map[string]any{
		"key": string(testRSAKey),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

func TestBuildAuthMethods_KeyNotString(t *testing.T) {
	_, err := buildAuthMethods(map[string]any{
		"key": 123,
	})
	if err == nil {
		t.Fatal("expected error for non-string key")
	}
}

func TestBuildAuthMethods_KeyInvalidPEM(t *testing.T) {
	_, err := buildAuthMethods(map[string]any{
		"key": "not-a-valid-pem",
	})
	if err == nil {
		t.Fatal("expected error for invalid PEM key")
	}
}

func TestBuildAuthMethods_KeyFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_test")
	if err := os.WriteFile(keyPath, testRSAKey, 0600); err != nil {
		t.Fatal(err)
	}

	methods, err := buildAuthMethods(map[string]any{
		"key_file": keyPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

func TestBuildAuthMethods_KeyFileNotString(t *testing.T) {
	_, err := buildAuthMethods(map[string]any{
		"key_file": 123,
	})
	if err == nil {
		t.Fatal("expected error for non-string key_file")
	}
}

func TestBuildAuthMethods_KeyFileNotFound(t *testing.T) {
	_, err := buildAuthMethods(map[string]any{
		"key_file": "/nonexistent/path/key",
	})
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
}

func TestBuildAuthMethods_KeyFileInvalidPEM(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "bad_key")
	if err := os.WriteFile(keyPath, []byte("not valid pem"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := buildAuthMethods(map[string]any{
		"key_file": keyPath,
	})
	if err == nil {
		t.Fatal("expected error for invalid PEM in key file")
	}
}

func TestBuildAuthMethods_Empty(t *testing.T) {
	methods, err := buildAuthMethods(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 0 {
		t.Fatalf("expected 0 auth methods, got %d", len(methods))
	}
}

func TestBuildAuthMethods_AllThree(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_test")
	if err := os.WriteFile(keyPath, testRSAKey, 0600); err != nil {
		t.Fatal(err)
	}

	methods, err := buildAuthMethods(map[string]any{
		"key_file": keyPath,
		"key":      string(testRSAKey),
		"password": "pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 3 {
		t.Fatalf("expected 3 auth methods, got %d", len(methods))
	}
}

func TestAuthFromPEM_Valid(t *testing.T) {
	method, err := authFromPEM(testRSAKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if method == nil {
		t.Fatal("expected non-nil auth method")
	}
}

func TestAuthFromPEM_Invalid(t *testing.T) {
	_, err := authFromPEM([]byte("garbage"))
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestAuthFromKeyFile_ReadError(t *testing.T) {
	origRead := readKeyFile
	defer func() { readKeyFile = origRead }()

	readKeyFile = func(path string) ([]byte, error) {
		return nil, fmt.Errorf("read error")
	}

	_, err := authFromKeyFile("/fake/path")
	if err == nil {
		t.Fatal("expected error for read failure")
	}
}

func TestBuildHostKeyCallback(t *testing.T) {
	// Both should return non-nil callbacks.
	cb1 := buildHostKeyCallback(true)
	if cb1 == nil {
		t.Fatal("expected non-nil callback with check=true")
	}

	cb2 := buildHostKeyCallback(false)
	if cb2 == nil {
		t.Fatal("expected non-nil callback with check=false")
	}
}
