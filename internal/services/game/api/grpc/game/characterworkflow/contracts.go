package characterworkflow

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
)

// StepProgress carries one creation step's completion state.
type StepProgress struct {
	Step     int32
	Key      string
	Complete bool
}

// Progress is the game transport shape for character-creation progress.
type Progress struct {
	Steps        []StepProgress
	NextStep     int32
	Ready        bool
	UnmetReasons []string
}

// CampaignContext carries the campaign fields character-creation workflows
// need without leaking full projection record dependencies.
type CampaignContext struct {
	ID     string
	System bridge.SystemID
	Status campaign.Status
}

// CharacterContext carries the character fields character-creation workflows
// need without leaking full projection record dependencies.
type CharacterContext struct {
	Kind character.Kind
}

// CreationDeps provides the game-layer operations a system-specific character
// workflow provider needs.
type CreationDeps interface {
	GetCharacterRecord(ctx context.Context, campaignID, characterID string) (CharacterContext, error)
	GetCharacterSystemProfile(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error)
	SystemContent() contentstore.DaggerheartContentReadStore
	ExecuteProfileReplace(ctx context.Context, campaignContext CampaignContext, characterID string, profile daggerheart.CharacterProfile) error
	ExecuteProfileDelete(ctx context.Context, campaignContext CampaignContext, characterID string) error
	RequireReadPolicy(ctx context.Context, campaignContext CampaignContext) error
	ProfileToProto(campaignID, characterID string, profile projectionstore.DaggerheartCharacterProfile) *campaignv1.CharacterProfile
}

// Provider defines system-specific character-creation workflow behavior behind
// the game character transport contract.
type Provider interface {
	GetProgress(ctx context.Context, deps CreationDeps, campaignContext CampaignContext, characterID string) (Progress, error)
	ApplyStep(ctx context.Context, deps CreationDeps, campaignContext CampaignContext, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.CharacterProfile, Progress, error)
	ApplyWorkflow(ctx context.Context, deps CreationDeps, campaignContext CampaignContext, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.CharacterProfile, Progress, error)
	Reset(ctx context.Context, deps CreationDeps, campaignContext CampaignContext, characterID string) (Progress, error)
}
