package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
)

// executeFilesystem handles the "filesystem" action. It checks file existence,
// content, and size constraints.
func (c *CLIConnector) executeFilesystem(_ context.Context, step connector.Step) (*connector.Result, error) {
	path, ok, err := paramString(step.Parameters, "path")
	if err != nil {
		return nil, err
	}
	if !ok || path == "" {
		return nil, fmt.Errorf("cli: filesystem requires a \"path\" parameter")
	}

	start := time.Now()

	result := &connector.Result{
		Data: map[string]any{},
		Meta: map[string]string{
			"connector": "cli",
			"action":    "filesystem",
		},
	}

	info, statErr := os.Stat(path)
	exists := statErr == nil

	result.Data["exists"] = exists

	// Check expected existence.
	expectedExists, hasExpected, err := paramBool(step.Parameters, "exists")
	if err != nil {
		return nil, err
	}
	if hasExpected && expectedExists != exists {
		result.Elapsed = time.Since(start)
		if expectedExists {
			result.Data["error"] = fmt.Sprintf("expected path %q to exist, but it does not", path)
		} else {
			result.Data["error"] = fmt.Sprintf("expected path %q to not exist, but it does", path)
		}
		return result, nil
	}

	if !exists {
		result.Elapsed = time.Since(start)
		return result, nil
	}

	result.Data["size"] = info.Size()
	result.Data["is_dir"] = info.IsDir()

	// Read content for regular files (not directories).
	if !info.IsDir() {
		content, readErr := os.ReadFile(path)
		if readErr == nil {
			result.Data["content"] = string(content)
		}

		// Check contains.
		containsStr, hasContains, err := paramString(step.Parameters, "contains")
		if err != nil {
			return nil, err
		}
		if hasContains && readErr == nil {
			result.Data["contains"] = strings.Contains(string(content), containsStr)
		}
	}

	// Check size constraints.
	sizeMap, hasSize, err := paramMap(step.Parameters, "size")
	if err != nil {
		return nil, err
	}
	if hasSize {
		fileSize := info.Size()
		if gt, ok := sizeMap["greater_than"]; ok {
			gtVal, err := toInt64(gt)
			if err != nil {
				return nil, fmt.Errorf("cli: size.greater_than: %w", err)
			}
			result.Data["size_greater_than"] = fileSize > gtVal
		}
		if lt, ok := sizeMap["less_than"]; ok {
			ltVal, err := toInt64(lt)
			if err != nil {
				return nil, fmt.Errorf("cli: size.less_than: %w", err)
			}
			result.Data["size_less_than"] = fileSize < ltVal
		}
	}

	result.Elapsed = time.Since(start)
	return result, nil
}

// toInt64 converts various numeric types to int64.
func toInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int:
		return int64(n), nil
	case int64:
		return n, nil
	case float64:
		return int64(n), nil
	case float32:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("expected numeric value, got %T", v)
	}
}
