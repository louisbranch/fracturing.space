package game

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Stores groups all campaign-related storage interfaces for service injection.
type Stores struct {
	Campaign           storage.CampaignStore
	Participant        storage.ParticipantStore
	ClaimIndex         storage.ClaimIndexStore
	Invite             storage.InviteStore
	Character          storage.CharacterStore
	Daggerheart        storage.DaggerheartStore
	Session            storage.SessionStore
	SessionGate        storage.SessionGateStore
	SessionSpotlight   storage.SessionSpotlightStore
	Event              storage.EventStore
	Telemetry          storage.TelemetryStore
	Statistics         storage.StatisticsStore
	Outcome            storage.RollOutcomeStore
	Snapshot           storage.SnapshotStore
	CampaignFork       storage.CampaignForkStore
	DaggerheartContent storage.DaggerheartContentStore
}

// Applier returns a projection Applier wired to the stores in this bundle.
// The returned Applier can apply any event type; unused stores are simply not
// invoked by the dispatch.
func (s Stores) Applier() projection.Applier {
	return projection.Applier{
		Campaign:         s.Campaign,
		Character:        s.Character,
		CampaignFork:     s.CampaignFork,
		Daggerheart:      s.Daggerheart,
		ClaimIndex:       s.ClaimIndex,
		Invite:           s.Invite,
		Participant:      s.Participant,
		Session:          s.Session,
		SessionGate:      s.SessionGate,
		SessionSpotlight: s.SessionSpotlight,
		Adapters:         adapterRegistryForStores(s),
	}
}

// Validate checks that every store field is non-nil. Call this at service
// construction time so that handlers do not need per-method nil guards.
func (s Stores) Validate() error {
	var missing []string
	if s.Campaign == nil {
		missing = append(missing, "Campaign")
	}
	if s.Participant == nil {
		missing = append(missing, "Participant")
	}
	if s.ClaimIndex == nil {
		missing = append(missing, "ClaimIndex")
	}
	if s.Invite == nil {
		missing = append(missing, "Invite")
	}
	if s.Character == nil {
		missing = append(missing, "Character")
	}
	if s.Daggerheart == nil {
		missing = append(missing, "Daggerheart")
	}
	if s.Session == nil {
		missing = append(missing, "Session")
	}
	if s.SessionGate == nil {
		missing = append(missing, "SessionGate")
	}
	if s.SessionSpotlight == nil {
		missing = append(missing, "SessionSpotlight")
	}
	if s.Event == nil {
		missing = append(missing, "Event")
	}
	if s.Telemetry == nil {
		missing = append(missing, "Telemetry")
	}
	if s.Statistics == nil {
		missing = append(missing, "Statistics")
	}
	if s.Outcome == nil {
		missing = append(missing, "Outcome")
	}
	if s.Snapshot == nil {
		missing = append(missing, "Snapshot")
	}
	if s.CampaignFork == nil {
		missing = append(missing, "CampaignFork")
	}
	if s.DaggerheartContent == nil {
		missing = append(missing, "DaggerheartContent")
	}
	if len(missing) > 0 {
		return fmt.Errorf("stores not configured: %s", strings.Join(missing, ", "))
	}
	return nil
}
