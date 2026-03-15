package charactermutationtransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore is the campaign-read contract consumed by character mutation
// transport.
type CampaignStore interface {
	Get(ctx context.Context, id string) (storage.CampaignRecord, error)
}

// DaggerheartStore is the character-profile projection contract consumed by
// character mutation transport.
type DaggerheartStore interface {
	GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error)
}

// CharacterCommandInput describes one character-targeted Daggerheart domain
// command emitted by the character mutation transport slice.
type CharacterCommandInput struct {
	CampaignID      string
	CharacterID     string
	CommandType     command.Type
	SessionID       string
	RequestID       string
	InvocationID    string
	PayloadJSON     []byte
	MissingEventMsg string
	ApplyErrMessage string
}

// Dependencies groups the exact reads and callbacks consumed by the character
// mutation transport slice.
type Dependencies struct {
	Campaign    CampaignStore
	Daggerheart DaggerheartStore

	ExecuteCharacterCommand func(ctx context.Context, in CharacterCommandInput) error
}
