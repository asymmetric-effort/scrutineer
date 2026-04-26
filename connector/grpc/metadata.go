package grpc

import (
	"fmt"

	"google.golang.org/grpc/metadata"
)

// buildOutgoingMetadata constructs gRPC metadata from step parameters.
// The "metadata" parameter should be a map[string]any where values are strings
// or slices of strings.
func buildOutgoingMetadata(params map[string]any) metadata.MD {
	md := metadata.MD{}
	v, ok := params["metadata"]
	if !ok {
		return md
	}
	m, ok := v.(map[string]any)
	if !ok {
		return md
	}
	for key, val := range m {
		switch tv := val.(type) {
		case string:
			md.Append(key, tv)
		case []any:
			for _, item := range tv {
				md.Append(key, fmt.Sprintf("%v", item))
			}
		default:
			md.Append(key, fmt.Sprintf("%v", val))
		}
	}
	return md
}

// metadataToMap converts gRPC metadata to a map[string]any for result output.
// Single-value keys are stored as strings; multi-value keys as []string.
func metadataToMap(md metadata.MD) map[string]any {
	result := make(map[string]any)
	for k, vals := range md {
		if len(vals) == 1 {
			result[k] = vals[0]
		} else {
			result[k] = vals
		}
	}
	return result
}
