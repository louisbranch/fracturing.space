package systems

import (
	"net/url"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

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
