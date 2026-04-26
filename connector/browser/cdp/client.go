package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// Client is a CDP JSON-RPC client that communicates over WebSocket.
type Client struct {
	ws        *wsConn
	nextID    atomic.Int64
	pending   sync.Map // map[int64]chan *Message
	subs      map[string][]func(map[string]any)
	subsMu    sync.RWMutex
	done      chan struct{}
	closeOnce sync.Once
	readErr   error
	mu        sync.Mutex
}

// Dial connects to a CDP WebSocket endpoint and starts the read loop.
func Dial(ctx context.Context, wsURL string) (*Client, error) {
	ws, err := dialWebSocket(wsURL)
	if err != nil {
		return nil, err
	}

	c := &Client{
		ws:   ws,
		subs: make(map[string][]func(map[string]any)),
		done: make(chan struct{}),
	}
	c.nextID.Store(0)

	go c.readLoop()

	return c, nil
}

// readLoop continuously reads messages from the WebSocket and dispatches them.
func (c *Client) readLoop() {
	defer c.closeOnce.Do(func() { close(c.done) })

	for {
		data, err := c.ws.ReadMessage()
		if err != nil {
			c.mu.Lock()
			c.readErr = err
			c.mu.Unlock()
			// Wake all pending callers.
			c.pending.Range(func(key, value any) bool {
				ch := value.(chan *Message)
				select {
				case ch <- &Message{Error: &ErrorInfo{Message: err.Error()}}:
				default:
				}
				return true
			})
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// Response to a request.
		if msg.ID != 0 {
			if ch, ok := c.pending.LoadAndDelete(msg.ID); ok {
				ch.(chan *Message) <- &msg
			}
			continue
		}

		// Event dispatch.
		if msg.Method != "" {
			c.dispatchEvent(&msg)
		}
	}
}

// dispatchEvent calls all subscribers for the given event method.
func (c *Client) dispatchEvent(msg *Message) {
	c.subsMu.RLock()
	handlers := c.subs[msg.Method]
	// Copy the slice under the lock so we can release it before calling handlers.
	hCopy := make([]func(map[string]any), len(handlers))
	copy(hCopy, handlers)
	c.subsMu.RUnlock()

	if len(hCopy) == 0 {
		return
	}

	var params map[string]any
	if msg.Params != nil {
		_ = json.Unmarshal(msg.Params, &params)
	}

	for _, h := range hCopy {
		go h(params)
	}
}

// Send sends a CDP command and waits for the response.
func (c *Client) Send(method string, params map[string]any) (map[string]any, error) {
	return c.sendWithSession(method, params, "")
}

// sendWithSession sends a CDP command with an optional session ID.
func (c *Client) sendWithSession(method string, params map[string]any, sessionID string) (map[string]any, error) {
	select {
	case <-c.done:
		c.mu.Lock()
		err := c.readErr
		c.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("cdp: connection closed: %w", err)
		}
		return nil, errors.New("cdp: connection closed")
	default:
	}

	id := c.nextID.Add(1)

	msg := Message{
		ID:     id,
		Method: method,
	}
	if sessionID != "" {
		msg.SessionID = sessionID
	}

	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("cdp: marshal params: %w", err)
		}
		msg.Params = raw
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("cdp: marshal message: %w", err)
	}

	ch := make(chan *Message, 1)
	c.pending.Store(id, ch)
	defer c.pending.Delete(id)

	if err := c.ws.WriteText(data); err != nil {
		return nil, fmt.Errorf("cdp: write: %w", err)
	}

	resp := <-ch
	if resp.Error != nil {
		return nil, resp.Error
	}

	var result map[string]any
	if resp.Result != nil {
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, fmt.Errorf("cdp: unmarshal result: %w", err)
		}
	}

	return result, nil
}

// Subscribe registers a handler for a CDP event.
func (c *Client) Subscribe(event string, handler func(params map[string]any)) {
	c.subsMu.Lock()
	c.subs[event] = append(c.subs[event], handler)
	c.subsMu.Unlock()
}

// Close closes the CDP client connection.
func (c *Client) Close() error {
	c.closeOnce.Do(func() { close(c.done) })
	return c.ws.Close()
}

// Done returns a channel that is closed when the client connection ends.
func (c *Client) Done() <-chan struct{} {
	return c.done
}
