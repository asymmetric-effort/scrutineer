package graphql

import (
	"context"
	"fmt"
	"net/http"
)

// IntrospectionQuery is the standard GraphQL introspection query string.
// It fetches the full schema including types, fields, query/mutation/subscription
// root types.
const IntrospectionQuery = `{
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      name
      kind
      fields(includeDeprecated: true) {
        name
        type {
          name
          kind
          ofType {
            name
            kind
          }
        }
      }
    }
  }
}`

// Schema represents the result of an introspection query.
type Schema struct {
	Types            []SchemaType
	QueryType        string
	MutationType     string
	SubscriptionType string
}

// SchemaType describes a GraphQL type from introspection.
type SchemaType struct {
	Name   string
	Kind   string
	Fields []SchemaField
}

// SchemaField describes a field within a GraphQL type.
type SchemaField struct {
	Name string
	Type string
}

// Introspect sends an introspection query to the given endpoint and parses
// the resulting schema. The http.Client and optional headers are forwarded
// to Execute.
func Introspect(ctx context.Context, client *http.Client, endpoint string, headers map[string]string) (*Schema, error) {
	req := Request{Query: IntrospectionQuery}
	resp, err := Execute(ctx, client, endpoint, req, headers)
	if err != nil {
		return nil, fmt.Errorf("introspection request failed: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("introspection returned errors: %s", resp.Errors[0].Message)
	}

	return parseIntrospectionResponse(resp)
}

// parseIntrospectionResponse extracts a Schema from the introspection response.
func parseIntrospectionResponse(resp *Response) (*Schema, error) {
	// resp.Data is any — typically map[string]any after JSON decode.
	dataMap, ok := resp.Data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected introspection data type: %T", resp.Data)
	}

	schemaMap, ok := dataMap["__schema"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing __schema in introspection response")
	}

	schema := &Schema{}

	// Extract root type names.
	if qt, ok := schemaMap["queryType"].(map[string]any); ok {
		if name, ok := qt["name"].(string); ok {
			schema.QueryType = name
		}
	}
	if mt, ok := schemaMap["mutationType"].(map[string]any); ok {
		if name, ok := mt["name"].(string); ok {
			schema.MutationType = name
		}
	}
	if st, ok := schemaMap["subscriptionType"].(map[string]any); ok {
		if name, ok := st["name"].(string); ok {
			schema.SubscriptionType = name
		}
	}

	// Extract types.
	types, ok := schemaMap["types"].([]any)
	if !ok {
		return schema, nil
	}

	for _, t := range types {
		typeMap, ok := t.(map[string]any)
		if !ok {
			continue
		}

		st := SchemaType{}
		if name, ok := typeMap["name"].(string); ok {
			st.Name = name
		}
		if kind, ok := typeMap["kind"].(string); ok {
			st.Kind = kind
		}

		if fields, ok := typeMap["fields"].([]any); ok {
			for _, f := range fields {
				fieldMap, ok := f.(map[string]any)
				if !ok {
					continue
				}
				sf := SchemaField{}
				if name, ok := fieldMap["name"].(string); ok {
					sf.Name = name
				}
				if typeInfo, ok := fieldMap["type"].(map[string]any); ok {
					sf.Type = extractTypeName(typeInfo)
				}
				st.Fields = append(st.Fields, sf)
			}
		}

		schema.Types = append(schema.Types, st)
	}

	return schema, nil
}

// extractTypeName builds a readable type name from introspection type info.
func extractTypeName(typeInfo map[string]any) string {
	if name, ok := typeInfo["name"].(string); ok && name != "" {
		return name
	}
	// For wrapper types (NON_NULL, LIST), recurse into ofType.
	kind, _ := typeInfo["kind"].(string)
	if ofType, ok := typeInfo["ofType"].(map[string]any); ok {
		inner := extractTypeName(ofType)
		switch kind {
		case "NON_NULL":
			return inner + "!"
		case "LIST":
			return "[" + inner + "]"
		default:
			return inner
		}
	}
	return kind
}
