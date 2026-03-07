// Package workflow defines the system-agnostic types shared between the game
// service's character creation transport layer and system-specific workflow
// providers (e.g., Daggerheart).
package workflow

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// StepProgress carries one step's completion state.
type StepProgress struct {
	Step     int32
	Key      string
	Complete bool
}

// Progress is a system-agnostic workflow progress shape.
type Progress struct {
	Steps        []StepProgress
	NextStep     int32
	Ready        bool
	UnmetReasons []string
}

// Deps provides the game-layer operations that a system-specific workflow
// provider needs. The game package's characterApplication implements this
// interface, bridging authorization, domain execution, and store access.
type Deps interface {
	GetCharacterRecord(ctx context.Context, campaignID, characterID string) (storage.CharacterRecord, error)
	GetDaggerheartProfile(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error)
	DaggerheartContent() storage.DaggerheartContentReadStore
	ExecuteProfileUpdate(ctx context.Context, campaignRecord storage.CampaignRecord, characterID string, systemProfile map[string]any) error
	RequireReadPolicy(ctx context.Context, campaignRecord storage.CampaignRecord) error
	ProfileToProto(campaignID, characterID string, profile storage.DaggerheartCharacterProfile) *campaignv1.CharacterProfile
}

// Provider defines system-specific workflow behavior behind a common
// CharacterService transport contract.
type Provider interface {
	GetProgress(ctx context.Context, deps Deps, campaignRecord storage.CampaignRecord, characterID string) (Progress, error)
	ApplyStep(ctx context.Context, deps Deps, campaignRecord storage.CampaignRecord, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.CharacterProfile, Progress, error)
	ApplyWorkflow(ctx context.Context, deps Deps, campaignRecord storage.CampaignRecord, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.CharacterProfile, Progress, error)
	Reset(ctx context.Context, deps Deps, campaignRecord storage.CampaignRecord, characterID string) (Progress, error)
}
