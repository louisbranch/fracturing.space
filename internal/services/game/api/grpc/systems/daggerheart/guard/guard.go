// Package guard provides shared campaign system guards used across Daggerheart
// transport subpackages. Centralised here to avoid import cycles between the
// parent daggerheart package and its subpackages.
package guard

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SessionGateStore is the read contract used by EnsureNoOpenSessionGate.
type SessionGateStore interface {
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error)
}

// CampaignStore is the campaign-read contract used by shared Daggerheart
// transport guards and sibling handler packages.
type CampaignStore interface {
	Get(ctx context.Context, campaignID string) (storage.CampaignRecord, error)
}

// SessionStore is the session-read contract used by shared Daggerheart
// transport helpers and sibling handler packages.
type SessionStore interface {
	GetSession(ctx context.Context, campaignID, sessionID string) (storage.SessionRecord, error)
}

// CampaignSupportsDaggerheart reports whether a campaign record belongs to the
// Daggerheart system.
func CampaignSupportsDaggerheart(record storage.CampaignRecord) bool {
	systemID, ok := bridge.NormalizeSystemID(record.System.String())
	return ok && systemID == bridge.SystemIDDaggerheart
}

// RequireDaggerheartSystem enforces that a campaign belongs to Daggerheart.
func RequireDaggerheartSystem(record storage.CampaignRecord, unsupportedMessage string) error {
	if CampaignSupportsDaggerheart(record) {
		return nil
	}
	return status.Error(codes.FailedPrecondition, unsupportedMessage)
}

// RequireDaggerheartSystemf enforces Daggerheart with a formatted error message.
func RequireDaggerheartSystemf(record storage.CampaignRecord, unsupportedFormat string, args ...any) error {
	if CampaignSupportsDaggerheart(record) {
		return nil
	}
	return status.Errorf(codes.FailedPrecondition, unsupportedFormat, args...)
}

// EnsureNoOpenSessionGate returns an error if a session gate is currently open
// for the given campaign and session.
func EnsureNoOpenSessionGate(ctx context.Context, store SessionGateStore, campaignID, sessionID string) error {
	if store == nil || strings.TrimSpace(campaignID) == "" || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	gate, err := store.GetOpenSessionGate(ctx, campaignID, sessionID)
	if err == nil {
		return status.Errorf(codes.FailedPrecondition, "session gate is open: %s", gate.GateID)
	}
	return grpcerror.OptionalLookupErrorContext(ctx, err, "load session gate")
}
