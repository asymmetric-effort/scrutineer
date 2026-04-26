package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/scrutineer/scrutineer/connector/ssh"
	"github.com/scrutineer/scrutineer/core/connector"
)

// Node describes a remote machine that participates in distributed load testing.
type Node struct {
	Host    string
	Port    int
	User    string
	KeyFile string
}

// DistributedConfig holds the configuration for a distributed load test.
type DistributedConfig struct {
	Nodes      []Node
	Binary     string // path to scrutineer binary on remote nodes
	TestConfig string // path to test config on remote nodes
}

// Validate checks that the distributed config is well-formed.
func (dc *DistributedConfig) Validate() error {
	if len(dc.Nodes) == 0 {
		return fmt.Errorf("distributed: no nodes configured")
	}
	if dc.Binary == "" {
		return fmt.Errorf("distributed: binary path is required")
	}
	if dc.TestConfig == "" {
		return fmt.Errorf("distributed: test config path is required")
	}
	for i, n := range dc.Nodes {
		if n.Host == "" {
			return fmt.Errorf("distributed: node %d: host is required", i)
		}
		if n.User == "" {
			return fmt.Errorf("distributed: node %d: user is required", i)
		}
		if n.KeyFile == "" {
			return fmt.Errorf("distributed: node %d: key_file is required", i)
		}
	}
	return nil
}

// SplitConcurrency distributes a total concurrency count across n nodes as
// evenly as possible. The first (concurrency % n) nodes each get one extra
// worker to absorb the remainder.
func SplitConcurrency(concurrency, n int) []int {
	if n <= 0 || concurrency <= 0 {
		return nil
	}
	base := concurrency / n
	remainder := concurrency % n
	splits := make([]int, n)
	for i := range n {
		splits[i] = base
		if i < remainder {
			splits[i]++
		}
	}
	return splits
}

// nodeExecutor is a function type for executing a command on a remote node.
// It is a package-level variable so tests can replace it.
var nodeExecutor = executeOnNode

// Distribute splits the given concurrency across the configured nodes and
// executes the load test on each node via SSH. It collects and returns
// results from all nodes.
func Distribute(ctx context.Context, cfg DistributedConfig, concurrency int) ([]*Results, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	splits := SplitConcurrency(concurrency, len(cfg.Nodes))
	results := make([]*Results, len(cfg.Nodes))
	errs := make([]error, len(cfg.Nodes))

	for i, node := range cfg.Nodes {
		cmd := buildRemoteCommand(cfg.Binary, cfg.TestConfig, splits[i])
		result, err := nodeExecutor(ctx, node, cmd)
		if err != nil {
			errs[i] = fmt.Errorf("node %s: %w", node.Host, err)
			continue
		}
		results[i] = result
	}

	// Check for errors.
	for _, err := range errs {
		if err != nil {
			return results, fmt.Errorf("distributed execution had errors: %w", err)
		}
	}

	return results, nil
}

// buildRemoteCommand constructs the command string to execute on a remote node.
func buildRemoteCommand(binary, testConfig string, concurrency int) string {
	return fmt.Sprintf("%s loadtest --config %s --concurrency %d --format json",
		binary, testConfig, concurrency)
}

// buildNodeConfig creates an SSH configuration map for connecting to a node.
func buildNodeConfig(node Node) map[string]any {
	port := node.Port
	if port == 0 {
		port = 22
	}
	return map[string]any{
		"host":           node.Host,
		"port":           port,
		"user":           node.User,
		"key_file":       node.KeyFile,
		"host_key_check": false,
	}
}

// buildExecStep creates a connector.Step for remote command execution.
func buildExecStep(cmd string) connector.Step {
	return connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": cmd,
		},
		Timeout: 30 * time.Minute,
	}
}

// extractResult extracts stdout, exit code, and stderr from a connector.Result
// and parses it into load test Results.
func extractResult(result *connector.Result) (*Results, error) {
	stdout, ok := result.Data["stdout"].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected stdout type")
	}

	exitCode, _ := result.Data["exit_code"].(int)
	stderr, _ := result.Data["stderr"].(string)

	return ParseNodeOutput(stdout, exitCode, stderr)
}

// newConnector creates a new SSH connector. It is a package-level variable
// so tests can replace it with a mock connector implementation.
var newConnector = func() connector.Connector {
	return ssh.New()
}

// executeOnNode connects to a node via SSH and runs the given command,
// parsing the JSON output into Results.
func executeOnNode(ctx context.Context, node Node, cmd string) (*Results, error) {
	conn := newConnector()

	config := buildNodeConfig(node)

	if err := conn.Setup(ctx, config); err != nil {
		return nil, fmt.Errorf("ssh setup: %w", err)
	}
	defer conn.Teardown(ctx)

	step := buildExecStep(cmd)

	result, err := conn.Execute(ctx, step)
	if err != nil {
		return nil, fmt.Errorf("ssh exec: %w", err)
	}

	return extractResult(result)
}

// ParseNodeOutput parses JSON output from a remote node into Results.
// Exported for testing and external use.
func ParseNodeOutput(stdout string, exitCode int, stderr string) (*Results, error) {
	if exitCode != 0 {
		return nil, fmt.Errorf("remote exited with code %s: %s",
			strconv.Itoa(exitCode), stderr)
	}

	var loadResults Results
	if err := json.Unmarshal([]byte(stdout), &loadResults); err != nil {
		return nil, fmt.Errorf("parse results: %w", err)
	}

	return &loadResults, nil
}

// AggregateResults combines results from multiple nodes into a single Results.
func AggregateResults(all []*Results, cfg Config) *Results {
	var totalRequests, successCount, errorCount int64
	var totalLatencySum int64
	var minLatency, maxLatency time.Duration
	errorSet := make(map[string]struct{})
	first := true

	for _, r := range all {
		if r == nil {
			continue
		}
		totalRequests += r.Metrics.TotalRequests
		successCount += r.Metrics.SuccessCount
		errorCount += r.Metrics.ErrorCount
		totalLatencySum += int64(r.Metrics.MeanLatency) * r.Metrics.TotalRequests

		if first || r.Metrics.MinLatency < minLatency {
			minLatency = r.Metrics.MinLatency
		}
		if first || r.Metrics.MaxLatency > maxLatency {
			maxLatency = r.Metrics.MaxLatency
		}
		first = false

		for _, e := range r.Errors {
			errorSet[e] = struct{}{}
		}
	}

	var meanLatency time.Duration
	if totalRequests > 0 {
		meanLatency = time.Duration(totalLatencySum / totalRequests)
	}

	var errs []string
	for e := range errorSet {
		errs = append(errs, e)
	}

	var elapsedTime time.Duration
	for _, r := range all {
		if r != nil && r.Metrics.ElapsedTime > elapsedTime {
			elapsedTime = r.Metrics.ElapsedTime
		}
	}

	var rps float64
	if elapsedTime > 0 {
		rps = float64(totalRequests) / elapsedTime.Seconds()
	}

	return &Results{
		Config: cfg,
		Metrics: MetricsSnapshot{
			TotalRequests:  totalRequests,
			SuccessCount:   successCount,
			ErrorCount:     errorCount,
			MeanLatency:    meanLatency,
			MinLatency:     minLatency,
			MaxLatency:     maxLatency,
			RequestsPerSec: rps,
			ElapsedTime:    elapsedTime,
		},
		Errors: errs,
	}
}
