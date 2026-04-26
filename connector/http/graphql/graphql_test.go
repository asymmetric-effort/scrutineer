package graphql

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecute_ValidQuery(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		if acc := r.Header.Get("Accept"); acc != "application/json" {
			t.Errorf("expected Accept application/json, got %s", acc)
		}

		body, _ := io.ReadAll(r.Body)
		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decoding request: %v", err)
		}
		if req.Query != "{ hero { name } }" {
			t.Errorf("unexpected query: %s", req.Query)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Data: map[string]any{
				"hero": map[string]any{"name": "Luke"},
			},
		})
	}))
	defer srv.Close()

	resp, err := Execute(context.Background(), srv.Client(), srv.URL, Request{
		Query: "{ hero { name } }",
	}, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.Data == nil {
		t.Fatal("expected data, got nil")
	}
	data := resp.Data.(map[string]any)
	hero := data["hero"].(map[string]any)
	if hero["name"] != "Luke" {
		t.Errorf("expected Luke, got %v", hero["name"])
	}
}

func TestExecute_WithHeaders(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer token123" {
			t.Errorf("expected Authorization header, got %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Data: "ok"})
	}))
	defer srv.Close()

	resp, err := Execute(context.Background(), srv.Client(), srv.URL, Request{
		Query: "{ me { id } }",
	}, map[string]string{"Authorization": "Bearer token123"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.Data == nil {
		t.Fatal("expected data")
	}
}

func TestExecute_GraphQLErrors(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Errors: []GraphQLError{
				{
					Message:   "field not found",
					Locations: []Location{{Line: 1, Column: 3}},
					Path:      []any{"hero", "age"},
					Extensions: map[string]any{
						"code": "FIELD_NOT_FOUND",
					},
				},
			},
		})
	}))
	defer srv.Close()

	resp, err := Execute(context.Background(), srv.Client(), srv.URL, Request{
		Query: "{ hero { age } }",
	}, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(resp.Errors))
	}
	if resp.Errors[0].Message != "field not found" {
		t.Errorf("unexpected error message: %s", resp.Errors[0].Message)
	}
	if len(resp.Errors[0].Locations) != 1 || resp.Errors[0].Locations[0].Line != 1 {
		t.Errorf("unexpected locations: %v", resp.Errors[0].Locations)
	}
	if resp.Errors[0].Extensions["code"] != "FIELD_NOT_FOUND" {
		t.Errorf("unexpected extension: %v", resp.Errors[0].Extensions)
	}
}

func TestExecute_NetworkError(t *testing.T) {
	t.Parallel()

	// Use a closed server to trigger a network error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	_, err := Execute(context.Background(), srv.Client(), srv.URL, Request{
		Query: "{ hero { name } }",
	}, nil)
	if err == nil {
		t.Fatal("expected error for closed server")
	}
}

func TestExecute_InvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := Execute(context.Background(), srv.Client(), srv.URL, Request{
		Query: "{ hero { name } }",
	}, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestExecute_InvalidURL(t *testing.T) {
	t.Parallel()

	_, err := Execute(context.Background(), nil, "://bad", Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestExecute_NilClient(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Data: "ok"})
	}))
	defer srv.Close()

	// nil client should use http.DefaultClient.
	resp, err := Execute(context.Background(), nil, srv.URL, Request{Query: "{ x }"}, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.Data == nil {
		t.Fatal("expected data")
	}
}

func TestExecute_ContextCancelled(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block forever — context should cancel.
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := Execute(ctx, srv.Client(), srv.URL, Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRequestJSONMarshal(t *testing.T) {
	t.Parallel()

	req := Request{
		Query:         "mutation { addUser(name: $name) { id } }",
		Variables:     map[string]any{"name": "Alice"},
		OperationName: "AddUser",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Query != req.Query {
		t.Errorf("query mismatch: %s vs %s", decoded.Query, req.Query)
	}
	if decoded.OperationName != req.OperationName {
		t.Errorf("operation name mismatch")
	}
	if decoded.Variables["name"] != "Alice" {
		t.Errorf("variables mismatch: %v", decoded.Variables)
	}
}

func TestRequestJSONMarshal_Minimal(t *testing.T) {
	t.Parallel()

	req := Request{Query: "{ hero { name } }"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Ensure omitempty works — no variables or operationName keys.
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if _, ok := raw["variables"]; ok {
		t.Error("expected variables to be omitted")
	}
	if _, ok := raw["operationName"]; ok {
		t.Error("expected operationName to be omitted")
	}
}

func TestExecute_ReadBodyError(t *testing.T) {
	t.Parallel()

	// Create a server that sends an incomplete chunked response to trigger io.ReadAll error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hijack the connection to write a broken response.
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("server doesn't support hijacking")
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			t.Fatal(err)
			return
		}
		// Write a chunked response that is incomplete.
		buf.WriteString("HTTP/1.1 200 OK\r\n")
		buf.WriteString("Transfer-Encoding: chunked\r\n")
		buf.WriteString("Content-Type: application/json\r\n")
		buf.WriteString("\r\n")
		// Write an invalid chunk to cause read error.
		buf.WriteString("ZZZZ\r\n")
		buf.Flush()
		conn.Close()
	}))
	defer srv.Close()

	_, err := Execute(context.Background(), srv.Client(), srv.URL, Request{Query: "{ x }"}, nil)
	if err == nil {
		t.Fatal("expected error for broken response body")
	}
}

func TestResponseJSONRoundtrip(t *testing.T) {
	t.Parallel()

	original := Response{
		Data: map[string]any{"foo": "bar"},
		Errors: []GraphQLError{
			{Message: "oops", Path: []any{"foo"}},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Errors) != 1 || decoded.Errors[0].Message != "oops" {
		t.Errorf("roundtrip failed: %+v", decoded)
	}
}
