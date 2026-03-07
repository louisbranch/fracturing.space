package status

import (
	"context"
	"testing"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"github.com/louisbranch/fracturing.space/internal/services/status/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestService_ReportAndQuery(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	agg := domain.NewAggregator(30*time.Second, func() time.Time { return now })
	svc := NewService(agg, nil, func() time.Time { return now })

	// Report status.
	_, err := svc.ReportStatus(context.Background(), &statusv1.ReportStatusRequest{
		Report: &statusv1.ServiceStatusReport{
			Service: "game",
			Capabilities: []*statusv1.CapabilityReport{
				{
					Name:       "game.campaign.service",
					Status:     statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL,
					ObservedAt: timestamppb.New(now),
				},
				{
					Name:       "game.social.integration",
					Status:     statusv1.CapabilityStatus_CAPABILITY_STATUS_DEGRADED,
					Detail:     "social unreachable",
					ObservedAt: timestamppb.New(now),
				},
			},
			ReportedAt: timestamppb.New(now),
		},
	})
	if err != nil {
		t.Fatalf("ReportStatus: %v", err)
	}

	// Query status.
	resp, err := svc.GetSystemStatus(context.Background(), &statusv1.GetSystemStatusRequest{})
	if err != nil {
		t.Fatalf("GetSystemStatus: %v", err)
	}
	if len(resp.Services) != 1 {
		t.Fatalf("got %d services, want 1", len(resp.Services))
	}
	ss := resp.Services[0]
	if ss.Service != "game" {
		t.Fatalf("service = %q, want game", ss.Service)
	}
	if ss.AggregateStatus != statusv1.CapabilityStatus_CAPABILITY_STATUS_DEGRADED {
		t.Fatalf("aggregate = %v, want DEGRADED", ss.AggregateStatus)
	}
	if len(ss.Capabilities) != 2 {
		t.Fatalf("got %d capabilities, want 2", len(ss.Capabilities))
	}
}

func TestService_SetAndClearOverride(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	agg := domain.NewAggregator(30*time.Second, func() time.Time { return now })
	svc := NewService(agg, nil, func() time.Time { return now })

	// Report operational.
	_, _ = svc.ReportStatus(context.Background(), &statusv1.ReportStatusRequest{
		Report: &statusv1.ServiceStatusReport{
			Service: "game",
			Capabilities: []*statusv1.CapabilityReport{
				{Name: "game.service", Status: statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL},
			},
			ReportedAt: timestamppb.New(now),
		},
	})

	// Set override.
	_, err := svc.SetOverride(context.Background(), &statusv1.SetOverrideRequest{
		Service:    "game",
		Capability: "game.service",
		Status:     statusv1.CapabilityStatus_CAPABILITY_STATUS_MAINTENANCE,
		Reason:     statusv1.OverrideReason_OVERRIDE_REASON_MAINTENANCE,
		Detail:     "planned downtime",
	})
	if err != nil {
		t.Fatalf("SetOverride: %v", err)
	}

	resp, _ := svc.GetSystemStatus(context.Background(), &statusv1.GetSystemStatusRequest{})
	cs := resp.Services[0].Capabilities[0]
	if cs.EffectiveStatus != statusv1.CapabilityStatus_CAPABILITY_STATUS_MAINTENANCE {
		t.Fatalf("effective = %v, want MAINTENANCE", cs.EffectiveStatus)
	}
	if !cs.HasOverride {
		t.Fatal("HasOverride should be true")
	}

	// Clear override.
	_, err = svc.ClearOverride(context.Background(), &statusv1.ClearOverrideRequest{
		Service:    "game",
		Capability: "game.service",
	})
	if err != nil {
		t.Fatalf("ClearOverride: %v", err)
	}

	resp, _ = svc.GetSystemStatus(context.Background(), &statusv1.GetSystemStatusRequest{})
	cs = resp.Services[0].Capabilities[0]
	if cs.EffectiveStatus != statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL {
		t.Fatalf("effective = %v, want OPERATIONAL after clear", cs.EffectiveStatus)
	}
}

func TestService_ReportStatus_validation(t *testing.T) {
	agg := domain.NewAggregator(30*time.Second, nil)
	svc := NewService(agg, nil, nil)

	_, err := svc.ReportStatus(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for nil request, got %v", status.Code(err))
	}

	_, err = svc.ReportStatus(context.Background(), &statusv1.ReportStatusRequest{
		Report: &statusv1.ServiceStatusReport{Service: ""},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for empty service, got %v", status.Code(err))
	}
}

func TestService_SetOverride_validation(t *testing.T) {
	agg := domain.NewAggregator(30*time.Second, nil)
	svc := NewService(agg, nil, nil)

	_, err := svc.SetOverride(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for nil set override request, got %v", status.Code(err))
	}

	_, err = svc.ClearOverride(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for nil clear override request, got %v", status.Code(err))
	}

	_, err = svc.SetOverride(context.Background(), &statusv1.SetOverrideRequest{
		Service: "", Capability: "cap",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for empty set override service, got %v", status.Code(err))
	}

	_, err = svc.SetOverride(context.Background(), &statusv1.SetOverrideRequest{
		Service: "game", Capability: "",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for empty set override capability, got %v", status.Code(err))
	}

	_, err = svc.ClearOverride(context.Background(), &statusv1.ClearOverrideRequest{
		Service: "", Capability: "cap",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for empty clear override service, got %v", status.Code(err))
	}

	_, err = svc.ClearOverride(context.Background(), &statusv1.ClearOverrideRequest{
		Service: "game", Capability: "",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for empty clear override capability, got %v", status.Code(err))
	}
}
