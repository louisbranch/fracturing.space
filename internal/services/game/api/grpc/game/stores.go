package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Stores groups all campaign-related storage interfaces for service injection.
type Stores struct {
	Campaign       storage.CampaignStore
	Participant    storage.ParticipantStore
	Invite         storage.InviteStore
	Character      storage.CharacterStore
	ControlDefault storage.ControlDefaultStore
	Daggerheart    storage.DaggerheartStore
	Session        storage.SessionStore
	Event          storage.EventStore
	Telemetry      storage.TelemetryStore
	Outcome        storage.RollOutcomeStore
	Snapshot       storage.SnapshotStore
	CampaignFork   storage.CampaignForkStore
}
