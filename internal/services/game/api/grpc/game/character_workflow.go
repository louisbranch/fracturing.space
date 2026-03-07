package game

import (
	"context"
	"encoding/json"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/workflow"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartgrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var characterCreationWorkflowProviders = map[commonv1.GameSystem]workflow.Provider{
	commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART: daggerheartgrpc.CreationWorkflowProvider{},
}

func characterCreationWorkflowProviderForSystem(system commonv1.GameSystem) (workflow.Provider, bool) {
	provider, ok := characterCreationWorkflowProviders[system]
	return provider, ok
}

func (c characterApplication) workflowProviderForCampaign(ctx context.Context, campaignID string) (workflow.CampaignContext, workflow.Provider, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return workflow.CampaignContext{}, nil, err
	}
	provider, ok := characterCreationWorkflowProviderForSystem(campaignRecord.System)
	if !ok {
		return workflow.CampaignContext{}, nil, status.Errorf(codes.Unimplemented, "character creation workflow is not supported for game system %s", campaignRecord.System.String())
	}
	return workflow.CampaignContext{
		ID:     campaignRecord.ID,
		System: campaignRecord.System,
		Status: campaignRecord.Status,
	}, provider, nil
}

func (c characterApplication) GetCharacterCreationProgress(ctx context.Context, campaignID, characterID string) (workflow.Progress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return workflow.Progress{}, err
	}
	return provider.GetProgress(ctx, c.workflowDeps(), campaignRecord, characterID)
}

func (c characterApplication) ApplyCharacterCreationStep(ctx context.Context, campaignID string, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.CharacterProfile, workflow.Progress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return nil, workflow.Progress{}, err
	}
	return provider.ApplyStep(ctx, c.workflowDeps(), campaignRecord, in)
}

func (c characterApplication) ApplyCharacterCreationWorkflow(ctx context.Context, campaignID string, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.CharacterProfile, workflow.Progress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return nil, workflow.Progress{}, err
	}
	return provider.ApplyWorkflow(ctx, c.workflowDeps(), campaignRecord, in)
}

func (c characterApplication) ResetCharacterCreationWorkflow(ctx context.Context, campaignID, characterID string) (workflow.Progress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return workflow.Progress{}, err
	}
	return provider.Reset(ctx, c.workflowDeps(), campaignRecord, characterID)
}

// workflowDeps returns a workflow.CreationDeps implementation backed by this
// characterApplication, bridging game-layer authorization and domain execution
// into the system-agnostic provider interface.
func (c characterApplication) workflowDeps() *characterWorkflowDeps {
	return &characterWorkflowDeps{app: c}
}

type characterWorkflowDeps struct {
	app characterApplication
}

func (d *characterWorkflowDeps) GetCharacterRecord(ctx context.Context, campaignID, characterID string) (workflow.CharacterContext, error) {
	record, err := d.app.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return workflow.CharacterContext{}, err
	}
	return workflow.CharacterContext{Kind: record.Kind}, nil
}

func (d *characterWorkflowDeps) GetCharacterSystemProfile(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	return d.app.stores.SystemStores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
}

func (d *characterWorkflowDeps) SystemContent() storage.DaggerheartContentReadStore {
	return d.app.stores.DaggerheartContent
}

func (d *characterWorkflowDeps) ExecuteProfileUpdate(ctx context.Context, campaignContext workflow.CampaignContext, characterID string, systemProfile map[string]any) error {
	return d.app.executeCharacterProfileUpdate(ctx, campaignContext, characterID, systemProfile)
}

func (d *characterWorkflowDeps) RequireReadPolicy(ctx context.Context, campaignContext workflow.CampaignContext) error {
	return requireReadPolicy(ctx, d.app.stores, storage.CampaignRecord{ID: campaignContext.ID})
}

func (d *characterWorkflowDeps) ProfileToProto(campaignID, characterID string, profile storage.DaggerheartCharacterProfile) *campaignv1.CharacterProfile {
	return daggerheartProfileToProto(campaignID, characterID, profile)
}

// executeCharacterProfileUpdate builds and executes a character profile update
// command through the domain engine.
func (c characterApplication) executeCharacterProfileUpdate(ctx context.Context, campaignContext workflow.CampaignContext, characterID string, systemProfile map[string]any) error {
	policyActor, err := requireCharacterMutationPolicy(ctx, c.stores, storage.CampaignRecord{ID: campaignContext.ID}, characterID)
	if err != nil {
		return err
	}

	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID == "" {
		actorID = strings.TrimSpace(policyActor.ID)
	}
	applier := c.stores.Applier()

	commandPayload := character.ProfileUpdatePayload{
		CharacterID:   characterID,
		SystemProfile: systemProfile,
	}
	commandPayloadJSON, err := json.Marshal(commandPayload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignContext.ID,
			Type:         commandTypeCharacterProfileUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  commandPayloadJSON,
		}),
		domainwrite.Options{},
	)
	return err
}
