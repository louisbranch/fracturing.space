package game

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func systemIDFromGameSystemProto(system commonv1.GameSystem) bridge.SystemID {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return bridge.SystemIDDaggerheart
	default:
		return bridge.SystemIDUnspecified
	}
}

func systemIDFromCampaignRecord(record storage.CampaignRecord) bridge.SystemID {
	if normalized, ok := bridge.NormalizeSystemID(record.System.String()); ok {
		return normalized
	}
	return bridge.SystemIDUnspecified
}
