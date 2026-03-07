// Package domain implements in-memory capability status aggregation with
// staleness detection and operator override merging.
package domain

import (
	"sync"
	"time"
)

// CapabilityStatus mirrors the proto enum for domain use without proto dependency.
type CapabilityStatus int

const (
	StatusUnspecified CapabilityStatus = 0
	StatusOperational CapabilityStatus = 1
	StatusDegraded    CapabilityStatus = 2
	StatusUnavailable CapabilityStatus = 3
	StatusMaintenance CapabilityStatus = 4
)

// OverrideReason classifies why an operator override was applied.
type OverrideReason int

const (
	OverrideReasonUnspecified OverrideReason = 0
	OverrideReasonMaintenance OverrideReason = 1
	OverrideReasonKnownIssue  OverrideReason = 2
	OverrideReasonDegraded    OverrideReason = 3
	OverrideReasonUnavailable OverrideReason = 4
)

// CapabilityReport is a single capability observation pushed by a service.
type CapabilityReport struct {
	Name       string
	Status     CapabilityStatus
	Detail     string
	ObservedAt time.Time
}

// Override is an operator-set status override for a specific capability.
type Override struct {
	Service    string
	Capability string
	Status     CapabilityStatus
	Reason     OverrideReason
	Detail     string
	SetAt      time.Time
}

// CapabilitySnapshot is the resolved view combining report and override.
type CapabilitySnapshot struct {
	Name            string
	ReportedStatus  CapabilityStatus
	ReportedDetail  string
	EffectiveStatus CapabilityStatus
	HasOverride     bool
	Override        *Override
	ObservedAt      time.Time
}

// ServiceSnapshot is the aggregate state of one service.
type ServiceSnapshot struct {
	Service         string
	AggregateStatus CapabilityStatus
	Capabilities    []CapabilitySnapshot
	LastReportAt    time.Time
	HasOverrides    bool
}

// serviceState holds mutable per-service state.
type serviceState struct {
	capabilities map[string]*capabilityState
	lastReportAt time.Time
}

type capabilityState struct {
	report   CapabilityReport
	override *Override
}

// DefaultStalenessThreshold is the duration after which a service without
// reports is considered stale.
const DefaultStalenessThreshold = 30 * time.Second

// Aggregator maintains in-memory capability health state for all services.
type Aggregator struct {
	mu                 sync.RWMutex
	services           map[string]*serviceState
	stalenessThreshold time.Duration
	now                func() time.Time
}

// NewAggregator creates an aggregator with the given staleness threshold.
// If threshold is zero, DefaultStalenessThreshold is used.
func NewAggregator(threshold time.Duration, now func() time.Time) *Aggregator {
	if threshold <= 0 {
		threshold = DefaultStalenessThreshold
	}
	if now == nil {
		now = time.Now
	}
	return &Aggregator{
		services:           make(map[string]*serviceState),
		stalenessThreshold: threshold,
		now:                now,
	}
}

// ApplyReport updates the aggregator with a full capability report from a service.
// It replaces all previously known capabilities for that service.
func (a *Aggregator) ApplyReport(service string, capabilities []CapabilityReport, reportedAt time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	state, ok := a.services[service]
	if !ok {
		state = &serviceState{
			capabilities: make(map[string]*capabilityState),
		}
		a.services[service] = state
	}

	// Preserve overrides, rebuild capabilities from the new report.
	overrides := make(map[string]*Override)
	for name, cs := range state.capabilities {
		if cs.override != nil {
			overrides[name] = cs.override
		}
	}

	state.capabilities = make(map[string]*capabilityState, len(capabilities))
	for _, cap := range capabilities {
		cs := &capabilityState{report: cap}
		if ov, ok := overrides[name(cap.Name)]; ok {
			cs.override = ov
		}
		state.capabilities[name(cap.Name)] = cs
	}
	state.lastReportAt = reportedAt
}

// SetOverride applies an operator override to a specific capability.
// If the service or capability doesn't exist yet, it creates placeholder state.
func (a *Aggregator) SetOverride(ov Override) {
	a.mu.Lock()
	defer a.mu.Unlock()

	state, ok := a.services[ov.Service]
	if !ok {
		state = &serviceState{
			capabilities: make(map[string]*capabilityState),
		}
		a.services[ov.Service] = state
	}
	cs, ok := state.capabilities[name(ov.Capability)]
	if !ok {
		cs = &capabilityState{}
		state.capabilities[name(ov.Capability)] = cs
	}
	cs.override = &ov
}

// ClearOverride removes an operator override from a capability.
func (a *Aggregator) ClearOverride(service, capability string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	state, ok := a.services[service]
	if !ok {
		return
	}
	cs, ok := state.capabilities[name(capability)]
	if !ok {
		return
	}
	cs.override = nil
}

// Snapshot returns the current resolved view of all services.
func (a *Aggregator) Snapshot() []ServiceSnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	now := a.now()
	result := make([]ServiceSnapshot, 0, len(a.services))
	for svc, state := range a.services {
		ss := a.snapshotService(svc, state, now)
		result = append(result, ss)
	}
	return result
}

func (a *Aggregator) snapshotService(svc string, state *serviceState, now time.Time) ServiceSnapshot {
	stale := !state.lastReportAt.IsZero() && now.Sub(state.lastReportAt) > a.stalenessThreshold

	caps := make([]CapabilitySnapshot, 0, len(state.capabilities))
	worstStatus := StatusOperational
	hasOverrides := false

	for _, cs := range state.capabilities {
		snap := resolveCapability(cs, stale)
		caps = append(caps, snap)
		if snap.HasOverride {
			hasOverrides = true
		}
		if snap.EffectiveStatus > worstStatus {
			worstStatus = snap.EffectiveStatus
		}
	}

	// If there are no capabilities, mark as unspecified.
	if len(caps) == 0 {
		worstStatus = StatusUnspecified
	}

	return ServiceSnapshot{
		Service:         svc,
		AggregateStatus: worstStatus,
		Capabilities:    caps,
		LastReportAt:    state.lastReportAt,
		HasOverrides:    hasOverrides,
	}
}

// resolveCapability merges reported status with override and staleness.
func resolveCapability(cs *capabilityState, stale bool) CapabilitySnapshot {
	reported := cs.report.Status
	if stale && reported != StatusUnspecified {
		// Stale services degrade their reported status if it was operational.
		if reported == StatusOperational {
			reported = StatusUnavailable
		}
	}

	effective := reported
	snap := CapabilitySnapshot{
		Name:            cs.report.Name,
		ReportedStatus:  cs.report.Status,
		ReportedDetail:  cs.report.Detail,
		EffectiveStatus: effective,
		ObservedAt:      cs.report.ObservedAt,
	}

	if cs.override != nil {
		snap.HasOverride = true
		snap.Override = cs.override
		snap.EffectiveStatus = cs.override.Status
	}

	return snap
}

func name(s string) string {
	return s
}
