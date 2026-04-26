package graphql

import "fmt"

// BuildRequest creates a GraphQL Request from step parameters.
// The params map accepts:
//   - "query" (string, required) — the GraphQL query or mutation string.
//   - "variables" (map[string]any, optional) — variables for the query.
//   - "operation_name" (string, optional) — the operation name when the query
//     contains multiple operations.
func BuildRequest(params map[string]any) (*Request, error) {
	query, ok := params["query"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: query")
	}

	queryStr, ok := query.(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string, got %T", query)
	}

	if queryStr == "" {
		return nil, fmt.Errorf("query must not be empty")
	}

	req := &Request{
		Query: queryStr,
	}

	if v, ok := params["variables"]; ok {
		vars, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("variables must be a map[string]any, got %T", v)
		}
		req.Variables = vars
	}

	if v, ok := params["operation_name"]; ok {
		name, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("operation_name must be a string, got %T", v)
		}
		req.OperationName = name
	}

	return req, nil
}
