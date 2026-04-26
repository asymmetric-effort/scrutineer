package cdp

import "fmt"

// Session represents a CDP session attached to a specific target (page, worker, etc.).
type Session struct {
	client    *Client
	sessionID string
	targetID  string
}

// NewSession attaches to a target and creates a CDP session.
func (c *Client) NewSession(targetID string) (*Session, error) {
	result, err := c.Send("Target.attachToTarget", map[string]any{
		"targetId": targetID,
		"flatten":  true,
	})
	if err != nil {
		return nil, fmt.Errorf("cdp: attach to target %s: %w", targetID, err)
	}

	sessionID, ok := result["sessionId"].(string)
	if !ok {
		return nil, fmt.Errorf("cdp: no sessionId in attach response")
	}

	return &Session{
		client:    c,
		sessionID: sessionID,
		targetID:  targetID,
	}, nil
}

// Send sends a CDP command within this session scope.
func (s *Session) Send(method string, params map[string]any) (map[string]any, error) {
	return s.client.sendWithSession(method, params, s.sessionID)
}

// SessionID returns the CDP session identifier.
func (s *Session) SessionID() string {
	return s.sessionID
}

// TargetID returns the target identifier for this session.
func (s *Session) TargetID() string {
	return s.targetID
}

// Close detaches from the target.
func (s *Session) Close() error {
	_, err := s.client.Send("Target.detachFromTarget", map[string]any{
		"sessionId": s.sessionID,
	})
	return err
}
