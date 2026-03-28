package status

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeStatusClient struct {
	statusv1.StatusServiceClient
	resp *statusv1.GetSystemStatusResponse
	err  error
}

func (c fakeStatusClient) GetSystemStatus(context.Context, *statusv1.GetSystemStatusRequest, ...grpc.CallOption) (*statusv1.GetSystemStatusResponse, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.resp, nil
}

func TestStatusHandlersRenderPageAndTableStates(t *testing.T) {
	t.Parallel()

	svcIface := NewHandlers(modulehandler.NewBase(), fakeStatusClient{})
	svc, ok := svcIface.(handlers)
	if !ok {
		t.Fatalf("NewHandlers() type = %T, want handlers", svcIface)
	}

	t.Run("status page renders lazy rows loader", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app/status", nil)
		rec := httptest.NewRecorder()

		svc.HandleStatusPage(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "/app/status?fragment=rows") {
			t.Fatalf("body = %q, want lazy rows path", body)
		}
	})

	t.Run("status table degrades gracefully when backend is unavailable", func(t *testing.T) {
		svcIface := NewHandlers(modulehandler.NewBase(), fakeStatusClient{
			err: status.Error(codes.Unavailable, "status backend down"),
		})
		svc := svcIface.(handlers)

		req := httptest.NewRequest(http.MethodGet, "/app/status?fragment=rows", nil)
		rec := httptest.NewRecorder()

		svc.HandleStatusTable(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "Status service is not connected.") {
			t.Fatalf("body = %q, want fallback message", body)
		}
	})

	t.Run("status table renders grouped service data", func(t *testing.T) {
		svcIface := NewHandlers(modulehandler.NewBase(), fakeStatusClient{
			resp: &statusv1.GetSystemStatusResponse{
				Services: []*statusv1.ServiceStatus{
					{
						Service:         "web",
						AggregateStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_DEGRADED,
						HasOverrides:    true,
						Capabilities: []*statusv1.CapabilitySnapshot{
							{
								Name:            "http",
								ReportedStatus:  statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL,
								EffectiveStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_MAINTENANCE,
								HasOverride:     true,
								Override: &statusv1.CapabilityOverride{
									Reason: statusv1.OverrideReason_OVERRIDE_REASON_MAINTENANCE,
									Detail: "deploy in progress",
								},
							},
						},
					},
				},
			},
		})
		svc := svcIface.(handlers)

		req := httptest.NewRequest(http.MethodGet, "/app/status?fragment=rows", nil)
		rec := httptest.NewRecorder()

		svc.HandleStatusTable(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		for _, want := range []string{"web", "http", "Degraded", "Operational", "Maintenance: deploy in progress"} {
			if !strings.Contains(body, want) {
				t.Fatalf("body = %q, want substring %q", body, want)
			}
		}
	})
}

func TestStatusViewMappersAndFragmentHelpers(t *testing.T) {
	t.Parallel()

	loc := i18nhttp.Printer(i18nhttp.Default())
	groups := buildServiceGroups([]*statusv1.ServiceStatus{
		nil,
		{
			Service:         "worker",
			AggregateStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_UNAVAILABLE,
			Capabilities: []*statusv1.CapabilitySnapshot{
				{
					Name:            "queue",
					ReportedStatus:  statusv1.CapabilityStatus_CAPABILITY_STATUS_UNAVAILABLE,
					EffectiveStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_UNAVAILABLE,
				},
			},
		},
		{
			Service:         "api",
			AggregateStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL,
			HasOverrides:    true,
			Capabilities: []*statusv1.CapabilitySnapshot{
				nil,
				{
					Name:            "zeta",
					ReportedStatus:  statusv1.CapabilityStatus_CAPABILITY_STATUS_DEGRADED,
					EffectiveStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_MAINTENANCE,
					HasOverride:     true,
					Override: &statusv1.CapabilityOverride{
						Reason: statusv1.OverrideReason_OVERRIDE_REASON_KNOWN_ISSUE,
						Detail: "rolling restart",
					},
				},
				{
					Name:            "alpha",
					ReportedStatus:  statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL,
					EffectiveStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL,
				},
			},
		},
	}, loc)

	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	if groups[0].Service != "api" || groups[1].Service != "worker" {
		t.Fatalf("group order = %+v", groups)
	}
	if len(groups[0].Capabilities) != 2 || groups[0].Capabilities[0].Capability != "alpha" || groups[0].Capabilities[1].Capability != "zeta" {
		t.Fatalf("capability order = %+v", groups[0].Capabilities)
	}
	if groups[0].Capabilities[1].OverrideDetail != "Known Issue: rolling restart" {
		t.Fatalf("override detail = %q, want %q", groups[0].Capabilities[1].OverrideDetail, "Known Issue: rolling restart")
	}

	if got := formatCapabilityStatus(statusv1.CapabilityStatus_CAPABILITY_STATUS_DEGRADED, loc); got != "Degraded" {
		t.Fatalf("formatCapabilityStatus(degraded) = %q, want %q", got, "Degraded")
	}
	if got := formatCapabilityStatus(statusv1.CapabilityStatus(99), loc); got != "Unspecified" {
		t.Fatalf("formatCapabilityStatus(default) = %q, want %q", got, "Unspecified")
	}
	if got := statusVariant(statusv1.CapabilityStatus_CAPABILITY_STATUS_MAINTENANCE); got != "info" {
		t.Fatalf("statusVariant(maintenance) = %q, want %q", got, "info")
	}
	if got := statusVariant(statusv1.CapabilityStatus(99)); got != "ghost" {
		t.Fatalf("statusVariant(default) = %q, want %q", got, "ghost")
	}
	if got := formatOverrideReason(statusv1.OverrideReason_OVERRIDE_REASON_UNAVAILABLE, loc); got != "Unavailable" {
		t.Fatalf("formatOverrideReason(unavailable) = %q, want %q", got, "Unavailable")
	}
	if got := formatOverrideReason(statusv1.OverrideReason(99), loc); got != "Unspecified" {
		t.Fatalf("formatOverrideReason(default) = %q, want %q", got, "Unspecified")
	}
	if got := formatOverrideDetail(&statusv1.CapabilityOverride{
		Reason: statusv1.OverrideReason_OVERRIDE_REASON_DEGRADED,
	}, loc); got != "Degraded" {
		t.Fatalf("formatOverrideDetail(no detail) = %q, want %q", got, "Degraded")
	}

	if buildServiceGroups(nil, loc) != nil {
		t.Fatal("expected nil groups for nil input")
	}

	if wantsRowsFragment(nil) {
		t.Fatal("expected nil request to return false")
	}
	req := httptest.NewRequest(http.MethodGet, "/app/status?fragment=%20cards%20", nil)
	if wantsRowsFragment(req) {
		t.Fatal("expected trimmed non-matching fragment to return false")
	}
	req = httptest.NewRequest(http.MethodGet, "/app/status?fragment=ROWS", nil)
	if !wantsRowsFragment(req) {
		t.Fatal("expected case-insensitive rows fragment to return true")
	}
}
