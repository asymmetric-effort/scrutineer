package graphql

import "testing"

func TestBuildRequest_AllFields(t *testing.T) {
	t.Parallel()

	params := map[string]any{
		"query":          "mutation AddUser($name: String!) { addUser(name: $name) { id } }",
		"variables":      map[string]any{"name": "Alice"},
		"operation_name": "AddUser",
	}

	req, err := BuildRequest(params)
	if err != nil {
		t.Fatalf("BuildRequest: %v", err)
	}
	if req.Query != params["query"] {
		t.Errorf("query mismatch")
	}
	if req.Variables["name"] != "Alice" {
		t.Errorf("variables mismatch: %v", req.Variables)
	}
	if req.OperationName != "AddUser" {
		t.Errorf("operation_name mismatch: %s", req.OperationName)
	}
}

func TestBuildRequest_QueryOnly(t *testing.T) {
	t.Parallel()

	params := map[string]any{
		"query": "{ hero { name } }",
	}

	req, err := BuildRequest(params)
	if err != nil {
		t.Fatalf("BuildRequest: %v", err)
	}
	if req.Query != "{ hero { name } }" {
		t.Errorf("query mismatch: %s", req.Query)
	}
	if req.Variables != nil {
		t.Errorf("expected nil variables, got %v", req.Variables)
	}
	if req.OperationName != "" {
		t.Errorf("expected empty operation name, got %s", req.OperationName)
	}
}

func TestBuildRequest_MissingQuery(t *testing.T) {
	t.Parallel()

	_, err := BuildRequest(map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing query")
	}
}

func TestBuildRequest_EmptyQuery(t *testing.T) {
	t.Parallel()

	_, err := BuildRequest(map[string]any{"query": ""})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestBuildRequest_QueryWrongType(t *testing.T) {
	t.Parallel()

	_, err := BuildRequest(map[string]any{"query": 42})
	if err == nil {
		t.Fatal("expected error for non-string query")
	}
}

func TestBuildRequest_VariablesWrongType(t *testing.T) {
	t.Parallel()

	_, err := BuildRequest(map[string]any{
		"query":     "{ x }",
		"variables": "not a map",
	})
	if err == nil {
		t.Fatal("expected error for non-map variables")
	}
}

func TestBuildRequest_OperationNameWrongType(t *testing.T) {
	t.Parallel()

	_, err := BuildRequest(map[string]any{
		"query":          "{ x }",
		"operation_name": 123,
	})
	if err == nil {
		t.Fatal("expected error for non-string operation_name")
	}
}

func TestBuildRequest_WithVariables(t *testing.T) {
	t.Parallel()

	vars := map[string]any{
		"id":     42,
		"active": true,
		"tags":   []any{"a", "b"},
	}
	params := map[string]any{
		"query":     "query GetUser($id: Int!) { user(id: $id) { name } }",
		"variables": vars,
	}

	req, err := BuildRequest(params)
	if err != nil {
		t.Fatalf("BuildRequest: %v", err)
	}
	if req.Variables["id"] != 42 {
		t.Errorf("expected id=42, got %v", req.Variables["id"])
	}
	if req.Variables["active"] != true {
		t.Errorf("expected active=true, got %v", req.Variables["active"])
	}
}
