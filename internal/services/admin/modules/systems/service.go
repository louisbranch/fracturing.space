package systems

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// service implements systems module handlers using shared module dependencies.
type service struct {
	base modulehandler.Base
}

// NewService builds the systems module service implementation.
func NewService(base modulehandler.Base) Service {
	return service{base: base}
}

// HandleSystemsPage renders the systems page fragment or full layout.
func (s service) HandleSystemsPage(w http.ResponseWriter, r *http.Request) {
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
func (s service) HandleSystemsTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := s.base.Localizer(w, r)
	client := s.base.SystemClient()
	if client == nil {
		s.renderSystemsTable(w, r, nil, loc.Sprintf("error.system_service_unavailable"), loc)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := client.ListGameSystems(ctx, &statev1.ListGameSystemsRequest{})
	if err != nil {
		log.Printf("list game systems: %v", err)
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
func (s service) HandleSystemDetail(w http.ResponseWriter, r *http.Request, systemID string) {
	loc, lang := s.base.Localizer(w, r)
	client := s.base.SystemClient()
	if client == nil {
		s.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_service_unavailable"), lang, loc)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	version := strings.TrimSpace(r.URL.Query().Get("version"))
	parsedID := parseSystemID(systemID)
	if parsedID == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		s.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
		return
	}

	response, err := client.GetGameSystem(ctx, &statev1.GetGameSystemRequest{
		Id:      parsedID,
		Version: version,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			s.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
			return
		}
		log.Printf("get game system: %v", err)
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
func (s service) renderSystemsTable(w http.ResponseWriter, r *http.Request, rows []templates.SystemRow, message string, loc *message.Printer) {
	templ.Handler(templates.SystemsTable(rows, message, loc)).ServeHTTP(w, r)
}

// renderSystemDetail renders the system detail fragment or full layout.
func (s service) renderSystemDetail(w http.ResponseWriter, r *http.Request, detail templates.SystemDetail, message string, lang string, loc *message.Printer) {
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.SystemDetailPage(detail, message, loc),
		templates.SystemDetailFullPage(detail, message, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.system", templates.AppName()),
	)
}

// buildSystemRows formats system rows for the systems table.
func buildSystemRows(systemsList []*statev1.GameSystemInfo, loc *message.Printer) []templates.SystemRow {
	rows := make([]templates.SystemRow, 0, len(systemsList))
	for _, system := range systemsList {
		if system == nil {
			continue
		}

		detailURL := routepath.System(system.GetId().String())
		version := strings.TrimSpace(system.GetVersion())
		if version != "" {
			detailURL = detailURL + "?version=" + url.QueryEscape(version)
		}

		rows = append(rows, templates.SystemRow{
			Name:                system.GetName(),
			Version:             version,
			ImplementationStage: formatImplementationStage(system.GetImplementationStage(), loc),
			OperationalStatus:   formatOperationalStatus(system.GetOperationalStatus(), loc),
			AccessLevel:         formatAccessLevel(system.GetAccessLevel(), loc),
			IsDefault:           system.GetIsDefault(),
			DetailURL:           detailURL,
		})
	}
	return rows
}

// buildSystemDetail formats a system into detail view data.
func buildSystemDetail(system *statev1.GameSystemInfo, loc *message.Printer) templates.SystemDetail {
	if system == nil {
		return templates.SystemDetail{}
	}
	return templates.SystemDetail{
		ID:                  system.GetId().String(),
		Name:                system.GetName(),
		Version:             system.GetVersion(),
		ImplementationStage: formatImplementationStage(system.GetImplementationStage(), loc),
		OperationalStatus:   formatOperationalStatus(system.GetOperationalStatus(), loc),
		AccessLevel:         formatAccessLevel(system.GetAccessLevel(), loc),
		IsDefault:           system.GetIsDefault(),
	}
}

// parseSystemID parses route ids into game system enum values.
func parseSystemID(value string) commonv1.GameSystem {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
	if trimmed == "DAGGERHEART" {
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	}
	if enumValue, ok := commonv1.GameSystem_value[trimmed]; ok {
		return commonv1.GameSystem(enumValue)
	}
	return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
}

// formatImplementationStage returns a localized system implementation stage.
func formatImplementationStage(stage commonv1.GameSystemImplementationStage, loc *message.Printer) string {
	switch stage {
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PLANNED:
		return loc.Sprintf("label.system_stage_planned")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL:
		return loc.Sprintf("label.system_stage_partial")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_COMPLETE:
		return loc.Sprintf("label.system_stage_complete")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_DEPRECATED:
		return loc.Sprintf("label.system_stage_deprecated")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

// formatOperationalStatus returns a localized system operational status.
func formatOperationalStatus(statusValue commonv1.GameSystemOperationalStatus, loc *message.Printer) string {
	switch statusValue {
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OFFLINE:
		return loc.Sprintf("label.system_status_offline")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_DEGRADED:
		return loc.Sprintf("label.system_status_degraded")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL:
		return loc.Sprintf("label.system_status_operational")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_MAINTENANCE:
		return loc.Sprintf("label.system_status_maintenance")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

// formatAccessLevel returns a localized system access-level label.
func formatAccessLevel(level commonv1.GameSystemAccessLevel, loc *message.Printer) string {
	switch level {
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_INTERNAL:
		return loc.Sprintf("label.system_access_internal")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA:
		return loc.Sprintf("label.system_access_beta")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_PUBLIC:
		return loc.Sprintf("label.system_access_public")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_RETIRED:
		return loc.Sprintf("label.system_access_retired")
	default:
		return loc.Sprintf("label.unspecified")
	}
}
