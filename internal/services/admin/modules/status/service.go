package status

import (
	"context"
	"net/http"
	"time"

	"github.com/a-h/templ"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	adminerrors "github.com/louisbranch/fracturing.space/internal/services/admin/platform/errors"
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
	ctx, cancel := context.WithTimeout(r.Context(), statusCallTimeout)
	defer cancel()

	resp, err := s.statusClient.GetSystemStatus(ctx, &statusv1.GetSystemStatusRequest{})
	if err != nil {
		// Status service is advisory — degrade gracefully when unreachable.
		adminerrors.LogError(r, "get system status: %v", err)
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
