package goncho

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/providerpolicy"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

type ProviderKind string

const (
	ProviderKindExtraction    ProviderKind = "extraction"
	ProviderKindEmbedding     ProviderKind = "embedding"
	ProviderKindReranking     ProviderKind = "reranking"
	ProviderKindSummarization ProviderKind = "summarization"
)

type ProviderStatus string

const (
	ProviderStatusHealthy  ProviderStatus = "healthy"
	ProviderStatusDegraded ProviderStatus = "degraded"
	ProviderStatusDisabled ProviderStatus = "disabled"
)

type ProviderCircuitState string

const (
	ProviderCircuitClosed   ProviderCircuitState = "closed"
	ProviderCircuitOpen     ProviderCircuitState = "open"
	ProviderCircuitHalfOpen ProviderCircuitState = "half_open"
)

var ErrProviderCircuitOpen = errors.New("goncho: provider circuit open")

const (
	defaultProviderFailureThreshold = providerpolicy.DefaultFailureThreshold
	defaultProviderCooldown         = providerpolicy.DefaultCooldown
	defaultProviderTimeout          = providerpolicy.DefaultTimeout
)

type ProviderResilienceConfig struct {
	FailureThreshold int
	Cooldown         time.Duration
	Timeout          time.Duration
	MaxPayloadBytes  int
}

type ProviderCircuitBreakerConfig struct {
	Name             string
	Kind             ProviderKind
	FailureThreshold int
	Cooldown         time.Duration
	Timeout          time.Duration
	MaxPayloadBytes  int
	Now              func() time.Time
}

type ProviderHealth struct {
	Name            string               `json:"name"`
	Kind            ProviderKind         `json:"kind"`
	Status          ProviderStatus       `json:"status"`
	CircuitState    ProviderCircuitState `json:"circuit_state"`
	Optional        bool                 `json:"optional"`
	LastError       string               `json:"last_error,omitempty"`
	FailureCount    int                  `json:"failure_count,omitempty"`
	RetryAfter      time.Time            `json:"retry_after,omitempty"`
	TimeoutMillis   int64                `json:"timeout_ms,omitempty"`
	MaxPayloadBytes int                  `json:"max_payload_bytes,omitempty"`
}

type ProviderHealthDiagnostics []ProviderHealth

func (d ProviderHealthDiagnostics) ByName(name string) ProviderHealth {
	name = strings.TrimSpace(name)
	for _, health := range d {
		if health.Name == name {
			return health
		}
	}
	return ProviderHealth{}
}

type ProviderCircuitBreaker struct {
	mu               sync.Mutex
	name             string
	kind             ProviderKind
	failureThreshold int
	cooldown         time.Duration
	timeout          time.Duration
	maxPayloadBytes  int
	now              func() time.Time
	state            ProviderCircuitState
	failureCount     int
	lastError        string
	openedAt         time.Time
}

func NewProviderCircuitBreaker(cfg ProviderCircuitBreakerConfig) *ProviderCircuitBreaker {
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = string(cfg.Kind)
	}
	kind := cfg.Kind
	if kind == "" {
		kind = ProviderKind(name)
	}
	threshold := defaultProviderThreshold(cfg.FailureThreshold)
	cooldown := defaultProviderCooldownDuration(cfg.Cooldown)
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &ProviderCircuitBreaker{name: name, kind: kind, failureThreshold: threshold, cooldown: cooldown, timeout: cfg.Timeout, maxPayloadBytes: cfg.MaxPayloadBytes, now: now, state: ProviderCircuitClosed}
}

func (b *ProviderCircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if b == nil {
		return fn(ctx)
	}
	if err := b.beforeCall(); err != nil {
		return err
	}
	callCtx := ctx
	cancel := func() {}
	if b.timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, b.timeout)
	}
	defer cancel()
	err := fn(callCtx)
	if err != nil {
		b.recordFailure(err)
		return err
	}
	b.recordSuccess()
	return nil
}

func (b *ProviderCircuitBreaker) Health() ProviderHealth {
	if b == nil {
		return ProviderHealth{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	status := ProviderStatusHealthy
	if b.state == ProviderCircuitOpen || b.failureCount > 0 {
		status = ProviderStatusDegraded
	}
	return ProviderHealth{Name: b.name, Kind: b.kind, Status: status, CircuitState: b.state, Optional: true, LastError: b.lastError, FailureCount: b.failureCount, RetryAfter: b.retryAfterLocked(), TimeoutMillis: b.timeout.Milliseconds(), MaxPayloadBytes: b.maxPayloadBytes}
}

func (b *ProviderCircuitBreaker) beforeCall() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now().UTC()
	if b.state == ProviderCircuitOpen {
		if now.Before(b.openedAt.Add(b.cooldown)) {
			return ErrProviderCircuitOpen
		}
		b.state = ProviderCircuitHalfOpen
	}
	return nil
}

func (b *ProviderCircuitBreaker) recordFailure(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failureCount++
	b.lastError = err.Error()
	if b.failureCount >= b.failureThreshold || b.state == ProviderCircuitHalfOpen {
		b.state = ProviderCircuitOpen
		b.openedAt = b.now().UTC()
	}
}

func (b *ProviderCircuitBreaker) recordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failureCount = 0
	b.lastError = ""
	b.state = ProviderCircuitClosed
	b.openedAt = time.Time{}
}

func (b *ProviderCircuitBreaker) retryAfterLocked() time.Time {
	if b.state != ProviderCircuitOpen || b.openedAt.IsZero() {
		return time.Time{}
	}
	return b.openedAt.Add(b.cooldown).UTC()
}

type ProviderHealthRegistry struct {
	mu       sync.Mutex
	config   ProviderResilienceConfig
	breakers map[string]*ProviderCircuitBreaker
}

func NewProviderHealthRegistry(cfg ProviderResilienceConfig, vectorStore VectorStore) *ProviderHealthRegistry {
	cfg = normalizeProviderResilienceConfig(cfg)
	registry := &ProviderHealthRegistry{config: cfg, breakers: map[string]*ProviderCircuitBreaker{}}
	if vectorStore != nil {
		registry.breakers[string(ProviderKindEmbedding)] = NewProviderCircuitBreaker(ProviderCircuitBreakerConfig{Name: string(ProviderKindEmbedding), Kind: ProviderKindEmbedding, FailureThreshold: cfg.FailureThreshold, Cooldown: cfg.Cooldown, Timeout: cfg.Timeout, MaxPayloadBytes: cfg.MaxPayloadBytes})
	}
	return registry
}

func normalizeProviderResilienceConfig(cfg ProviderResilienceConfig) ProviderResilienceConfig {
	normalized := providerpolicy.Normalize(providerpolicy.Config{
		FailureThreshold: cfg.FailureThreshold,
		Cooldown:         cfg.Cooldown,
		Timeout:          cfg.Timeout,
		MaxPayloadBytes:  cfg.MaxPayloadBytes,
	})
	return ProviderResilienceConfig{
		FailureThreshold: normalized.FailureThreshold,
		Cooldown:         normalized.Cooldown,
		Timeout:          normalized.Timeout,
		MaxPayloadBytes:  normalized.MaxPayloadBytes,
	}
}

func defaultProviderThreshold(threshold int) int {
	return providerpolicy.FailureThreshold(threshold)
}

func defaultProviderCooldownDuration(cooldown time.Duration) time.Duration {
	return providerpolicy.Cooldown(cooldown)
}

func providerResilienceConfigFromServiceConfig(cfg Config) ProviderResilienceConfig {
	return normalizeProviderResilienceConfig(ProviderResilienceConfig{FailureThreshold: cfg.ProviderFailureThreshold, Cooldown: cfg.ProviderCooldown, Timeout: cfg.ProviderTimeout, MaxPayloadBytes: cfg.ProviderMaxPayloadBytes})
}

func (r *ProviderHealthRegistry) Execute(ctx context.Context, name string, fn func(context.Context) error) error {
	if r == nil {
		return fn(ctx)
	}
	breaker := r.breaker(name)
	if breaker == nil {
		return fn(ctx)
	}
	return breaker.Execute(ctx, fn)
}

func (r *ProviderHealthRegistry) MaxPayloadBytes(name string) int {
	if r == nil {
		return 0
	}
	breaker := r.breaker(name)
	if breaker == nil {
		return r.config.MaxPayloadBytes
	}
	breaker.mu.Lock()
	defer breaker.mu.Unlock()
	return breaker.maxPayloadBytes
}

func (r *ProviderHealthRegistry) Diagnostics() ProviderHealthDiagnostics {
	if r == nil {
		return defaultProviderHealthDiagnostics(nil)
	}
	r.mu.Lock()
	breakers := make(map[string]*ProviderCircuitBreaker, len(r.breakers))
	for name, breaker := range r.breakers {
		breakers[name] = breaker
	}
	r.mu.Unlock()
	return defaultProviderHealthDiagnostics(breakers)
}

func (r *ProviderHealthRegistry) breaker(name string) *ProviderCircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.breakers[strings.TrimSpace(name)]
}

func defaultProviderHealthDiagnostics(breakers map[string]*ProviderCircuitBreaker) ProviderHealthDiagnostics {
	providers := []ProviderHealth{}
	for _, kind := range []ProviderKind{ProviderKindExtraction, ProviderKindEmbedding, ProviderKindReranking, ProviderKindSummarization} {
		name := string(kind)
		if breaker := breakers[name]; breaker != nil {
			providers = append(providers, breaker.Health())
			continue
		}
		providers = append(providers, ProviderHealth{Name: name, Kind: kind, Status: ProviderStatusDisabled, CircuitState: ProviderCircuitClosed, Optional: true})
	}
	sort.SliceStable(providers, func(i, j int) bool { return providers[i].Name < providers[j].Name })
	return ProviderHealthDiagnostics(providers)
}

func (s *Service) ProviderHealthDiagnostics() ProviderHealthDiagnostics {
	if s == nil || s.providerRegistry == nil {
		return defaultProviderHealthDiagnostics(nil)
	}
	return s.providerRegistry.Diagnostics()
}

func providerWarnings(health ProviderHealthDiagnostics) []string {
	warnings := []string{}
	for _, provider := range health {
		switch provider.Status {
		case ProviderStatusDisabled:
			warnings = append(warnings, fmt.Sprintf("provider %s disabled", provider.Name))
		case ProviderStatusDegraded:
			message := fmt.Sprintf("provider %s degraded", provider.Name)
			if provider.LastError != "" {
				message += ": " + provider.LastError
			}
			warnings = append(warnings, message)
		}
	}
	return warnings
}

type recallWarningBuffer struct {
	mu       sync.Mutex
	warnings []RecallWarning
}

func (b *recallWarningBuffer) append(warnings ...RecallWarning) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.warnings = appendRecallWarnings(b.warnings, warnings...)
}

func (b *recallWarningBuffer) list() []RecallWarning {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return sliceutil.Clone(b.warnings)
}
