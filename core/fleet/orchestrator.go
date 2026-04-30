package fleet

import (
	"context"
	"fmt"
	mrand "math/rand/v2"
	"sync"
	"time"

	"github.com/scrutineer/scrutineer/core/schema"
)

// Orchestrator distributes test groups across fleet providers based on
// weights and manages the fleet lifecycle including TTL-based recycling.
type Orchestrator struct {
	registry  *Registry
	config    schema.FleetConfig
	providers map[string]Provider
	hosts     map[string][]Host
	ttls      map[string]int // TTL in minutes per provider
	mu        sync.Mutex
	rng       *mrand.Rand
}

// NewOrchestrator creates an Orchestrator.
func NewOrchestrator(reg *Registry, cfg schema.FleetConfig) *Orchestrator {
	return &Orchestrator{
		registry:  reg,
		config:    cfg,
		providers: make(map[string]Provider),
		hosts:     make(map[string][]Host),
		ttls:      make(map[string]int),
		rng:       mrand.New(mrand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
}

// Setup initializes all providers and acquires initial hosts.
func (o *Orchestrator) Setup(ctx context.Context) error {
	for _, fp := range o.config.Providers {
		prov, err := o.registry.Get(fp.Provider)
		if err != nil {
			return fmt.Errorf("fleet setup: %w", err)
		}

		if err := prov.Setup(ctx, fp.Config); err != nil {
			return fmt.Errorf("fleet setup %s: %w", fp.Provider, err)
		}

		o.providers[fp.Provider] = prov
		o.ttls[fp.Provider] = fp.TTL
	}
	return nil
}

// SelectProvider picks a provider name using weighted random selection
// based on provider weights.
func (o *Orchestrator) SelectProvider() string {
	o.mu.Lock()
	defer o.mu.Unlock()

	weights := make([]int, len(o.config.Providers))
	for i, p := range o.config.Providers {
		weights[i] = p.Weight
	}
	idx := SelectByWeight(weights, o.rng)
	return o.config.Providers[idx].Provider
}

// SelectHost picks a host from the named provider's pool using
// round-robin selection.
func (o *Orchestrator) SelectHost(provider string) (Host, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	hosts := o.hosts[provider]
	if len(hosts) == 0 {
		return Host{}, fmt.Errorf("fleet: no hosts available for provider %q", provider)
	}
	// Simple round-robin: rotate the slice.
	host := hosts[0]
	o.hosts[provider] = append(hosts[1:], host)
	return host, nil
}

// AddHosts adds hosts to a provider's pool.
func (o *Orchestrator) AddHosts(provider string, hosts []Host) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.hosts[provider] = append(o.hosts[provider], hosts...)
}

// Execute runs a command on a host via its provider.
func (o *Orchestrator) Execute(ctx context.Context, host Host, cmd string) (*ExecResult, error) {
	prov, ok := o.providers[host.Provider]
	if !ok {
		return nil, fmt.Errorf("fleet: provider %q not found for host %s", host.Provider, host.ID)
	}
	return prov.Execute(ctx, host, cmd)
}

// CheckTTL identifies hosts that have exceeded their TTL, releases them,
// acquires replacements, and adds them to the pool.
func (o *Orchestrator) CheckTTL(ctx context.Context) error {
	o.mu.Lock()
	providerNames := make([]string, 0, len(o.providers))
	for name := range o.providers {
		providerNames = append(providerNames, name)
	}
	o.mu.Unlock()

	now := time.Now()

	for _, name := range providerNames {
		ttl := o.ttls[name]
		if ttl <= 0 {
			continue
		}
		ttlDuration := time.Duration(ttl) * time.Minute

		o.mu.Lock()
		hosts := o.hosts[name]
		var expired []Host
		var remaining []Host
		for _, h := range hosts {
			if now.Sub(h.BornAt) >= ttlDuration {
				expired = append(expired, h)
			} else {
				remaining = append(remaining, h)
			}
		}
		o.hosts[name] = remaining
		o.mu.Unlock()

		if len(expired) == 0 {
			continue
		}

		prov := o.providers[name]

		// Acquire replacements before releasing expired hosts.
		replacements, err := prov.Acquire(ctx, len(expired))
		if err != nil {
			return fmt.Errorf("fleet: failed to acquire replacements for %s: %w", name, err)
		}

		o.mu.Lock()
		o.hosts[name] = append(o.hosts[name], replacements...)
		o.mu.Unlock()

		// Release expired hosts.
		if err := prov.Release(ctx, expired); err != nil {
			return fmt.Errorf("fleet: failed to release expired hosts for %s: %w", name, err)
		}
	}

	return nil
}

// Teardown releases all hosts and tears down all providers.
func (o *Orchestrator) Teardown(ctx context.Context) error {
	var firstErr error

	for name, prov := range o.providers {
		o.mu.Lock()
		hosts := o.hosts[name]
		o.hosts[name] = nil
		o.mu.Unlock()

		if len(hosts) > 0 {
			if err := prov.Release(ctx, hosts); err != nil && firstErr == nil {
				firstErr = err
			}
		}

		if err := prov.Teardown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
