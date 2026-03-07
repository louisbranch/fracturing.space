// Package workflow defines character-creation contracts shared between the game
// transport layer and system providers. The value model is generic, while the
// dependency surface currently reflects the active Daggerheart-backed system
// profile workflow.
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

// CreationDeps provides the game-layer operations that a system workflow
// provider needs. The game package's characterApplication implements this
// interface, bridging authorization, domain execution, and store access.
type CreationDeps interface {
	GetCharacterRecord(ctx context.Context, campaignID, characterID string) (storage.CharacterRecord, error)
	GetCharacterSystemProfile(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error)
	SystemContent() storage.DaggerheartContentReadStore
	ExecuteProfileUpdate(ctx context.Context, campaignRecord storage.CampaignRecord, characterID string, systemProfile map[string]any) error
	RequireReadPolicy(ctx context.Context, campaignRecord storage.CampaignRecord) error
	ProfileToProto(campaignID, characterID string, profile storage.DaggerheartCharacterProfile) *campaignv1.CharacterProfile
}

// Provider defines system-specific workflow behavior behind a common
// CharacterService transport contract.
type Provider interface {
	GetProgress(ctx context.Context, deps CreationDeps, campaignRecord storage.CampaignRecord, characterID string) (Progress, error)
	ApplyStep(ctx context.Context, deps CreationDeps, campaignRecord storage.CampaignRecord, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.CharacterProfile, Progress, error)
	ApplyWorkflow(ctx context.Context, deps CreationDeps, campaignRecord storage.CampaignRecord, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.CharacterProfile, Progress, error)
	Reset(ctx context.Context, deps CreationDeps, campaignRecord storage.CampaignRecord, characterID string) (Progress, error)
}
