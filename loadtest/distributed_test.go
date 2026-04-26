package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
)

func TestDistributedConfig_Validate_NoNodes(t *testing.T) {
	cfg := &DistributedConfig{
		Binary:     "/usr/bin/scrutineer",
		TestConfig: "/etc/scrutineer/test.yaml",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for no nodes")
	}
	if err.Error() != "distributed: no nodes configured" {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestDistributedConfig_Validate_NoBinary(t *testing.T) {
	cfg := &DistributedConfig{
		Nodes:      []Node{{Host: "h", User: "u", KeyFile: "k"}},
		TestConfig: "/etc/scrutineer/test.yaml",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for no binary")
	}
	if err.Error() != "distributed: binary path is required" {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestDistributedConfig_Validate_NoTestConfig(t *testing.T) {
	cfg := &DistributedConfig{
		Nodes:  []Node{{Host: "h", User: "u", KeyFile: "k"}},
		Binary: "/usr/bin/scrutineer",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for no test config")
	}
	if err.Error() != "distributed: test config path is required" {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestDistributedConfig_Validate_MissingHost(t *testing.T) {
	cfg := &DistributedConfig{
		Nodes:      []Node{{User: "u", KeyFile: "k"}},
		Binary:     "/usr/bin/scrutineer",
		TestConfig: "/etc/scrutineer/test.yaml",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing host")
	}
}

func TestDistributedConfig_Validate_MissingUser(t *testing.T) {
	cfg := &DistributedConfig{
		Nodes:      []Node{{Host: "h", KeyFile: "k"}},
		Binary:     "/usr/bin/scrutineer",
		TestConfig: "/etc/scrutineer/test.yaml",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestDistributedConfig_Validate_MissingKeyFile(t *testing.T) {
	cfg := &DistributedConfig{
		Nodes:      []Node{{Host: "h", User: "u"}},
		Binary:     "/usr/bin/scrutineer",
		TestConfig: "/etc/scrutineer/test.yaml",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
}

func TestDistributedConfig_Validate_Valid(t *testing.T) {
	cfg := &DistributedConfig{
		Nodes: []Node{
			{Host: "node1", User: "admin", KeyFile: "/key"},
			{Host: "node2", User: "admin", KeyFile: "/key"},
		},
		Binary:     "/usr/bin/scrutineer",
		TestConfig: "/etc/scrutineer/test.yaml",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %s", err)
	}
}

func TestSplitConcurrency_Even(t *testing.T) {
	splits := SplitConcurrency(10, 2)
	expected := []int{5, 5}
	assertIntSlice(t, expected, splits)
}

func TestSplitConcurrency_Uneven(t *testing.T) {
	splits := SplitConcurrency(10, 3)
	expected := []int{4, 3, 3}
	assertIntSlice(t, expected, splits)
}

func TestSplitConcurrency_OneNode(t *testing.T) {
	splits := SplitConcurrency(10, 1)
	expected := []int{10}
	assertIntSlice(t, expected, splits)
}

func TestSplitConcurrency_MoreNodesThanConcurrency(t *testing.T) {
	splits := SplitConcurrency(2, 5)
	expected := []int{1, 1, 0, 0, 0}
	assertIntSlice(t, expected, splits)
}

func TestSplitConcurrency_ZeroConcurrency(t *testing.T) {
	splits := SplitConcurrency(0, 3)
	if splits != nil {
		t.Errorf("expected nil, got %v", splits)
	}
}

func TestSplitConcurrency_ZeroNodes(t *testing.T) {
	splits := SplitConcurrency(10, 0)
	if splits != nil {
		t.Errorf("expected nil, got %v", splits)
	}
}

func TestSplitConcurrency_OneEach(t *testing.T) {
	splits := SplitConcurrency(3, 3)
	expected := []int{1, 1, 1}
	assertIntSlice(t, expected, splits)
}

func TestSplitConcurrency_SevenAcrossThree(t *testing.T) {
	splits := SplitConcurrency(7, 3)
	expected := []int{3, 2, 2}
	assertIntSlice(t, expected, splits)

	total := 0
	for _, s := range splits {
		total += s
	}
	if total != 7 {
		t.Errorf("sum of splits %d != 7", total)
	}
}

func TestBuildRemoteCommand(t *testing.T) {
	cmd := buildRemoteCommand("/usr/bin/scrutineer", "/etc/test.yaml", 5)
	expected := "/usr/bin/scrutineer loadtest --config /etc/test.yaml --concurrency 5 --format json"
	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestDistribute_ValidationFails(t *testing.T) {
	cfg := DistributedConfig{}
	_, err := Distribute(context.Background(), cfg, 10)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDistribute_MockSuccess(t *testing.T) {
	// Save and restore the original executor.
	orig := nodeExecutor
	defer func() { nodeExecutor = orig }()

	nodeExecutor = func(_ context.Context, node Node, _ string) (*Results, error) {
		return &Results{
			Metrics: MetricsSnapshot{
				TotalRequests: 50,
				SuccessCount:  50,
				MeanLatency:   10 * time.Millisecond,
				MinLatency:    5 * time.Millisecond,
				MaxLatency:    20 * time.Millisecond,
				ElapsedTime:   time.Second,
			},
		}, nil
	}

	cfg := DistributedConfig{
		Nodes: []Node{
			{Host: "node1", Port: 22, User: "admin", KeyFile: "/key"},
			{Host: "node2", Port: 22, User: "admin", KeyFile: "/key"},
		},
		Binary:     "/usr/bin/scrutineer",
		TestConfig: "/etc/test.yaml",
	}

	results, err := Distribute(context.Background(), cfg, 10)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if r == nil {
			t.Errorf("result %d is nil", i)
			continue
		}
		if r.Metrics.TotalRequests != 50 {
			t.Errorf("result %d: expected 50 requests, got %d", i, r.Metrics.TotalRequests)
		}
	}
}

func TestDistribute_MockPartialFailure(t *testing.T) {
	orig := nodeExecutor
	defer func() { nodeExecutor = orig }()

	nodeExecutor = func(_ context.Context, node Node, _ string) (*Results, error) {
		if node.Host == "node2" {
			return nil, fmt.Errorf("connection refused")
		}
		return &Results{
			Metrics: MetricsSnapshot{TotalRequests: 50, SuccessCount: 50},
		}, nil
	}

	cfg := DistributedConfig{
		Nodes: []Node{
			{Host: "node1", Port: 22, User: "admin", KeyFile: "/key"},
			{Host: "node2", Port: 22, User: "admin", KeyFile: "/key"},
		},
		Binary:     "/usr/bin/scrutineer",
		TestConfig: "/etc/test.yaml",
	}

	results, err := Distribute(context.Background(), cfg, 10)
	if err == nil {
		t.Fatal("expected error for partial failure")
	}
	// First result should be present.
	if results[0] == nil {
		t.Error("expected first result to be non-nil")
	}
	// Second result should be nil (failed).
	if results[1] != nil {
		t.Error("expected second result to be nil")
	}
}

func TestDistribute_MockAllFail(t *testing.T) {
	orig := nodeExecutor
	defer func() { nodeExecutor = orig }()

	nodeExecutor = func(_ context.Context, node Node, _ string) (*Results, error) {
		return nil, fmt.Errorf("unreachable")
	}

	cfg := DistributedConfig{
		Nodes: []Node{
			{Host: "node1", Port: 22, User: "admin", KeyFile: "/key"},
		},
		Binary:     "/usr/bin/scrutineer",
		TestConfig: "/etc/test.yaml",
	}

	_, err := Distribute(context.Background(), cfg, 5)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseNodeOutput_Success(t *testing.T) {
	r := &Results{
		Config: Config{Concurrency: 5, Duration: time.Second},
		Metrics: MetricsSnapshot{
			TotalRequests: 100,
			SuccessCount:  100,
		},
	}
	data, _ := json.Marshal(r)

	parsed, err := ParseNodeOutput(string(data), 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if parsed.Metrics.TotalRequests != 100 {
		t.Errorf("expected 100 requests, got %d", parsed.Metrics.TotalRequests)
	}
}

func TestParseNodeOutput_NonZeroExit(t *testing.T) {
	_, err := ParseNodeOutput("", 1, "something failed")
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}
	if err.Error() != "remote exited with code 1: something failed" {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestParseNodeOutput_InvalidJSON(t *testing.T) {
	_, err := ParseNodeOutput("not json", 0, "")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAggregateResults_Empty(t *testing.T) {
	cfg := Config{Concurrency: 10, Duration: time.Second}
	result := AggregateResults(nil, cfg)

	if result.Metrics.TotalRequests != 0 {
		t.Errorf("expected 0 requests, got %d", result.Metrics.TotalRequests)
	}
	if result.Config.Concurrency != 10 {
		t.Errorf("expected concurrency 10, got %d", result.Config.Concurrency)
	}
}

func TestAggregateResults_MultipleNodes(t *testing.T) {
	cfg := Config{Concurrency: 6, Duration: time.Second}

	r1 := &Results{
		Metrics: MetricsSnapshot{
			TotalRequests: 100,
			SuccessCount:  90,
			ErrorCount:    10,
			MeanLatency:   50 * time.Millisecond,
			MinLatency:    10 * time.Millisecond,
			MaxLatency:    200 * time.Millisecond,
			ElapsedTime:   time.Second,
		},
		Errors: []string{"err1"},
	}
	r2 := &Results{
		Metrics: MetricsSnapshot{
			TotalRequests: 200,
			SuccessCount:  195,
			ErrorCount:    5,
			MeanLatency:   30 * time.Millisecond,
			MinLatency:    5 * time.Millisecond,
			MaxLatency:    150 * time.Millisecond,
			ElapsedTime:   time.Second,
		},
		Errors: []string{"err2"},
	}

	agg := AggregateResults([]*Results{r1, r2}, cfg)

	if agg.Metrics.TotalRequests != 300 {
		t.Errorf("expected 300 total, got %d", agg.Metrics.TotalRequests)
	}
	if agg.Metrics.SuccessCount != 285 {
		t.Errorf("expected 285 successes, got %d", agg.Metrics.SuccessCount)
	}
	if agg.Metrics.ErrorCount != 15 {
		t.Errorf("expected 15 errors, got %d", agg.Metrics.ErrorCount)
	}
	if agg.Metrics.MinLatency != 5*time.Millisecond {
		t.Errorf("expected min 5ms, got %s", agg.Metrics.MinLatency)
	}
	if agg.Metrics.MaxLatency != 200*time.Millisecond {
		t.Errorf("expected max 200ms, got %s", agg.Metrics.MaxLatency)
	}
	if len(agg.Errors) != 2 {
		t.Errorf("expected 2 unique errors, got %d", len(agg.Errors))
	}

	// Weighted mean: (50*100 + 30*200) / 300 = 11000/300 = 36.666ms
	expectedMean := time.Duration(11000 * int64(time.Millisecond) / 300)
	if agg.Metrics.MeanLatency != expectedMean {
		t.Errorf("expected mean %s, got %s", expectedMean, agg.Metrics.MeanLatency)
	}
}

func TestAggregateResults_WithNilEntries(t *testing.T) {
	cfg := Config{Concurrency: 4, Duration: time.Second}

	r1 := &Results{
		Metrics: MetricsSnapshot{
			TotalRequests: 50,
			SuccessCount:  50,
			MeanLatency:   10 * time.Millisecond,
			MinLatency:    5 * time.Millisecond,
			MaxLatency:    20 * time.Millisecond,
			ElapsedTime:   time.Second,
		},
	}

	agg := AggregateResults([]*Results{nil, r1, nil}, cfg)
	if agg.Metrics.TotalRequests != 50 {
		t.Errorf("expected 50, got %d", agg.Metrics.TotalRequests)
	}
}

func TestAggregateResults_RequestsPerSec(t *testing.T) {
	cfg := Config{Concurrency: 2, Duration: time.Second}
	r := &Results{
		Metrics: MetricsSnapshot{
			TotalRequests: 100,
			SuccessCount:  100,
			MeanLatency:   10 * time.Millisecond,
			MinLatency:    5 * time.Millisecond,
			MaxLatency:    20 * time.Millisecond,
			ElapsedTime:   time.Second,
		},
	}
	agg := AggregateResults([]*Results{r}, cfg)
	if agg.Metrics.RequestsPerSec != 100.0 {
		t.Errorf("expected 100 rps, got %.2f", agg.Metrics.RequestsPerSec)
	}
}

func TestAggregateResults_DuplicateErrors(t *testing.T) {
	cfg := Config{Concurrency: 2, Duration: time.Second}
	r1 := &Results{
		Metrics: MetricsSnapshot{TotalRequests: 10, SuccessCount: 5, ErrorCount: 5, MeanLatency: time.Millisecond, MinLatency: time.Millisecond, MaxLatency: time.Millisecond, ElapsedTime: time.Second},
		Errors:  []string{"common error", "err1"},
	}
	r2 := &Results{
		Metrics: MetricsSnapshot{TotalRequests: 10, SuccessCount: 5, ErrorCount: 5, MeanLatency: time.Millisecond, MinLatency: time.Millisecond, MaxLatency: time.Millisecond, ElapsedTime: time.Second},
		Errors:  []string{"common error", "err2"},
	}
	agg := AggregateResults([]*Results{r1, r2}, cfg)
	if len(agg.Errors) != 3 {
		t.Errorf("expected 3 unique errors, got %d: %v", len(agg.Errors), agg.Errors)
	}
}

// mockConnector implements connector.Connector for testing executeOnNode.
type mockConnector struct {
	setupErr   error
	execResult *connector.Result
	execErr    error
}

func (m *mockConnector) Name() string { return "mock" }

func (m *mockConnector) Setup(_ context.Context, _ map[string]any) error {
	return m.setupErr
}

func (m *mockConnector) Execute(_ context.Context, _ connector.Step) (*connector.Result, error) {
	return m.execResult, m.execErr
}

func (m *mockConnector) Teardown(_ context.Context) error {
	return nil
}

func TestExecuteOnNode_Success(t *testing.T) {
	origConn := newConnector
	defer func() { newConnector = origConn }()

	r := &Results{Metrics: MetricsSnapshot{TotalRequests: 77}}
	data, _ := json.Marshal(r)

	newConnector = func() connector.Connector {
		return &mockConnector{
			execResult: &connector.Result{
				Data: map[string]any{
					"stdout":    string(data),
					"stderr":    "",
					"exit_code": 0,
				},
			},
		}
	}

	node := Node{Host: "test", User: "admin", KeyFile: "/key"}
	result, err := executeOnNode(context.Background(), node, "echo test")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if result.Metrics.TotalRequests != 77 {
		t.Errorf("expected 77, got %d", result.Metrics.TotalRequests)
	}
}

func TestExecuteOnNode_SetupError(t *testing.T) {
	origConn := newConnector
	defer func() { newConnector = origConn }()

	newConnector = func() connector.Connector {
		return &mockConnector{setupErr: fmt.Errorf("connection refused")}
	}

	node := Node{Host: "test", User: "admin", KeyFile: "/key"}
	_, err := executeOnNode(context.Background(), node, "echo test")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecuteOnNode_ExecError(t *testing.T) {
	origConn := newConnector
	defer func() { newConnector = origConn }()

	newConnector = func() connector.Connector {
		return &mockConnector{execErr: fmt.Errorf("command failed")}
	}

	node := Node{Host: "test", User: "admin", KeyFile: "/key"}
	_, err := executeOnNode(context.Background(), node, "echo test")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecuteOnNode_BadResult(t *testing.T) {
	origConn := newConnector
	defer func() { newConnector = origConn }()

	newConnector = func() connector.Connector {
		return &mockConnector{
			execResult: &connector.Result{
				Data: map[string]any{
					"stdout":    "not json",
					"stderr":    "",
					"exit_code": 0,
				},
			},
		}
	}

	node := Node{Host: "test", User: "admin", KeyFile: "/key"}
	_, err := executeOnNode(context.Background(), node, "echo test")
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestBuildNodeConfig_DefaultPort(t *testing.T) {
	node := Node{Host: "myhost", User: "admin", KeyFile: "/key"}
	cfg := buildNodeConfig(node)

	if cfg["host"] != "myhost" {
		t.Errorf("expected host myhost, got %v", cfg["host"])
	}
	if cfg["port"] != 22 {
		t.Errorf("expected default port 22, got %v", cfg["port"])
	}
	if cfg["user"] != "admin" {
		t.Errorf("expected user admin, got %v", cfg["user"])
	}
	if cfg["key_file"] != "/key" {
		t.Errorf("expected key_file /key, got %v", cfg["key_file"])
	}
	if cfg["host_key_check"] != false {
		t.Errorf("expected host_key_check false")
	}
}

func TestBuildNodeConfig_CustomPort(t *testing.T) {
	node := Node{Host: "myhost", Port: 2222, User: "admin", KeyFile: "/key"}
	cfg := buildNodeConfig(node)
	if cfg["port"] != 2222 {
		t.Errorf("expected port 2222, got %v", cfg["port"])
	}
}

func TestBuildExecStep(t *testing.T) {
	step := buildExecStep("echo hello")
	if step.Action != "exec" {
		t.Errorf("expected action exec, got %s", step.Action)
	}
	cmd, ok := step.Parameters["command"].(string)
	if !ok || cmd != "echo hello" {
		t.Errorf("expected command 'echo hello', got %v", step.Parameters["command"])
	}
	if step.Timeout != 30*time.Minute {
		t.Errorf("expected 30m timeout, got %s", step.Timeout)
	}
}

func TestExtractResult_Success(t *testing.T) {
	r := &Results{
		Metrics: MetricsSnapshot{TotalRequests: 42},
	}
	data, _ := json.Marshal(r)

	connResult := &connector.Result{
		Data: map[string]any{
			"stdout":    string(data),
			"stderr":    "",
			"exit_code": 0,
		},
	}

	parsed, err := extractResult(connResult)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if parsed.Metrics.TotalRequests != 42 {
		t.Errorf("expected 42, got %d", parsed.Metrics.TotalRequests)
	}
}

func TestExtractResult_BadStdoutType(t *testing.T) {
	connResult := &connector.Result{
		Data: map[string]any{
			"stdout":    123, // not a string
			"exit_code": 0,
		},
	}
	_, err := extractResult(connResult)
	if err == nil {
		t.Fatal("expected error for non-string stdout")
	}
}

func TestExtractResult_NonZeroExit(t *testing.T) {
	connResult := &connector.Result{
		Data: map[string]any{
			"stdout":    "",
			"stderr":    "failed",
			"exit_code": 1,
		},
	}
	_, err := extractResult(connResult)
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}
}

func TestExtractResult_InvalidJSON(t *testing.T) {
	connResult := &connector.Result{
		Data: map[string]any{
			"stdout":    "not json",
			"stderr":    "",
			"exit_code": 0,
		},
	}
	_, err := extractResult(connResult)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func assertIntSlice(t *testing.T, expected, got []int) {
	t.Helper()
	if len(expected) != len(got) {
		t.Fatalf("length mismatch: expected %d, got %d", len(expected), len(got))
	}
	for i := range expected {
		if expected[i] != got[i] {
			t.Errorf("index %d: expected %d, got %d", i, expected[i], got[i])
		}
	}
}
