package yaml

import (
	"testing"
)

func TestUnmarshalSimpleMap(t *testing.T) {
	input := []byte("name: Alice\nage: 30\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result["name"] != "Alice" {
		t.Errorf("name: expected 'Alice', got %v", result["name"])
	}
	if result["age"] != 30 {
		t.Errorf("age: expected 30, got %v (%T)", result["age"], result["age"])
	}
}

func TestUnmarshalNestedMap(t *testing.T) {
	input := []byte("db:\n  host: localhost\n  port: 5432\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	db, ok := result["db"].(map[string]any)
	if !ok {
		t.Fatalf("db: expected map, got %T", result["db"])
	}
	if db["host"] != "localhost" {
		t.Errorf("host: expected 'localhost', got %v", db["host"])
	}
	if db["port"] != 5432 {
		t.Errorf("port: expected 5432, got %v", db["port"])
	}
}

func TestUnmarshalSequence(t *testing.T) {
	input := []byte("- one\n- two\n- three\n")
	var result []any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}
	if result[0] != "one" {
		t.Errorf("item 0: expected 'one', got %v", result[0])
	}
}

func TestUnmarshalBooleans(t *testing.T) {
	input := []byte("a: true\nb: false\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result["a"] != true {
		t.Errorf("a: expected true, got %v", result["a"])
	}
	if result["b"] != false {
		t.Errorf("b: expected false, got %v", result["b"])
	}
}

func TestUnmarshalNull(t *testing.T) {
	input := []byte("a: null\nb:\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result["a"] != nil {
		t.Errorf("a: expected nil, got %v", result["a"])
	}
	if result["b"] != nil {
		t.Errorf("b: expected nil, got %v", result["b"])
	}
}

func TestUnmarshalFloats(t *testing.T) {
	input := []byte("pi: 3.14\nneg: -0.5\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	pi, ok := result["pi"].(float64)
	if !ok || pi < 3.13 || pi > 3.15 {
		t.Errorf("pi: expected 3.14, got %v", result["pi"])
	}
}

func TestUnmarshalFlowSequence(t *testing.T) {
	input := []byte("tags: [api, smoke]\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	tags, ok := result["tags"].([]any)
	if !ok {
		t.Fatalf("tags: expected []any, got %T", result["tags"])
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	if tags[0] != "api" || tags[1] != "smoke" {
		t.Errorf("tags: expected [api, smoke], got %v", tags)
	}
}

func TestUnmarshalFlowMapping(t *testing.T) {
	input := []byte("check: {equals: \"Alice\"}\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	check, ok := result["check"].(map[string]any)
	if !ok {
		t.Fatalf("check: expected map, got %T", result["check"])
	}
	if check["equals"] != "Alice" {
		t.Errorf("equals: expected 'Alice', got %v", check["equals"])
	}
}

func TestUnmarshalStruct(t *testing.T) {
	type Config struct {
		Name    string `yaml:"name"`
		Port    int    `yaml:"port"`
		Debug   bool   `yaml:"debug"`
		Version string `yaml:"version"`
	}
	input := []byte("name: myapp\nport: 8080\ndebug: true\nversion: 1.0\n")
	var cfg Config
	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "myapp" {
		t.Errorf("Name: expected 'myapp', got %q", cfg.Name)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port: expected 8080, got %d", cfg.Port)
	}
	if !cfg.Debug {
		t.Error("Debug: expected true")
	}
	if cfg.Version != "1.0" {
		t.Errorf("Version: expected '1.0', got %q", cfg.Version)
	}
}

func TestUnmarshalStructWithNestedMap(t *testing.T) {
	type Server struct {
		Host string         `yaml:"host"`
		Meta map[string]any `yaml:"meta"`
	}
	input := []byte("host: localhost\nmeta:\n  env: prod\n  count: 5\n")
	var srv Server
	if err := Unmarshal(input, &srv); err != nil {
		t.Fatal(err)
	}
	if srv.Host != "localhost" {
		t.Errorf("Host: expected 'localhost', got %q", srv.Host)
	}
	if srv.Meta["env"] != "prod" {
		t.Errorf("Meta.env: expected 'prod', got %v", srv.Meta["env"])
	}
}

func TestUnmarshalStructWithSlice(t *testing.T) {
	type Config struct {
		Items []any `yaml:"items"`
	}
	input := []byte("items:\n  - one\n  - 2\n  - true\n")
	var cfg Config
	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(cfg.Items))
	}
}

func TestUnmarshalStructTagSkip(t *testing.T) {
	type Config struct {
		Name   string `yaml:"name"`
		Hidden string `yaml:"-"`
	}
	input := []byte("name: test\nhidden: secret\n")
	var cfg Config
	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "test" {
		t.Errorf("Name: expected 'test', got %q", cfg.Name)
	}
	if cfg.Hidden != "" {
		t.Errorf("Hidden: expected empty, got %q", cfg.Hidden)
	}
}

func TestUnmarshalStructNoTag(t *testing.T) {
	type Config struct {
		Name string
	}
	input := []byte("name: test\n")
	var cfg Config
	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "test" {
		t.Errorf("Name: expected 'test', got %q", cfg.Name)
	}
}

func TestUnmarshalIntoWrongType(t *testing.T) {
	input := []byte("key: value\n")
	var s string
	err := Unmarshal(input, &s)
	if err == nil {
		t.Error("expected error unmarshaling map into string")
	}
}

func TestUnmarshalScalarIntoSlice(t *testing.T) {
	input := []byte("hello\n")
	var s []string
	err := Unmarshal(input, &s)
	if err == nil {
		t.Error("expected error unmarshaling scalar into slice")
	}
}

func TestUnmarshalScalarIntoMap(t *testing.T) {
	input := []byte("hello\n")
	var m map[string]any
	err := Unmarshal(input, &m)
	if err == nil {
		t.Error("expected error unmarshaling scalar into map")
	}
}

func TestUnmarshalNonPointer(t *testing.T) {
	input := []byte("key: value\n")
	var s string
	err := Unmarshal(input, s) // not a pointer
	if err == nil {
		t.Error("expected error for non-pointer")
	}
}

func TestUnmarshalIntoStringSlice(t *testing.T) {
	input := []byte("- hello\n- world\n")
	var result []string
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result[0] != "hello" || result[1] != "world" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestUnmarshalIntoFloat(t *testing.T) {
	input := []byte("3.14\n")
	var result float64
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result < 3.13 || result > 3.15 {
		t.Errorf("expected 3.14, got %f", result)
	}
}

func TestUnmarshalIntIntoFloat(t *testing.T) {
	input := []byte("42\n")
	var result float64
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result != 42.0 {
		t.Errorf("expected 42.0, got %f", result)
	}
}

func TestUnmarshalBoolIntoWrongType(t *testing.T) {
	input := []byte("true\n")
	var result int
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error unmarshaling bool into int")
	}
}

func TestUnmarshalStringIntoInt(t *testing.T) {
	input := []byte("hello\n")
	var result int
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error unmarshaling string into int")
	}
}

func TestUnmarshalStringIntoFloat(t *testing.T) {
	input := []byte("hello\n")
	var result float64
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error unmarshaling string into float")
	}
}

func TestUnmarshalStringIntoBool(t *testing.T) {
	input := []byte("hello\n")
	var result bool
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error unmarshaling non-bool string into bool")
	}
}

func TestUnmarshalSequenceIntoMap(t *testing.T) {
	input := []byte("- one\n- two\n")
	var result map[string]any
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error unmarshaling sequence into map")
	}
}

func TestUnmarshalMapIntoSlice(t *testing.T) {
	input := []byte("a: 1\nb: 2\n")
	var result []any
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error unmarshaling map into slice")
	}
}

func TestUnmarshalStructWithOmitEmpty(t *testing.T) {
	type Config struct {
		Name  string `yaml:"name,omitempty"`
		Value string `yaml:"value"`
	}
	input := []byte("name: test\nvalue: hello\n")
	var cfg Config
	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "test" || cfg.Value != "hello" {
		t.Errorf("unexpected: %+v", cfg)
	}
}

func TestUnmarshalNullIntoPtr(t *testing.T) {
	input := []byte("null\n")
	var result *string
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestUnmarshalIntoPtrField(t *testing.T) {
	type Config struct {
		Name *string `yaml:"name"`
	}
	input := []byte("name: hello\n")
	var cfg Config
	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Name == nil || *cfg.Name != "hello" {
		t.Errorf("expected 'hello', got %v", cfg.Name)
	}
}

// Integration test: full scrutineer test file
func TestUnmarshalFullScrutineerTestFile(t *testing.T) {
	input := []byte(`suite: "Example Tests"
tags: [api, smoke]

fixtures:
  user:
    name: "Alice"
    email: "alice@example.com"

tests:
  - name: "Create user"
    connector: http
    steps:
      - action: request
        method: POST
        path: /users
        body:
          name: "Alice"
        assert:
          - status: 201
          - body.name: {equals: "Alice"}
`)

	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}

	// Check suite name
	if result["suite"] != "Example Tests" {
		t.Errorf("suite: expected 'Example Tests', got %v", result["suite"])
	}

	// Check tags
	tags, ok := result["tags"].([]any)
	if !ok {
		t.Fatalf("tags: expected []any, got %T", result["tags"])
	}
	if len(tags) != 2 || tags[0] != "api" || tags[1] != "smoke" {
		t.Errorf("tags: expected [api, smoke], got %v", tags)
	}

	// Check fixtures
	fixtures, ok := result["fixtures"].(map[string]any)
	if !ok {
		t.Fatalf("fixtures: expected map, got %T", result["fixtures"])
	}
	user, ok := fixtures["user"].(map[string]any)
	if !ok {
		t.Fatalf("fixtures.user: expected map, got %T", fixtures["user"])
	}
	if user["name"] != "Alice" {
		t.Errorf("fixtures.user.name: expected 'Alice', got %v", user["name"])
	}
	if user["email"] != "alice@example.com" {
		t.Errorf("fixtures.user.email: expected 'alice@example.com', got %v", user["email"])
	}

	// Check tests
	tests, ok := result["tests"].([]any)
	if !ok {
		t.Fatalf("tests: expected []any, got %T", result["tests"])
	}
	if len(tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(tests))
	}

	test0, ok := tests[0].(map[string]any)
	if !ok {
		t.Fatalf("tests[0]: expected map, got %T", tests[0])
	}
	if test0["name"] != "Create user" {
		t.Errorf("tests[0].name: expected 'Create user', got %v", test0["name"])
	}
	if test0["connector"] != "http" {
		t.Errorf("tests[0].connector: expected 'http', got %v", test0["connector"])
	}

	steps, ok := test0["steps"].([]any)
	if !ok {
		t.Fatalf("tests[0].steps: expected []any, got %T", test0["steps"])
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}

	step0, ok := steps[0].(map[string]any)
	if !ok {
		t.Fatalf("steps[0]: expected map, got %T", steps[0])
	}
	if step0["action"] != "request" {
		t.Errorf("step.action: expected 'request', got %v", step0["action"])
	}
	if step0["method"] != "POST" {
		t.Errorf("step.method: expected 'POST', got %v", step0["method"])
	}
	if step0["path"] != "/users" {
		t.Errorf("step.path: expected '/users', got %v", step0["path"])
	}

	body, ok := step0["body"].(map[string]any)
	if !ok {
		t.Fatalf("step.body: expected map, got %T", step0["body"])
	}
	if body["name"] != "Alice" {
		t.Errorf("step.body.name: expected 'Alice', got %v", body["name"])
	}

	asserts, ok := step0["assert"].([]any)
	if !ok {
		t.Fatalf("step.assert: expected []any, got %T", step0["assert"])
	}
	if len(asserts) != 2 {
		t.Fatalf("expected 2 asserts, got %d", len(asserts))
	}

	assert0, ok := asserts[0].(map[string]any)
	if !ok {
		t.Fatalf("assert[0]: expected map, got %T", asserts[0])
	}
	if assert0["status"] != 201 {
		t.Errorf("assert[0].status: expected 201, got %v (%T)", assert0["status"], assert0["status"])
	}

	assert1, ok := asserts[1].(map[string]any)
	if !ok {
		t.Fatalf("assert[1]: expected map, got %T", asserts[1])
	}
	bodyName, ok := assert1["body.name"].(map[string]any)
	if !ok {
		t.Fatalf("assert[1][body.name]: expected map, got %T", assert1["body.name"])
	}
	if bodyName["equals"] != "Alice" {
		t.Errorf("assert equals: expected 'Alice', got %v", bodyName["equals"])
	}
}

func TestUnmarshalStructTypedTest(t *testing.T) {
	type Step struct {
		Action string `yaml:"action"`
		Method string `yaml:"method"`
		Path   string `yaml:"path"`
	}
	type Test struct {
		Name      string `yaml:"name"`
		Connector string `yaml:"connector"`
		Steps     []Step `yaml:"steps"`
	}
	type Suite struct {
		Suite string   `yaml:"suite"`
		Tags  []string `yaml:"tags"`
		Tests []Test   `yaml:"tests"`
	}

	input := []byte(`suite: "My Suite"
tags: [api, smoke]
tests:
  - name: "Test 1"
    connector: http
    steps:
      - action: request
        method: GET
        path: /health
`)

	var suite Suite
	if err := Unmarshal(input, &suite); err != nil {
		t.Fatal(err)
	}
	if suite.Suite != "My Suite" {
		t.Errorf("Suite: expected 'My Suite', got %q", suite.Suite)
	}
	if len(suite.Tags) != 2 || suite.Tags[0] != "api" {
		t.Errorf("Tags: expected [api, smoke], got %v", suite.Tags)
	}
	if len(suite.Tests) != 1 {
		t.Fatalf("Tests: expected 1, got %d", len(suite.Tests))
	}
	if suite.Tests[0].Name != "Test 1" {
		t.Errorf("Test.Name: expected 'Test 1', got %q", suite.Tests[0].Name)
	}
	if len(suite.Tests[0].Steps) != 1 {
		t.Fatalf("Steps: expected 1, got %d", len(suite.Tests[0].Steps))
	}
	step := suite.Tests[0].Steps[0]
	if step.Action != "request" || step.Method != "GET" || step.Path != "/health" {
		t.Errorf("Step: unexpected %+v", step)
	}
}

func TestUnmarshalNilPointer(t *testing.T) {
	err := Unmarshal([]byte("key: val\n"), (*map[string]any)(nil))
	if err == nil {
		t.Error("expected error for nil pointer")
	}
}

func TestUnmarshalFloatIntoInt(t *testing.T) {
	input := []byte("3.5\n")
	var result int
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result != 3 {
		t.Errorf("expected 3, got %d", result)
	}
}

func TestUnmarshalSequenceIntoStruct(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}
	input := []byte("- one\n- two\n")
	var cfg Config
	err := Unmarshal(input, &cfg)
	if err == nil {
		t.Error("expected error unmarshaling sequence into struct")
	}
}

func TestUnmarshalMappingIntoString(t *testing.T) {
	input := []byte("a: 1\n")
	var s string
	err := Unmarshal(input, &s)
	if err == nil {
		t.Error("expected error")
	}
}

func TestUnmarshalEmptyValue(t *testing.T) {
	input := []byte("key:\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result["key"] != nil {
		t.Errorf("expected nil for empty value, got %v (%T)", result["key"], result["key"])
	}
}

func TestUnmarshalLiteralBlock(t *testing.T) {
	input := []byte("desc: |\n  hello\n  world\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result["desc"] != "hello\nworld\n" {
		t.Errorf("expected 'hello\\nworld\\n', got %q", result["desc"])
	}
}

func TestUnmarshalFoldedBlock(t *testing.T) {
	input := []byte("desc: >\n  hello\n  world\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result["desc"] != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", result["desc"])
	}
}

func TestUnmarshalSequenceIntoInterface(t *testing.T) {
	input := []byte("items:\n  - 1\n  - two\n")
	var result map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	items, ok := result["items"].([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result["items"])
	}
	if items[0] != 1 || items[1] != "two" {
		t.Errorf("unexpected items: %v", items)
	}
}

func TestUnmarshalIntoMapStringString(t *testing.T) {
	input := []byte("a: hello\nb: world\n")
	var result map[string]string
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result["a"] != "hello" || result["b"] != "world" {
		t.Errorf("unexpected: %v", result)
	}
}

func TestUnmarshalIntoInt(t *testing.T) {
	input := []byte("42\n")
	var result int
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestUnmarshalIntoBool(t *testing.T) {
	input := []byte("true\n")
	var result bool
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("expected true")
	}
}

func TestUnmarshalIntoString(t *testing.T) {
	input := []byte("hello\n")
	var result string
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestUnmarshalNullIntoInterface(t *testing.T) {
	input := []byte("null\n")
	var result any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestUnmarshalMapIntoInterface(t *testing.T) {
	input := []byte("a: 1\n")
	var result any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["a"] != 1 {
		t.Errorf("expected a=1, got %v", m["a"])
	}
}

func TestUnmarshalSequenceIntoAny(t *testing.T) {
	input := []byte("- one\n- two\n")
	var result any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	s, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	if len(s) != 2 {
		t.Errorf("expected 2 items, got %d", len(s))
	}
}

func TestUnmarshalMappingIntoPtr(t *testing.T) {
	input := []byte("a: 1\n")
	var result *map[string]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil")
	}
	if (*result)["a"] != 1 {
		t.Errorf("expected a=1, got %v", (*result)["a"])
	}
}

func TestUnmarshalSequenceIntoPtr(t *testing.T) {
	input := []byte("- one\n")
	var result *[]any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result == nil || len(*result) != 1 {
		t.Error("unexpected result")
	}
}

func TestUnmarshalStructFieldError(t *testing.T) {
	type Config struct {
		Count int `yaml:"count"`
	}
	input := []byte("count: notanumber\n")
	var cfg Config
	err := Unmarshal(input, &cfg)
	if err == nil {
		t.Error("expected error decoding string into int field")
	}
}

func TestUnmarshalScalarIntoStruct(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}
	input := []byte("hello\n")
	var cfg Config
	err := Unmarshal(input, &cfg)
	if err == nil {
		t.Error("expected error unmarshaling scalar into struct")
	}
}

func TestUnmarshalMappingIntoInt(t *testing.T) {
	input := []byte("a: 1\n")
	var result int
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error")
	}
}

func TestUnmarshalSequenceIntoInt(t *testing.T) {
	input := []byte("- 1\n")
	var result int
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error")
	}
}

func TestUnmarshalSequenceIntoBool(t *testing.T) {
	input := []byte("- 1\n")
	var result bool
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error")
	}
}

func TestUnmarshalMappingIntoBool(t *testing.T) {
	input := []byte("a: 1\n")
	var result bool
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error")
	}
}

func TestUnmarshalMappingIntoFloat(t *testing.T) {
	input := []byte("a: 1\n")
	var result float64
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error")
	}
}

func TestUnmarshalSequenceIntoFloat(t *testing.T) {
	input := []byte("- 1\n")
	var result float64
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error")
	}
}

func TestUnmarshalSequenceIntoString(t *testing.T) {
	input := []byte("- 1\n")
	var result string
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error")
	}
}

func TestUnmarshalMappingIntoMapStringInt(t *testing.T) {
	input := []byte("a: 1\nb: 2\n")
	var result map[string]int
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result["a"] != 1 || result["b"] != 2 {
		t.Errorf("unexpected: %v", result)
	}
}

func TestUnmarshalMappingIntoMapStringIntError(t *testing.T) {
	input := []byte("a: notnum\n")
	var result map[string]int
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error decoding string into int map value")
	}
}

func TestUnmarshalNullMappingIntoInterface(t *testing.T) {
	// Mapping node that has all null values, decoded into interface
	input := []byte("a:\nb:\n")
	var result any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["a"] != nil || m["b"] != nil {
		t.Errorf("expected nil values, got %v", m)
	}
}

func TestUnmarshalParseError(t *testing.T) {
	input := []byte(`"unterminated`)
	var result any
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestUnmarshalDecodeNodeError(t *testing.T) {
	// Pass non-pointer
	input := []byte("a: 1\n")
	err := Unmarshal(input, 42)
	if err == nil {
		t.Error("expected error for non-pointer int")
	}
}

func TestUnmarshalSequenceIntoSliceError(t *testing.T) {
	// Sequence of maps into []int
	input := []byte("- a: 1\n")
	var result []int
	err := Unmarshal(input, &result)
	if err == nil {
		t.Error("expected error decoding map into int slice element")
	}
}

func TestUnmarshalNullIntoScalarInterface(t *testing.T) {
	input := []byte("~\n")
	var result any
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestUnmarshalEmptySequenceIntoSlice(t *testing.T) {
	input := []byte("items: []\n")
	type Config struct {
		Items []string `yaml:"items"`
	}
	var cfg Config
	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Items) != 0 {
		t.Errorf("expected empty slice, got %v", cfg.Items)
	}
}

func TestUnmarshalStructNestedStruct(t *testing.T) {
	type Inner struct {
		Value string `yaml:"value"`
	}
	type Outer struct {
		Inner Inner `yaml:"inner"`
	}
	input := []byte("inner:\n  value: test\n")
	var result Outer
	if err := Unmarshal(input, &result); err != nil {
		t.Fatal(err)
	}
	if result.Inner.Value != "test" {
		t.Errorf("expected 'test', got %q", result.Inner.Value)
	}
}

func TestUnmarshalStructUnexportedField(t *testing.T) {
	type Config struct {
		Name    string `yaml:"name"`
		hidden  string //nolint:unused
	}
	input := []byte("name: test\nhidden: secret\n")
	var cfg Config
	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "test" {
		t.Errorf("expected 'test', got %q", cfg.Name)
	}
	_ = cfg.hidden
}

func TestUnmarshalStructUnknownField(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}
	input := []byte("name: test\nunknown: ignored\n")
	var cfg Config
	if err := Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "test" {
		t.Errorf("expected 'test', got %q", cfg.Name)
	}
}
