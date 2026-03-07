package status

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/a-h/templ"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

// statusCallTimeout caps the gRPC call time for status queries.
const statusCallTimeout = 5 * time.Second

// service implements status module handlers.
type service struct {
	base         modulehandler.Base
	statusClient statusv1.StatusServiceClient
}

// NewService builds the status module service implementation.
func NewService(base modulehandler.Base, client statusv1.StatusServiceClient) Service {
	return service{base: base, statusClient: client}
}

// HandleStatusPage renders the status page fragment or full layout.
func (s service) HandleStatusPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.StatusPage(loc),
		templates.StatusFullPage(pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.status", templates.AppName()),
	)
}

// HandleStatusTable renders the status table via HTMX.
func (s service) HandleStatusTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := s.base.Localizer(w, r)
	if s.statusClient == nil {
		s.renderStatusTable(w, r, nil, loc.Sprintf("status.not_connected"), loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), statusCallTimeout)
	defer cancel()

	resp, err := s.statusClient.GetSystemStatus(ctx, &statusv1.GetSystemStatusRequest{})
	if err != nil {
		// Status service is advisory — degrade gracefully when unreachable.
		log.Printf("get system status: %v", err)
		s.renderStatusTable(w, r, nil, loc.Sprintf("status.not_connected"), loc)
		return
	}

	groups := buildServiceGroups(resp.GetServices(), loc)
	s.renderStatusTable(w, r, groups, "", loc)
}

// renderStatusTable renders the status table fragment.
func (s service) renderStatusTable(w http.ResponseWriter, r *http.Request, groups []templates.StatusServiceGroup, msg string, loc *message.Printer) {
	templ.Handler(templates.StatusTable(groups, msg, loc)).ServeHTTP(w, r)
}

// buildServiceGroups transforms proto service status into template view data.
func buildServiceGroups(services []*statusv1.ServiceStatus, loc *message.Printer) []templates.StatusServiceGroup {
	if len(services) == 0 {
		return nil
	}

	groups := make([]templates.StatusServiceGroup, 0, len(services))
	for _, svc := range services {
		if svc == nil {
			continue
		}
		group := templates.StatusServiceGroup{
			Service:         svc.GetService(),
			AggregateStatus: formatCapabilityStatus(svc.GetAggregateStatus(), loc),
			StatusVariant:   statusVariant(svc.GetAggregateStatus()),
			HasOverrides:    svc.GetHasOverrides(),
		}

		caps := svc.GetCapabilities()
		group.Capabilities = make([]templates.StatusCapabilityRow, 0, len(caps))
		for _, cap := range caps {
			if cap == nil {
				continue
			}
			row := templates.StatusCapabilityRow{
				Service:         svc.GetService(),
				Capability:      cap.GetName(),
				ReportedStatus:  formatCapabilityStatus(cap.GetReportedStatus(), loc),
				EffectiveStatus: formatCapabilityStatus(cap.GetEffectiveStatus(), loc),
				StatusVariant:   statusVariant(cap.GetEffectiveStatus()),
				HasOverride:     cap.GetHasOverride(),
			}
			if cap.GetHasOverride() && cap.GetOverride() != nil {
				row.OverrideDetail = formatOverrideDetail(cap.GetOverride(), loc)
			}
			group.Capabilities = append(group.Capabilities, row)
		}
		sort.Slice(group.Capabilities, func(i, j int) bool {
			return group.Capabilities[i].Capability < group.Capabilities[j].Capability
		})
		groups = append(groups, group)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Service < groups[j].Service
	})
	return groups
}

// formatCapabilityStatus returns a localized capability status label.
func formatCapabilityStatus(s statusv1.CapabilityStatus, loc *message.Printer) string {
	switch s {
	case statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL:
		return loc.Sprintf("label.status_operational")
	case statusv1.CapabilityStatus_CAPABILITY_STATUS_DEGRADED:
		return loc.Sprintf("label.status_degraded")
	case statusv1.CapabilityStatus_CAPABILITY_STATUS_UNAVAILABLE:
		return loc.Sprintf("label.status_unavailable")
	case statusv1.CapabilityStatus_CAPABILITY_STATUS_MAINTENANCE:
		return loc.Sprintf("label.status_maintenance")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

// statusVariant maps capability status to DaisyUI badge variants.
func statusVariant(s statusv1.CapabilityStatus) string {
	switch s {
	case statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL:
		return "success"
	case statusv1.CapabilityStatus_CAPABILITY_STATUS_DEGRADED:
		return "warning"
	case statusv1.CapabilityStatus_CAPABILITY_STATUS_UNAVAILABLE:
		return "error"
	case statusv1.CapabilityStatus_CAPABILITY_STATUS_MAINTENANCE:
		return "info"
	default:
		return "ghost"
	}
}

// formatOverrideDetail renders a human-readable override description.
func formatOverrideDetail(o *statusv1.CapabilityOverride, loc *message.Printer) string {
	reason := formatOverrideReason(o.GetReason(), loc)
	detail := strings.TrimSpace(o.GetDetail())
	if detail != "" {
		return fmt.Sprintf("%s: %s", reason, detail)
	}
	return reason
}

// formatOverrideReason returns a localized override reason label.
func formatOverrideReason(r statusv1.OverrideReason, loc *message.Printer) string {
	switch r {
	case statusv1.OverrideReason_OVERRIDE_REASON_MAINTENANCE:
		return loc.Sprintf("label.override_maintenance")
	case statusv1.OverrideReason_OVERRIDE_REASON_KNOWN_ISSUE:
		return loc.Sprintf("label.override_known_issue")
	case statusv1.OverrideReason_OVERRIDE_REASON_DEGRADED:
		return loc.Sprintf("label.override_degraded")
	case statusv1.OverrideReason_OVERRIDE_REASON_UNAVAILABLE:
		return loc.Sprintf("label.override_unavailable")
	default:
		return loc.Sprintf("label.unspecified")
	}
}
