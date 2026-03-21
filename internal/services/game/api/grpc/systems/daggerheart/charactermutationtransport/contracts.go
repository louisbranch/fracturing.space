package charactermutationtransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
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
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error)
	GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error)
}

type ContentStore interface {
	GetDaggerheartArmor(ctx context.Context, id string) (contentstore.DaggerheartArmor, error)
	GetDaggerheartBeastform(ctx context.Context, id string) (contentstore.DaggerheartBeastformEntry, error)
	GetDaggerheartClass(ctx context.Context, id string) (contentstore.DaggerheartClass, error)
	GetDaggerheartSubclass(ctx context.Context, id string) (contentstore.DaggerheartSubclass, error)
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
	Content     ContentStore

	ExecuteCharacterCommand func(ctx context.Context, in CharacterCommandInput) error
}
