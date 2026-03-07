package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// campaignSupportsDaggerheart reports whether a campaign record belongs to the
// Daggerheart system.
func campaignSupportsDaggerheart(record storage.CampaignRecord) bool {
	systemID, ok := bridge.NormalizeSystemID(record.System.String())
	return ok && systemID == bridge.SystemIDDaggerheart
}

// requireDaggerheartSystem enforces that a campaign belongs to Daggerheart.
func requireDaggerheartSystem(record storage.CampaignRecord, unsupportedMessage string) error {
	if campaignSupportsDaggerheart(record) {
		return nil
	}
	return status.Error(codes.FailedPrecondition, unsupportedMessage)
}

// requireDaggerheartSystemf enforces Daggerheart with a formatted error message.
func requireDaggerheartSystemf(record storage.CampaignRecord, unsupportedFormat string, args ...any) error {
	if campaignSupportsDaggerheart(record) {
		return nil
	}
	return status.Errorf(codes.FailedPrecondition, unsupportedFormat, args...)
}
