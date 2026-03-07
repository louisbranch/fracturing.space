package systems

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	adminerrors "github.com/louisbranch/fracturing.space/internal/services/admin/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// handlers implements the systems Handlers contract.
type handlers struct {
	base         modulehandler.Base
	systemClient statev1.SystemServiceClient
}

// NewHandlers builds the systems handler implementation.
func NewHandlers(base modulehandler.Base, systemClient statev1.SystemServiceClient) Handlers {
	return handlers{base: base, systemClient: systemClient}
}

// HandleSystemsPage renders the systems page fragment or full layout.
func (s handlers) HandleSystemsPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.SystemsPage(loc),
		templates.SystemsFullPage(pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.systems", templates.AppName()),
	)
}

// HandleSystemsTable renders the systems table via HTMX.
func (s handlers) HandleSystemsTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := s.base.Localizer(w, r)
	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.systemClient.ListGameSystems(ctx, &statev1.ListGameSystemsRequest{})
	if err != nil {
		adminerrors.LogError(r, "list game systems: %v", err)
		s.renderSystemsTable(w, r, nil, loc.Sprintf("error.systems_unavailable"), loc)
		return
	}

	systemsList := response.GetSystems()
	if len(systemsList) == 0 {
		s.renderSystemsTable(w, r, nil, loc.Sprintf("error.no_systems"), loc)
		return
	}

	rows := buildSystemRows(systemsList, loc)
	s.renderSystemsTable(w, r, rows, "", loc)
}

// HandleSystemDetail renders the system detail page.
func (s handlers) HandleSystemDetail(w http.ResponseWriter, r *http.Request, systemID string) {
	loc, lang := s.base.Localizer(w, r)
	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	version := strings.TrimSpace(r.URL.Query().Get("version"))
	parsedID := parseSystemID(systemID)
	if parsedID == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		s.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
		return
	}

	response, err := s.systemClient.GetGameSystem(ctx, &statev1.GetGameSystemRequest{
		Id:      parsedID,
		Version: version,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			s.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
			return
		}
		adminerrors.LogError(r, "get game system: %v", err)
		s.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_unavailable"), lang, loc)
		return
	}
	if response.GetSystem() == nil {
		s.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
		return
	}

	detail := buildSystemDetail(response.GetSystem(), loc)
	s.renderSystemDetail(w, r, detail, "", lang, loc)
}

// renderSystemsTable renders a systems table with optional rows and message.
func (s handlers) renderSystemsTable(w http.ResponseWriter, r *http.Request, rows []templates.SystemRow, message string, loc *message.Printer) {
	templ.Handler(templates.SystemsTable(rows, message, loc)).ServeHTTP(w, r)
}

// renderSystemDetail renders the system detail fragment or full layout.
func (s handlers) renderSystemDetail(w http.ResponseWriter, r *http.Request, detail templates.SystemDetail, message string, lang string, loc *message.Printer) {
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.SystemDetailPage(detail, message, loc),
		templates.SystemDetailFullPage(detail, message, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.system", templates.AppName()),
	)
}
