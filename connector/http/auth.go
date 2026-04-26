package http

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

// applyAuth reads the "auth" parameter from step parameters and sets the
// appropriate authentication header on the request.
//
// Supported auth types:
//   - "bearer": sets Authorization: Bearer <token>
//   - "basic": sets Authorization: Basic <base64(user:pass)>
//   - "api_key": sets a custom header to the key value
func applyAuth(req *http.Request, params map[string]any) error {
	v, ok := params["auth"]
	if !ok {
		return nil
	}

	authMap, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("auth must be a map")
	}

	authType, ok := authMap["type"].(string)
	if !ok {
		return fmt.Errorf("auth.type must be a string")
	}

	switch authType {
	case "bearer":
		token, ok := authMap["token"].(string)
		if !ok {
			return fmt.Errorf("auth.token must be a string")
		}
		req.Header.Set("Authorization", "Bearer "+token)

	case "basic":
		user, ok := authMap["username"].(string)
		if !ok {
			return fmt.Errorf("auth.username must be a string")
		}
		pass, ok := authMap["password"].(string)
		if !ok {
			return fmt.Errorf("auth.password must be a string")
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		req.Header.Set("Authorization", "Basic "+encoded)

	case "api_key":
		header, ok := authMap["header"].(string)
		if !ok {
			return fmt.Errorf("auth.header must be a string")
		}
		key, ok := authMap["key"].(string)
		if !ok {
			return fmt.Errorf("auth.key must be a string")
		}
		req.Header.Set(header, key)

	default:
		return fmt.Errorf("unsupported auth type: %s", authType)
	}

	return nil
}
