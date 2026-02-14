package projection

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Applier applies event journal entries to projection stores.
type Applier struct {
	Campaign         storage.CampaignStore
	Character        storage.CharacterStore
	CampaignFork     storage.CampaignForkStore
	Daggerheart      storage.DaggerheartStore
	ClaimIndex       storage.ClaimIndexStore
	Invite           storage.InviteStore
	Participant      storage.ParticipantStore
	Session          storage.SessionStore
	SessionGate      storage.SessionGateStore
	SessionSpotlight storage.SessionSpotlightStore
	Adapters         *systems.AdapterRegistry
}

// Apply applies an event to projection stores.
func (a Applier) Apply(ctx context.Context, evt event.Event) error {
	switch evt.Type {
	case event.TypeCampaignCreated:
		return a.applyCampaignCreated(ctx, evt)
	case event.TypeCampaignForked:
		return a.applyCampaignForked(ctx, evt)
	case event.TypeCampaignUpdated:
		return a.applyCampaignUpdated(ctx, evt)
	case event.TypeParticipantJoined:
		return a.applyParticipantJoined(ctx, evt)
	case event.TypeParticipantUpdated:
		return a.applyParticipantUpdated(ctx, evt)
	case event.TypeParticipantLeft:
		return a.applyParticipantLeft(ctx, evt)
	case event.TypeParticipantBound:
		return a.applyParticipantBound(ctx, evt)
	case event.TypeParticipantUnbound:
		return a.applyParticipantUnbound(ctx, evt)
	case event.TypeSeatReassigned:
		return a.applySeatReassigned(ctx, evt)
	case event.TypeInviteCreated:
		return a.applyInviteCreated(ctx, evt)
	case event.TypeInviteClaimed:
		return a.applyInviteClaimed(ctx, evt)
	case event.TypeInviteRevoked:
		return a.applyInviteRevoked(ctx, evt)
	case event.TypeCharacterCreated:
		return a.applyCharacterCreated(ctx, evt)
	case event.TypeCharacterUpdated:
		return a.applyCharacterUpdated(ctx, evt)
	case event.TypeCharacterDeleted:
		return a.applyCharacterDeleted(ctx, evt)
	case event.TypeProfileUpdated:
		return a.applyProfileUpdated(ctx, evt)
	case event.TypeInviteUpdated:
		return a.applyInviteUpdated(ctx, evt)
	case event.TypeSessionStarted:
		return a.applySessionStarted(ctx, evt)
	case event.TypeSessionEnded:
		return a.applySessionEnded(ctx, evt)
	case event.TypeSessionGateOpened:
		return a.applySessionGateOpened(ctx, evt)
	case event.TypeSessionGateResolved:
		return a.applySessionGateResolved(ctx, evt)
	case event.TypeSessionGateAbandoned:
		return a.applySessionGateAbandoned(ctx, evt)
	case event.TypeSessionSpotlightSet:
		return a.applySessionSpotlightSet(ctx, evt)
	case event.TypeSessionSpotlightCleared:
		return a.applySessionSpotlightCleared(ctx, evt)
	default:
		if strings.TrimSpace(evt.SystemID) != "" {
			return a.applySystemEvent(ctx, evt)
		}
		return nil
	}
}

// ensureTimestamp returns the given timestamp in UTC, or the current time if zero.
func ensureTimestamp(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now().UTC()
	}
	return ts.UTC()
}

// marshalResolutionPayload encodes a gate resolution decision and resolution map
// into a single JSON blob. Returns nil when both inputs are empty.
func marshalResolutionPayload(decision string, resolution map[string]any) ([]byte, error) {
	if strings.TrimSpace(decision) == "" && len(resolution) == 0 {
		return nil, nil
	}
	combined := map[string]any{}
	if strings.TrimSpace(decision) != "" {
		combined["decision"] = strings.TrimSpace(decision)
	}
	for key, value := range resolution {
		combined[key] = value
	}
	return json.Marshal(combined)
}

// marshalOptionalMap encodes a map to JSON, returning nil when the map is empty.
func marshalOptionalMap(values map[string]any) ([]byte, error) {
	if len(values) == 0 {
		return nil, nil
	}
	return json.Marshal(values)
}
