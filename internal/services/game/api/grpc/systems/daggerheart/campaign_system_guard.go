package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// SessionGateStore is the read contract used by EnsureNoOpenSessionGate.
type SessionGateStore = guard.SessionGateStore

// CampaignSupportsDaggerheart reports whether a campaign record belongs to the
// Daggerheart system.
func CampaignSupportsDaggerheart(record storage.CampaignRecord) bool {
	return guard.CampaignSupportsDaggerheart(record)
}

// RequireDaggerheartSystem enforces that a campaign belongs to Daggerheart.
func RequireDaggerheartSystem(record storage.CampaignRecord, unsupportedMessage string) error {
	return guard.RequireDaggerheartSystem(record, unsupportedMessage)
}

// RequireDaggerheartSystemf enforces Daggerheart with a formatted error message.
func RequireDaggerheartSystemf(record storage.CampaignRecord, unsupportedFormat string, args ...any) error {
	return guard.RequireDaggerheartSystemf(record, unsupportedFormat, args...)
}

// EnsureNoOpenSessionGate returns an error if a session gate is currently open
// for the given campaign and session.
func EnsureNoOpenSessionGate(ctx context.Context, store SessionGateStore, campaignID, sessionID string) error {
	return guard.EnsureNoOpenSessionGate(ctx, store, campaignID, sessionID)
}
