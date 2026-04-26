package cdp

import (
	"encoding/json"
	"testing"
)

func TestErrorInfo_Error(t *testing.T) {
	tests := []struct {
		name string
		err  ErrorInfo
		want string
	}{
		{
			name: "message only",
			err:  ErrorInfo{Code: -32600, Message: "Invalid Request"},
			want: "Invalid Request",
		},
		{
			name: "message with data",
			err:  ErrorInfo{Code: -32601, Message: "Method not found", Data: "DOM.foo"},
			want: "Method not found: DOM.foo",
		},
		{
			name: "empty data",
			err:  ErrorInfo{Code: -32602, Message: "Invalid params", Data: ""},
			want: "Invalid params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMessage_JSONRoundTrip(t *testing.T) {
	t.Run("request message", func(t *testing.T) {
		params := json.RawMessage(`{"url":"http://example.com"}`)
		msg := Message{
			ID:     1,
			Method: "Page.navigate",
			Params: params,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.ID != 1 {
			t.Errorf("ID = %d, want 1", decoded.ID)
		}
		if decoded.Method != "Page.navigate" {
			t.Errorf("Method = %q, want %q", decoded.Method, "Page.navigate")
		}
	})

	t.Run("response message", func(t *testing.T) {
		result := json.RawMessage(`{"frameId":"123"}`)
		msg := Message{
			ID:     1,
			Result: result,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.ID != 1 {
			t.Errorf("ID = %d, want 1", decoded.ID)
		}
		if string(decoded.Result) != `{"frameId":"123"}` {
			t.Errorf("Result = %s, want %s", decoded.Result, `{"frameId":"123"}`)
		}
	})

	t.Run("error message", func(t *testing.T) {
		msg := Message{
			ID:    1,
			Error: &ErrorInfo{Code: -32600, Message: "Invalid Request"},
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.Error == nil {
			t.Fatal("Error should not be nil")
		}
		if decoded.Error.Code != -32600 {
			t.Errorf("Error.Code = %d, want -32600", decoded.Error.Code)
		}
	})

	t.Run("event message", func(t *testing.T) {
		params := json.RawMessage(`{"timestamp":1234.5}`)
		msg := Message{
			Method: "Page.loadEventFired",
			Params: params,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.ID != 0 {
			t.Errorf("ID = %d, want 0", decoded.ID)
		}
		if decoded.Method != "Page.loadEventFired" {
			t.Errorf("Method = %q, want %q", decoded.Method, "Page.loadEventFired")
		}
	})

	t.Run("session message", func(t *testing.T) {
		msg := Message{
			ID:        2,
			Method:    "Runtime.evaluate",
			SessionID: "session-123",
			Params:    json.RawMessage(`{"expression":"1+1"}`),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.SessionID != "session-123" {
			t.Errorf("SessionID = %q, want %q", decoded.SessionID, "session-123")
		}
	})
}

func TestMessage_OmitEmpty(t *testing.T) {
	msg := Message{}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Should be essentially empty JSON object.
	want := `{}`
	if string(data) != want {
		t.Errorf("empty message = %s, want %s", data, want)
	}
}

func TestErrorInfo_Implements_error(t *testing.T) {
	var err error = &ErrorInfo{Message: "test"}
	if err.Error() != "test" {
		t.Errorf("unexpected: %v", err)
	}
}
