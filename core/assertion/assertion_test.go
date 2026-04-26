package assertion

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAssertionError_Error(t *testing.T) {
	t.Run("without path", func(t *testing.T) {
		e := &AssertionError{
			Assertion: "equal",
			Expected:  42,
			Actual:    99,
			Message:   "values differ",
		}
		got := e.Error()
		if !strings.Contains(got, "equal") {
			t.Errorf("expected error to contain assertion name, got %q", got)
		}
		if !strings.Contains(got, "values differ") {
			t.Errorf("expected error to contain message, got %q", got)
		}
		if strings.Contains(got, "path") {
			t.Errorf("expected error to not contain path info, got %q", got)
		}
	})

	t.Run("with path", func(t *testing.T) {
		e := &AssertionError{
			Assertion: "json_path",
			Expected:  "alice",
			Actual:    "bob",
			Message:   "value mismatch",
			Path:      "body.user.name",
		}
		got := e.Error()
		if !strings.Contains(got, "body.user.name") {
			t.Errorf("expected error to contain path, got %q", got)
		}
		if !strings.Contains(got, "json_path") {
			t.Errorf("expected error to contain assertion name, got %q", got)
		}
	})
}

func TestDefaultBuilder_Build(t *testing.T) {
	b := &DefaultBuilder{}

	t.Run("equal", func(t *testing.T) {
		a, err := b.Build("equal", 42, nil)
		assertNoError(t, err)
		assertEqual(t, "equal", a.Name())
		assertNoError(t, a.Evaluate(42))
		assertError(t, a.Evaluate(99))
	})

	t.Run("eq alias", func(t *testing.T) {
		a, err := b.Build("eq", "hello", nil)
		assertNoError(t, err)
		assertEqual(t, "equal", a.Name())
	})

	t.Run("not_equal", func(t *testing.T) {
		a, err := b.Build("not_equal", 42, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(99))
		assertError(t, a.Evaluate(42))
	})

	t.Run("neq alias", func(t *testing.T) {
		a, err := b.Build("neq", 42, nil)
		assertNoError(t, err)
		assertEqual(t, "not_equal", a.Name())
	})

	t.Run("deep_equal", func(t *testing.T) {
		a, err := b.Build("deep_equal", []int{1, 2, 3}, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate([]int{1, 2, 3}))
		assertError(t, a.Evaluate([]int{1, 2}))
	})

	t.Run("contains", func(t *testing.T) {
		a, err := b.Build("contains", "world", nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate("hello world"))
	})

	t.Run("contains non-string", func(t *testing.T) {
		_, err := b.Build("contains", 42, nil)
		assertError(t, err)
	})

	t.Run("not_contains", func(t *testing.T) {
		a, err := b.Build("not_contains", "xyz", nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate("hello"))
	})

	t.Run("not_contains non-string", func(t *testing.T) {
		_, err := b.Build("not_contains", 42, nil)
		assertError(t, err)
	})

	t.Run("has_prefix", func(t *testing.T) {
		a, err := b.Build("has_prefix", "hello", nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate("hello world"))
	})

	t.Run("has_prefix non-string", func(t *testing.T) {
		_, err := b.Build("has_prefix", 42, nil)
		assertError(t, err)
	})

	t.Run("has_suffix", func(t *testing.T) {
		a, err := b.Build("has_suffix", "world", nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate("hello world"))
	})

	t.Run("has_suffix non-string", func(t *testing.T) {
		_, err := b.Build("has_suffix", 42, nil)
		assertError(t, err)
	})

	t.Run("matches", func(t *testing.T) {
		a, err := b.Build("matches", `^\d+$`, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate("12345"))
	})

	t.Run("matches non-string", func(t *testing.T) {
		_, err := b.Build("matches", 42, nil)
		assertError(t, err)
	})

	t.Run("matches invalid regex", func(t *testing.T) {
		_, err := b.Build("matches", "[invalid", nil)
		assertError(t, err)
	})

	t.Run("greater_than", func(t *testing.T) {
		a, err := b.Build("greater_than", 10, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(20))
	})

	t.Run("gt alias", func(t *testing.T) {
		a, err := b.Build("gt", 10, nil)
		assertNoError(t, err)
		assertEqual(t, "greater_than", a.Name())
	})

	t.Run("less_than", func(t *testing.T) {
		a, err := b.Build("less_than", 10, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(5))
	})

	t.Run("lt alias", func(t *testing.T) {
		a, err := b.Build("lt", 10, nil)
		assertNoError(t, err)
		assertEqual(t, "less_than", a.Name())
	})

	t.Run("greater_or_equal", func(t *testing.T) {
		a, err := b.Build("greater_or_equal", 10, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(10))
	})

	t.Run("gte alias", func(t *testing.T) {
		a, err := b.Build("gte", 10, nil)
		assertNoError(t, err)
		assertEqual(t, "greater_or_equal", a.Name())
	})

	t.Run("less_or_equal", func(t *testing.T) {
		a, err := b.Build("less_or_equal", 10, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(10))
	})

	t.Run("lte alias", func(t *testing.T) {
		a, err := b.Build("lte", 10, nil)
		assertNoError(t, err)
		assertEqual(t, "less_or_equal", a.Name())
	})

	t.Run("in_range", func(t *testing.T) {
		a, err := b.Build("in_range", nil, map[string]any{"min": 1, "max": 10})
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(5))
	})

	t.Run("in_range missing options", func(t *testing.T) {
		_, err := b.Build("in_range", nil, map[string]any{"min": 1})
		assertError(t, err)
	})

	t.Run("in_range nil options", func(t *testing.T) {
		_, err := b.Build("in_range", nil, nil)
		assertError(t, err)
	})

	t.Run("json_path", func(t *testing.T) {
		a, err := b.Build("json_path", "name", map[string]any{"expected": "alice"})
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(map[string]any{"name": "alice"}))
	})

	t.Run("json_path non-string", func(t *testing.T) {
		_, err := b.Build("json_path", 42, nil)
		assertError(t, err)
	})

	t.Run("json_path missing expected", func(t *testing.T) {
		_, err := b.Build("json_path", "name", map[string]any{})
		assertError(t, err)
	})

	t.Run("not_empty", func(t *testing.T) {
		a, err := b.Build("not_empty", nil, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate("hello"))
	})

	t.Run("status_code", func(t *testing.T) {
		a, err := b.Build("status_code", 200, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(200))
	})

	t.Run("status_code non-int", func(t *testing.T) {
		_, err := b.Build("status_code", "abc", nil)
		assertError(t, err)
	})

	t.Run("status_class", func(t *testing.T) {
		a, err := b.Build("status_class", "2xx", nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(200))
	})

	t.Run("status_class non-string", func(t *testing.T) {
		_, err := b.Build("status_class", 42, nil)
		assertError(t, err)
	})

	t.Run("status_class invalid", func(t *testing.T) {
		_, err := b.Build("status_class", "6xx", nil)
		assertError(t, err)
	})

	t.Run("header_equals", func(t *testing.T) {
		a, err := b.Build("header_equals", "text/html", map[string]any{"header": "Content-Type"})
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}))
	})

	t.Run("header_equals missing header option", func(t *testing.T) {
		_, err := b.Build("header_equals", "v", map[string]any{})
		assertError(t, err)
	})

	t.Run("header_contains", func(t *testing.T) {
		a, err := b.Build("header_contains", "text", map[string]any{"header": "Content-Type"})
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}))
	})

	t.Run("header_contains missing header option", func(t *testing.T) {
		_, err := b.Build("header_contains", "v", map[string]any{})
		assertError(t, err)
	})

	t.Run("header_contains non-string expected", func(t *testing.T) {
		_, err := b.Build("header_contains", 42, map[string]any{"header": "X"})
		assertError(t, err)
	})

	t.Run("header_exists", func(t *testing.T) {
		a, err := b.Build("header_exists", "Content-Type", nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}))
	})

	t.Run("header_exists non-string", func(t *testing.T) {
		_, err := b.Build("header_exists", 42, nil)
		assertError(t, err)
	})

	t.Run("response_time_below with duration", func(t *testing.T) {
		a, err := b.Build("response_time_below", 500*time.Millisecond, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(100*time.Millisecond))
	})

	t.Run("response_time_below with string", func(t *testing.T) {
		a, err := b.Build("response_time_below", "500ms", nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(100*time.Millisecond))
	})

	t.Run("response_time_below invalid", func(t *testing.T) {
		_, err := b.Build("response_time_below", "not-a-duration", nil)
		assertError(t, err)
	})

	t.Run("response_time_below unconvertible", func(t *testing.T) {
		_, err := b.Build("response_time_below", []int{1}, nil)
		assertError(t, err)
	})

	t.Run("length", func(t *testing.T) {
		a, err := b.Build("length", 3, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate([]int{1, 2, 3}))
	})

	t.Run("length non-int", func(t *testing.T) {
		_, err := b.Build("length", "abc", nil)
		assertError(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		a, err := b.Build("empty", nil, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate([]int{}))
	})

	t.Run("collection_not_empty", func(t *testing.T) {
		a, err := b.Build("collection_not_empty", nil, nil)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate([]int{1}))
	})

	t.Run("each", func(t *testing.T) {
		inner := &GreaterThanAssertion{Expected: 0}
		a, err := b.Build("each", nil, map[string]any{"assertion": Assertion(inner)})
		assertNoError(t, err)
		assertNoError(t, a.Evaluate([]int{1, 2, 3}))
	})

	t.Run("each missing assertion", func(t *testing.T) {
		_, err := b.Build("each", nil, map[string]any{})
		assertError(t, err)
	})

	t.Run("any", func(t *testing.T) {
		inner := &EqualAssertion{Expected: 2}
		a, err := b.Build("any", nil, map[string]any{"assertion": Assertion(inner)})
		assertNoError(t, err)
		assertNoError(t, a.Evaluate([]int{1, 2, 3}))
	})

	t.Run("any missing assertion", func(t *testing.T) {
		_, err := b.Build("any", nil, map[string]any{})
		assertError(t, err)
	})

	t.Run("all", func(t *testing.T) {
		inner := &GreaterThanAssertion{Expected: 0}
		a, err := b.Build("all", nil, map[string]any{"assertion": Assertion(inner)})
		assertNoError(t, err)
		assertNoError(t, a.Evaluate([]int{1, 2, 3}))
	})

	t.Run("all missing assertion", func(t *testing.T) {
		_, err := b.Build("all", nil, map[string]any{})
		assertError(t, err)
	})

	t.Run("unknown operator", func(t *testing.T) {
		_, err := b.Build("foobar", nil, nil)
		assertError(t, err)
		if !strings.Contains(err.Error(), "foobar") {
			t.Errorf("error should mention unknown operator, got %q", err.Error())
		}
	})
}

func TestAssertionError_IsError(t *testing.T) {
	var err error = &AssertionError{
		Assertion: "test",
		Message:   "test error",
	}
	var ae *AssertionError
	if !errors.As(err, &ae) {
		t.Error("expected errors.As to work with AssertionError")
	}
}

// test helpers

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func assertEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func assertAssertionError(t *testing.T, err error, assertionName string) *AssertionError {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ae *AssertionError
	if !errors.As(err, &ae) {
		t.Fatalf("expected AssertionError, got %T: %v", err, err)
	}
	if ae.Assertion != assertionName {
		t.Errorf("expected assertion name %q, got %q", assertionName, ae.Assertion)
	}
	return ae
}
