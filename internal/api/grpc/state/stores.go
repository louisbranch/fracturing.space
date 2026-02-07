package state

import (
	"github.com/louisbranch/fracturing.space/internal/storage"
)

// Stores groups all state-related storage interfaces for service injection.
type Stores struct {
	Campaign       storage.CampaignStore
	Participant    storage.ParticipantStore
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
