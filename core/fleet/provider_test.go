package fleet

import (
	"context"
	"testing"
	"time"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	name       string
	setupErr   error
	acquireN   int
	acquireErr error
	execResult *ExecResult
	execErr    error
	releaseErr error
	tearErr    error

	setupCalls   int
	acquireCalls int
	execCalls    int
	releaseCalls int
	tearCalls    int
	released     []Host
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Setup(_ context.Context, _ map[string]any) error {
	m.setupCalls++
	return m.setupErr
}

func (m *mockProvider) Acquire(_ context.Context, n int) ([]Host, error) {
	m.acquireCalls++
	if m.acquireErr != nil {
		return nil, m.acquireErr
	}
	m.acquireN = n
	hosts := make([]Host, n)
	for i := 0; i < n; i++ {
		hosts[i] = Host{
			ID:       m.name + "-host-" + string(rune('0'+i)),
			Address:  "10.0.0." + string(rune('1'+i)),
			Provider: m.name,
			BornAt:   time.Now(),
		}
	}
	return hosts, nil
}

func (m *mockProvider) Push(_ context.Context, _ Host, _ []string) error {
	return nil
}

func (m *mockProvider) Execute(_ context.Context, _ Host, _ string) (*ExecResult, error) {
	m.execCalls++
	if m.execErr != nil {
		return nil, m.execErr
	}
	if m.execResult != nil {
		return m.execResult, nil
	}
	return &ExecResult{Stdout: "ok", ExitCode: 0}, nil
}

func (m *mockProvider) Release(_ context.Context, hosts []Host) error {
	m.releaseCalls++
	m.released = append(m.released, hosts...)
	return m.releaseErr
}

func (m *mockProvider) Teardown(_ context.Context) error {
	m.tearCalls++
	return m.tearErr
}

func TestHostFields(t *testing.T) {
	h := Host{
		ID:       "h1",
		Address:  "10.0.0.1",
		Provider: "static",
		Meta:     map[string]string{"region": "us-east-1"},
		BornAt:   time.Now(),
	}
	if h.ID != "h1" || h.Address != "10.0.0.1" || h.Provider != "static" {
		t.Errorf("host fields incorrect: %+v", h)
	}
	if h.Meta["region"] != "us-east-1" {
		t.Errorf("meta: %v", h.Meta)
	}
}

func TestExecResultFields(t *testing.T) {
	r := ExecResult{
		Stdout:   "output",
		Stderr:   "err",
		ExitCode: 1,
		Elapsed:  100 * time.Millisecond,
	}
	if r.Stdout != "output" || r.Stderr != "err" || r.ExitCode != 1 {
		t.Errorf("result fields incorrect: %+v", r)
	}
}

func TestMockProviderSatisfiesInterface(t *testing.T) {
	var _ Provider = &mockProvider{name: "test"}
}
