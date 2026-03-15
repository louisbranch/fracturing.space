package handler

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// SystemIDFromGameSystemProto maps a proto game system to the domain system ID.
func SystemIDFromGameSystemProto(system commonv1.GameSystem) bridge.SystemID {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return bridge.SystemIDDaggerheart
	default:
		return bridge.SystemIDUnspecified
	}
}

// SystemIDFromCampaignRecord resolves the domain system ID from a campaign
// record's system field.
func SystemIDFromCampaignRecord(record storage.CampaignRecord) bridge.SystemID {
	if normalized, ok := bridge.NormalizeSystemID(record.System.String()); ok {
		return normalized
	}
	return bridge.SystemIDUnspecified
}
