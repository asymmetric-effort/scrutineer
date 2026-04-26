package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntrospect_ValidSchema(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Data: map[string]any{
				"__schema": map[string]any{
					"queryType":        map[string]any{"name": "Query"},
					"mutationType":     map[string]any{"name": "Mutation"},
					"subscriptionType": map[string]any{"name": "Subscription"},
					"types": []any{
						map[string]any{
							"name": "Query",
							"kind": "OBJECT",
							"fields": []any{
								map[string]any{
									"name": "hero",
									"type": map[string]any{
										"name": "Hero",
										"kind": "OBJECT",
									},
								},
								map[string]any{
									"name": "users",
									"type": map[string]any{
										"name": "",
										"kind": "NON_NULL",
										"ofType": map[string]any{
											"name": "User",
											"kind": "OBJECT",
										},
									},
								},
							},
						},
						map[string]any{
							"name":   "Hero",
							"kind":   "OBJECT",
							"fields": []any{},
						},
						map[string]any{
							"name":   "String",
							"kind":   "SCALAR",
							"fields": nil,
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	schema, err := Introspect(context.Background(), srv.Client(), srv.URL, nil)
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}

	if schema.QueryType != "Query" {
		t.Errorf("expected QueryType=Query, got %s", schema.QueryType)
	}
	if schema.MutationType != "Mutation" {
		t.Errorf("expected MutationType=Mutation, got %s", schema.MutationType)
	}
	if schema.SubscriptionType != "Subscription" {
		t.Errorf("expected SubscriptionType=Subscription, got %s", schema.SubscriptionType)
	}
	if len(schema.Types) != 3 {
		t.Fatalf("expected 3 types, got %d", len(schema.Types))
	}

	queryType := schema.Types[0]
	if queryType.Name != "Query" || queryType.Kind != "OBJECT" {
		t.Errorf("unexpected query type: %+v", queryType)
	}
	if len(queryType.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(queryType.Fields))
	}
	if queryType.Fields[0].Name != "hero" || queryType.Fields[0].Type != "Hero" {
		t.Errorf("unexpected field: %+v", queryType.Fields[0])
	}
	// NON_NULL wrapper type.
	if queryType.Fields[1].Type != "User!" {
		t.Errorf("expected User!, got %s", queryType.Fields[1].Type)
	}
}

func TestIntrospect_ErrorResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Errors: []GraphQLError{{Message: "not authorized"}},
		})
	}))
	defer srv.Close()

	_, err := Introspect(context.Background(), srv.Client(), srv.URL, nil)
	if err == nil {
		t.Fatal("expected error for error response")
	}
	if !strings.Contains(err.Error(), "not authorized") {
		t.Errorf("error should contain message: %v", err)
	}
}

func TestIntrospect_NetworkError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	_, err := Introspect(context.Background(), srv.Client(), srv.URL, nil)
	if err == nil {
		t.Fatal("expected error for closed server")
	}
}

func TestIntrospect_WithHeaders(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Data: map[string]any{
				"__schema": map[string]any{
					"queryType":        map[string]any{"name": "Query"},
					"mutationType":     nil,
					"subscriptionType": nil,
					"types":            []any{},
				},
			},
		})
	}))
	defer srv.Close()

	schema, err := Introspect(context.Background(), srv.Client(), srv.URL, map[string]string{
		"X-Api-Key": "test-key",
	})
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}
	if schema.QueryType != "Query" {
		t.Errorf("expected QueryType=Query, got %s", schema.QueryType)
	}
	// Nil mutation/subscription types should be empty strings.
	if schema.MutationType != "" {
		t.Errorf("expected empty MutationType, got %s", schema.MutationType)
	}
}

func TestIntrospect_InvalidDataType(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return data as a string instead of object.
		w.Write([]byte(`{"data":"not an object"}`))
	}))
	defer srv.Close()

	_, err := Introspect(context.Background(), srv.Client(), srv.URL, nil)
	if err == nil {
		t.Fatal("expected error for invalid data type")
	}
}

func TestIntrospect_MissingSchema(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Data: map[string]any{"other": "stuff"},
		})
	}))
	defer srv.Close()

	_, err := Introspect(context.Background(), srv.Client(), srv.URL, nil)
	if err == nil {
		t.Fatal("expected error for missing __schema")
	}
}

func TestExtractTypeName_LIST(t *testing.T) {
	t.Parallel()

	typeInfo := map[string]any{
		"name": "",
		"kind": "LIST",
		"ofType": map[string]any{
			"name": "String",
			"kind": "SCALAR",
		},
	}

	name := extractTypeName(typeInfo)
	if name != "[String]" {
		t.Errorf("expected [String], got %s", name)
	}
}

func TestExtractTypeName_NoOfType(t *testing.T) {
	t.Parallel()

	typeInfo := map[string]any{
		"name": "",
		"kind": "UNKNOWN",
	}

	name := extractTypeName(typeInfo)
	if name != "UNKNOWN" {
		t.Errorf("expected UNKNOWN, got %s", name)
	}
}

func TestIntrospect_NilTypes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Data: map[string]any{
				"__schema": map[string]any{
					"queryType": map[string]any{"name": "Query"},
					"types":     nil,
				},
			},
		})
	}))
	defer srv.Close()

	schema, err := Introspect(context.Background(), srv.Client(), srv.URL, nil)
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}
	if len(schema.Types) != 0 {
		t.Errorf("expected 0 types, got %d", len(schema.Types))
	}
}

func TestExtractTypeName_UnknownKindWithOfType(t *testing.T) {
	t.Parallel()

	// A kind that is neither NON_NULL nor LIST but has ofType.
	typeInfo := map[string]any{
		"name": "",
		"kind": "SOME_OTHER",
		"ofType": map[string]any{
			"name": "Inner",
			"kind": "OBJECT",
		},
	}

	name := extractTypeName(typeInfo)
	if name != "Inner" {
		t.Errorf("expected Inner, got %s", name)
	}
}

func TestIntrospectionQueryConstant(t *testing.T) {
	t.Parallel()

	if !strings.Contains(IntrospectionQuery, "__schema") {
		t.Error("introspection query should contain __schema")
	}
	if !strings.Contains(IntrospectionQuery, "queryType") {
		t.Error("introspection query should contain queryType")
	}
	if !strings.Contains(IntrospectionQuery, "mutationType") {
		t.Error("introspection query should contain mutationType")
	}
	if !strings.Contains(IntrospectionQuery, "subscriptionType") {
		t.Error("introspection query should contain subscriptionType")
	}
}

func TestIntrospect_TypeWithBadFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Data: map[string]any{
				"__schema": map[string]any{
					"queryType": map[string]any{"name": "Query"},
					"types": []any{
						map[string]any{
							"name": "Query",
							"kind": "OBJECT",
							"fields": []any{
								"not a map", // invalid field entry
							},
						},
						"not a type map", // invalid type entry
					},
				},
			},
		})
	}))
	defer srv.Close()

	schema, err := Introspect(context.Background(), srv.Client(), srv.URL, nil)
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}
	// Should skip bad entries gracefully.
	if len(schema.Types) != 1 {
		t.Errorf("expected 1 valid type, got %d", len(schema.Types))
	}
	if len(schema.Types[0].Fields) != 0 {
		t.Errorf("expected 0 valid fields, got %d", len(schema.Types[0].Fields))
	}
}
