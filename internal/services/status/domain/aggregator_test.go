package domain

import (
	"testing"
	"time"
)

func TestAggregator_ApplyReport_basic(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agg := NewAggregator(30*time.Second, func() time.Time { return now })

	caps := []CapabilityReport{
		{Name: "game.campaign.service", Status: StatusOperational},
		{Name: "game.social.integration", Status: StatusDegraded, Detail: "social unavailable"},
	}
	agg.ApplyReport("game", caps, now)

	snapshots := agg.Snapshot()
	if len(snapshots) != 1 {
		t.Fatalf("got %d services, want 1", len(snapshots))
	}
	ss := snapshots[0]
	if ss.Service != "game" {
		t.Fatalf("service = %q, want game", ss.Service)
	}
	if ss.AggregateStatus != StatusDegraded {
		t.Fatalf("aggregate = %v, want DEGRADED", ss.AggregateStatus)
	}
	if len(ss.Capabilities) != 2 {
		t.Fatalf("got %d capabilities, want 2", len(ss.Capabilities))
	}
}

func TestAggregator_ApplyReport_replaces_previous(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agg := NewAggregator(30*time.Second, func() time.Time { return now })

	agg.ApplyReport("game", []CapabilityReport{
		{Name: "cap.a", Status: StatusOperational},
		{Name: "cap.b", Status: StatusDegraded},
	}, now)

	// Second report replaces all capabilities.
	agg.ApplyReport("game", []CapabilityReport{
		{Name: "cap.c", Status: StatusOperational},
	}, now)

	snapshots := agg.Snapshot()
	ss := snapshots[0]
	if len(ss.Capabilities) != 1 {
		t.Fatalf("got %d capabilities, want 1 after replacement", len(ss.Capabilities))
	}
	if ss.Capabilities[0].Name != "cap.c" {
		t.Fatalf("capability = %q, want cap.c", ss.Capabilities[0].Name)
	}
}

func TestAggregator_staleness(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	current := start
	agg := NewAggregator(30*time.Second, func() time.Time { return current })

	agg.ApplyReport("game", []CapabilityReport{
		{Name: "game.service", Status: StatusOperational},
	}, start)

	// Before staleness threshold.
	current = start.Add(29 * time.Second)
	snap := agg.Snapshot()
	for _, cs := range snap[0].Capabilities {
		if cs.EffectiveStatus != StatusOperational {
			t.Fatalf("before threshold: effective = %v, want OPERATIONAL", cs.EffectiveStatus)
		}
	}

	// After staleness threshold.
	current = start.Add(31 * time.Second)
	snap = agg.Snapshot()
	for _, cs := range snap[0].Capabilities {
		if cs.EffectiveStatus != StatusUnavailable {
			t.Fatalf("after threshold: effective = %v, want UNAVAILABLE", cs.EffectiveStatus)
		}
	}
}

func TestAggregator_override_takes_precedence(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agg := NewAggregator(30*time.Second, func() time.Time { return now })

	agg.ApplyReport("game", []CapabilityReport{
		{Name: "game.service", Status: StatusOperational},
	}, now)

	agg.SetOverride(Override{
		Service:    "game",
		Capability: "game.service",
		Status:     StatusMaintenance,
		Reason:     OverrideReasonMaintenance,
		Detail:     "scheduled maintenance",
		SetAt:      now,
	})

	snap := agg.Snapshot()
	cs := snap[0].Capabilities[0]
	if cs.EffectiveStatus != StatusMaintenance {
		t.Fatalf("effective = %v, want MAINTENANCE", cs.EffectiveStatus)
	}
	if !cs.HasOverride {
		t.Fatal("HasOverride should be true")
	}
	if snap[0].HasOverrides != true {
		t.Fatal("service HasOverrides should be true")
	}
}

func TestAggregator_override_persists_across_reports(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agg := NewAggregator(30*time.Second, func() time.Time { return now })

	agg.SetOverride(Override{
		Service:    "game",
		Capability: "game.service",
		Status:     StatusMaintenance,
		SetAt:      now,
	})

	// New report should preserve override.
	agg.ApplyReport("game", []CapabilityReport{
		{Name: "game.service", Status: StatusOperational},
	}, now)

	snap := agg.Snapshot()
	cs := snap[0].Capabilities[0]
	if cs.EffectiveStatus != StatusMaintenance {
		t.Fatalf("effective = %v, want MAINTENANCE (override should persist)", cs.EffectiveStatus)
	}
}

func TestAggregator_ClearOverride(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agg := NewAggregator(30*time.Second, func() time.Time { return now })

	agg.ApplyReport("game", []CapabilityReport{
		{Name: "game.service", Status: StatusOperational},
	}, now)

	agg.SetOverride(Override{
		Service:    "game",
		Capability: "game.service",
		Status:     StatusMaintenance,
		SetAt:      now,
	})

	agg.ClearOverride("game", "game.service")

	snap := agg.Snapshot()
	cs := snap[0].Capabilities[0]
	if cs.EffectiveStatus != StatusOperational {
		t.Fatalf("effective = %v, want OPERATIONAL after clearing override", cs.EffectiveStatus)
	}
	if cs.HasOverride {
		t.Fatal("HasOverride should be false after clear")
	}
}

func TestAggregator_aggregate_worst_status(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agg := NewAggregator(30*time.Second, func() time.Time { return now })

	agg.ApplyReport("game", []CapabilityReport{
		{Name: "cap.a", Status: StatusOperational},
		{Name: "cap.b", Status: StatusDegraded},
		{Name: "cap.c", Status: StatusUnavailable},
	}, now)

	snap := agg.Snapshot()
	if snap[0].AggregateStatus != StatusUnavailable {
		t.Fatalf("aggregate = %v, want UNAVAILABLE (worst of all caps)", snap[0].AggregateStatus)
	}
}

func TestAggregator_empty_service(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agg := NewAggregator(30*time.Second, func() time.Time { return now })

	agg.ApplyReport("empty", nil, now)

	snap := agg.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("got %d services, want 1", len(snap))
	}
	if snap[0].AggregateStatus != StatusUnspecified {
		t.Fatalf("aggregate = %v, want UNSPECIFIED for empty capabilities", snap[0].AggregateStatus)
	}
}

func TestAggregator_multiple_services(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agg := NewAggregator(30*time.Second, func() time.Time { return now })

	agg.ApplyReport("game", []CapabilityReport{
		{Name: "game.service", Status: StatusOperational},
	}, now)
	agg.ApplyReport("web", []CapabilityReport{
		{Name: "web.render", Status: StatusDegraded},
	}, now)

	snap := agg.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("got %d services, want 2", len(snap))
	}
}

func TestAggregator_ClearOverride_nonexistent(t *testing.T) {
	agg := NewAggregator(30*time.Second, nil)
	// Should not panic.
	agg.ClearOverride("nonexistent", "nonexistent.cap")
}
