package expression

import (
	"sort"
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	fn := func(args []any) (any, error) { return "ok", nil }
	if err := r.Register("test_fn", fn); err != nil {
		t.Fatal(err)
	}
	got, ok := r.Get("test_fn")
	if !ok {
		t.Fatal("expected function to be found")
	}
	val, err := got(nil)
	if err != nil || val != "ok" {
		t.Errorf("got %v, %v", val, err)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	r := NewRegistry()
	fn := func(args []any) (any, error) { return nil, nil }
	_ = r.Register("dup", fn)
	err := r.Register("dup", fn)
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestGetUnknown(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("missing")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestNames(t *testing.T) {
	r := NewRegistry()
	_ = r.Register("beta", func(args []any) (any, error) { return nil, nil })
	_ = r.Register("alpha", func(args []any) (any, error) { return nil, nil })
	names := r.Names()
	sort.Strings(names)
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("names = %v", names)
	}
}

func TestDefaultRegistryHasAllBuiltins(t *testing.T) {
	r := DefaultRegistry()
	expected := []string{
		"random_string", "uuid", "upper", "lower", "trim", "concat",
		"substring", "replace", "length",
		"random_int", "random_float", "abs", "ceil", "floor", "round",
		"min", "max", "mod",
		"now", "now_unix", "now_iso", "now_rfc3339", "format_time", "add_duration",
		"base64_encode", "base64_decode", "url_encode", "url_decode", "json_encode",
		"md5", "sha256", "sha512", "hmac_sha256",
		"env", "env_or",
		"db_query", "db_query_one", "db_count",
	}
	for _, name := range expected {
		if _, ok := r.Get(name); !ok {
			t.Errorf("missing builtin: %s", name)
		}
	}
}
