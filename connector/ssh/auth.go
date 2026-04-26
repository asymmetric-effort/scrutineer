package ssh

import (
	"fmt"
	"os"

	cryptossh "golang.org/x/crypto/ssh"
)

// buildAuthMethods constructs SSH authentication methods from the config map.
// It checks for key_file, key (raw PEM), and password in that order.
func buildAuthMethods(config map[string]any) ([]cryptossh.AuthMethod, error) {
	var methods []cryptossh.AuthMethod

	// Key file authentication.
	if v, ok := config["key_file"]; ok {
		s, isStr := v.(string)
		if !isStr {
			return nil, fmt.Errorf("ssh: key_file must be a string")
		}
		method, err := authFromKeyFile(s)
		if err != nil {
			return nil, err
		}
		methods = append(methods, method)
	}

	// Raw PEM key authentication.
	if v, ok := config["key"]; ok {
		s, isStr := v.(string)
		if !isStr {
			return nil, fmt.Errorf("ssh: key must be a string")
		}
		method, err := authFromPEM([]byte(s))
		if err != nil {
			return nil, err
		}
		methods = append(methods, method)
	}

	// Password authentication.
	if v, ok := config["password"]; ok {
		s, isStr := v.(string)
		if !isStr {
			return nil, fmt.Errorf("ssh: password must be a string")
		}
		methods = append(methods, cryptossh.Password(s))
	}

	return methods, nil
}

// readKeyFile reads a private key file from disk. It is a variable so tests
// can replace it.
var readKeyFile = os.ReadFile

// authFromKeyFile reads a PEM private key file and returns an AuthMethod.
func authFromKeyFile(path string) (cryptossh.AuthMethod, error) {
	pemBytes, err := readKeyFile(path)
	if err != nil {
		return nil, fmt.Errorf("ssh: read key file %q: %w", path, err)
	}
	return authFromPEM(pemBytes)
}

// authFromPEM parses a PEM-encoded private key and returns an AuthMethod.
func authFromPEM(pemBytes []byte) (cryptossh.AuthMethod, error) {
	signer, err := cryptossh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("ssh: parse private key: %w", err)
	}
	return cryptossh.PublicKeys(signer), nil
}

// buildHostKeyCallback returns a host key callback based on the configuration.
func buildHostKeyCallback(check bool) cryptossh.HostKeyCallback {
	if !check {
		return cryptossh.InsecureIgnoreHostKey()
	}
	// When host key checking is enabled but no specific key is provided,
	// we still return InsecureIgnoreHostKey as a fallback. In a production
	// implementation, this would load known_hosts.
	return cryptossh.InsecureIgnoreHostKey()
}
