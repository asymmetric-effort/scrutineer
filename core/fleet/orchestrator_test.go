package fleet

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/core/schema"
)

func testOrchestratorConfig() schema.FleetConfig {
	return schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "mock1", Weight: 60, TTL: 0},
			{Provider: "mock2", Weight: 40, TTL: 0},
		},
	}
}

func testRegistry() *Registry {
	r := NewRegistry()
	_ = r.Register("mock1", func() Provider { return &mockProvider{name: "mock1"} })
	_ = r.Register("mock2", func() Provider { return &mockProvider{name: "mock2"} })
	return r
}

func TestOrchestratorSetup(t *testing.T) {
	reg := testRegistry()
	cfg := testOrchestratorConfig()
	orch := NewOrchestrator(reg, cfg)

	if err := orch.Setup(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(orch.providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(orch.providers))
	}
}

func TestOrchestratorSetupProviderNotFound(t *testing.T) {
	reg := NewRegistry()
	cfg := schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "unknown", Weight: 100},
		},
	}
	orch := NewOrchestrator(reg, cfg)
	err := orch.Setup(context.Background())
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestOrchestratorSetupProviderError(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register("fail", func() Provider {
		return &mockProvider{name: "fail", setupErr: fmt.Errorf("setup failed")}
	})
	cfg := schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "fail", Weight: 100},
		},
	}
	orch := NewOrchestrator(reg, cfg)
	err := orch.Setup(context.Background())
	if err == nil {
		t.Fatal("expected setup error")
	}
}

func TestOrchestratorSelectProvider(t *testing.T) {
	reg := testRegistry()
	cfg := testOrchestratorConfig()
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())

	counts := make(map[string]int)
	for i := 0; i < 10000; i++ {
		name := orch.SelectProvider()
		counts[name]++
	}
	ratio := float64(counts["mock1"]) / 10000.0
	if ratio < 0.5 || ratio > 0.7 {
		t.Errorf("mock1 selected %.1f%% (expected ~60%%)", ratio*100)
	}
}

func TestOrchestratorSelectHost(t *testing.T) {
	reg := testRegistry()
	cfg := testOrchestratorConfig()
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())

	orch.AddHosts("mock1", []Host{
		{ID: "h1", Address: "10.0.0.1", Provider: "mock1"},
		{ID: "h2", Address: "10.0.0.2", Provider: "mock1"},
	})

	h1, err := orch.SelectHost("mock1")
	if err != nil {
		t.Fatal(err)
	}
	if h1.ID != "h1" {
		t.Errorf("first host = %q, want h1", h1.ID)
	}

	h2, err := orch.SelectHost("mock1")
	if err != nil {
		t.Fatal(err)
	}
	if h2.ID != "h2" {
		t.Errorf("second host = %q, want h2", h2.ID)
	}

	// Round-robin wraps around.
	h3, err := orch.SelectHost("mock1")
	if err != nil {
		t.Fatal(err)
	}
	if h3.ID != "h1" {
		t.Errorf("third host = %q, want h1 (round-robin)", h3.ID)
	}
}

func TestOrchestratorSelectHostNoHosts(t *testing.T) {
	reg := testRegistry()
	cfg := testOrchestratorConfig()
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())

	_, err := orch.SelectHost("mock1")
	if err == nil {
		t.Fatal("expected error for no hosts")
	}
}

func TestOrchestratorExecute(t *testing.T) {
	reg := testRegistry()
	cfg := testOrchestratorConfig()
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())

	host := Host{ID: "h1", Provider: "mock1"}
	result, err := orch.Execute(context.Background(), host, "echo hi")
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "ok" {
		t.Errorf("stdout = %q", result.Stdout)
	}
}

func TestOrchestratorExecuteUnknownProvider(t *testing.T) {
	orch := NewOrchestrator(NewRegistry(), schema.FleetConfig{})
	host := Host{ID: "h1", Provider: "unknown"}
	_, err := orch.Execute(context.Background(), host, "cmd")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOrchestratorCheckTTL(t *testing.T) {
	reg := NewRegistry()
	mock := &mockProvider{name: "ttl_test"}
	_ = reg.Register("ttl_test", func() Provider { return mock })

	cfg := schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "ttl_test", Weight: 100, TTL: 1},
		},
	}
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())

	// Add a host that was born 2 minutes ago (expired).
	expired := Host{
		ID:       "old",
		Provider: "ttl_test",
		BornAt:   time.Now().Add(-2 * time.Minute),
	}
	orch.AddHosts("ttl_test", []Host{expired})

	err := orch.CheckTTL(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Expired host should have been released and replaced.
	if mock.releaseCalls != 1 {
		t.Errorf("release calls = %d, want 1", mock.releaseCalls)
	}
	if mock.acquireCalls != 1 {
		t.Errorf("acquire calls = %d, want 1", mock.acquireCalls)
	}
}

func TestOrchestratorCheckTTLNotExpired(t *testing.T) {
	reg := NewRegistry()
	mock := &mockProvider{name: "fresh"}
	_ = reg.Register("fresh", func() Provider { return mock })

	cfg := schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "fresh", Weight: 100, TTL: 60},
		},
	}
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())

	fresh := Host{
		ID:       "new",
		Provider: "fresh",
		BornAt:   time.Now(),
	}
	orch.AddHosts("fresh", []Host{fresh})

	err := orch.CheckTTL(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if mock.releaseCalls != 0 {
		t.Errorf("release calls = %d, want 0", mock.releaseCalls)
	}
}

func TestOrchestratorCheckTTLZero(t *testing.T) {
	reg := NewRegistry()
	mock := &mockProvider{name: "notl"}
	_ = reg.Register("notl", func() Provider { return mock })

	cfg := schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "notl", Weight: 100, TTL: 0},
		},
	}
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())

	old := Host{
		ID:       "old",
		Provider: "notl",
		BornAt:   time.Now().Add(-24 * time.Hour),
	}
	orch.AddHosts("notl", []Host{old})

	err := orch.CheckTTL(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// TTL=0 means no expiry — nothing should be released.
	if mock.releaseCalls != 0 {
		t.Errorf("release calls = %d, want 0", mock.releaseCalls)
	}
}

func TestOrchestratorTeardown(t *testing.T) {
	reg := NewRegistry()
	mock := &mockProvider{name: "td"}
	_ = reg.Register("td", func() Provider { return mock })

	cfg := schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "td", Weight: 100},
		},
	}
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())
	orch.AddHosts("td", []Host{{ID: "h1", Provider: "td"}})

	err := orch.Teardown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if mock.releaseCalls != 1 {
		t.Errorf("release calls = %d, want 1", mock.releaseCalls)
	}
	if mock.tearCalls != 1 {
		t.Errorf("teardown calls = %d, want 1", mock.tearCalls)
	}
}

func TestOrchestratorCheckTTLAcquireError(t *testing.T) {
	reg := NewRegistry()
	mock := &mockProvider{
		name:       "acqfail",
		acquireErr: fmt.Errorf("out of capacity"),
	}
	_ = reg.Register("acqfail", func() Provider { return mock })

	cfg := schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "acqfail", Weight: 100, TTL: 1},
		},
	}
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())

	expired := Host{
		ID:       "old",
		Provider: "acqfail",
		BornAt:   time.Now().Add(-2 * time.Minute),
	}
	orch.AddHosts("acqfail", []Host{expired})

	err := orch.CheckTTL(context.Background())
	if err == nil {
		t.Fatal("expected error from failed acquire")
	}
}

func TestOrchestratorTeardownNoHosts(t *testing.T) {
	reg := NewRegistry()
	mock := &mockProvider{name: "empty"}
	_ = reg.Register("empty", func() Provider { return mock })

	cfg := schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "empty", Weight: 100},
		},
	}
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())

	err := orch.Teardown(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mock.releaseCalls != 0 {
		t.Errorf("release calls = %d, want 0", mock.releaseCalls)
	}
	if mock.tearCalls != 1 {
		t.Errorf("teardown calls = %d, want 1", mock.tearCalls)
	}
}

func TestOrchestratorTeardownTeardownError(t *testing.T) {
	reg := NewRegistry()
	mock := &mockProvider{
		name:    "terr",
		tearErr: fmt.Errorf("teardown failed"),
	}
	_ = reg.Register("terr", func() Provider { return mock })

	cfg := schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "terr", Weight: 100},
		},
	}
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())

	err := orch.Teardown(context.Background())
	if err == nil {
		t.Fatal("expected teardown error")
	}
}

func TestOrchestratorTeardownPartialError(t *testing.T) {
	reg := NewRegistry()
	mock := &mockProvider{
		name:       "err",
		releaseErr: fmt.Errorf("release failed"),
	}
	_ = reg.Register("err", func() Provider { return mock })

	cfg := schema.FleetConfig{
		Providers: []schema.FleetProvider{
			{Provider: "err", Weight: 100},
		},
	}
	orch := NewOrchestrator(reg, cfg)
	_ = orch.Setup(context.Background())
	orch.AddHosts("err", []Host{{ID: "h1", Provider: "err"}})

	err := orch.Teardown(context.Background())
	if err == nil {
		t.Fatal("expected error from release failure")
	}
	// Teardown should still be called even if release fails.
	if mock.tearCalls != 1 {
		t.Errorf("teardown calls = %d, want 1", mock.tearCalls)
	}
}
