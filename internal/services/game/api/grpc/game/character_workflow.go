package game

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/charactertransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/characterworkflow"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartcreation "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/creationworkflow"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var characterCreationWorkflowProviders = map[bridge.SystemID]characterworkflow.Provider{
	bridge.SystemIDDaggerheart: daggerheartcreation.CreationWorkflowProvider{},
}

func characterCreationWorkflowProviderForSystem(system bridge.SystemID) (characterworkflow.Provider, bool) {
	provider, ok := characterCreationWorkflowProviders[system]
	return provider, ok
}

func (c characterApplication) workflowProviderForCampaign(ctx context.Context, campaignID string) (characterworkflow.CampaignContext, characterworkflow.Provider, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return characterworkflow.CampaignContext{}, nil, err
	}
	systemID := systemIDFromCampaignRecord(campaignRecord)
	provider, ok := characterCreationWorkflowProviderForSystem(systemID)
	if !ok {
		return characterworkflow.CampaignContext{}, nil, status.Errorf(codes.Unimplemented, "character creation workflow is not supported for game system %s", campaignRecord.System.String())
	}
	return characterworkflow.CampaignContext{
		ID:     campaignRecord.ID,
		System: systemID,
		Status: campaignRecord.Status,
	}, provider, nil
}

func (c characterApplication) GetCharacterCreationProgress(ctx context.Context, campaignID, characterID string) (characterworkflow.Progress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return characterworkflow.Progress{}, err
	}
	return provider.GetProgress(ctx, c.workflowDeps(), campaignRecord, characterID)
}

func (c characterApplication) ApplyCharacterCreationStep(ctx context.Context, campaignID string, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.CharacterProfile, characterworkflow.Progress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return nil, characterworkflow.Progress{}, err
	}
	return provider.ApplyStep(ctx, c.workflowDeps(), campaignRecord, in)
}

func (c characterApplication) ApplyCharacterCreationWorkflow(ctx context.Context, campaignID string, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.CharacterProfile, characterworkflow.Progress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return nil, characterworkflow.Progress{}, err
	}
	return provider.ApplyWorkflow(ctx, c.workflowDeps(), campaignRecord, in)
}

func (c characterApplication) ResetCharacterCreationWorkflow(ctx context.Context, campaignID, characterID string) (characterworkflow.Progress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return characterworkflow.Progress{}, err
	}
	return provider.Reset(ctx, c.workflowDeps(), campaignRecord, characterID)
}

// workflowDeps returns a characterworkflow.CreationDeps implementation backed
// by this characterApplication.
func (c characterApplication) workflowDeps() *characterWorkflowDeps {
	return &characterWorkflowDeps{app: c}
}

type characterWorkflowDeps struct {
	app characterApplication
}

func (d *characterWorkflowDeps) GetCharacterRecord(ctx context.Context, campaignID, characterID string) (characterworkflow.CharacterContext, error) {
	record, err := d.app.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return characterworkflow.CharacterContext{}, err
	}
	return characterworkflow.CharacterContext{Kind: record.Kind}, nil
}

func (d *characterWorkflowDeps) GetCharacterSystemProfile(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	return d.app.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
}

func (d *characterWorkflowDeps) SystemContent() contentstore.DaggerheartContentReadStore {
	return d.app.stores.DaggerheartContent
}

func (d *characterWorkflowDeps) ExecuteProfileReplace(ctx context.Context, campaignContext characterworkflow.CampaignContext, characterID string, profile daggerheart.CharacterProfile) error {
	return d.app.executeDaggerheartProfileReplace(ctx, campaignContext, characterID, profile)
}

func (d *characterWorkflowDeps) ExecuteProfileDelete(ctx context.Context, campaignContext characterworkflow.CampaignContext, characterID string) error {
	return d.app.executeDaggerheartProfileDelete(ctx, campaignContext, characterID)
}

func (d *characterWorkflowDeps) RequireReadPolicy(ctx context.Context, campaignContext characterworkflow.CampaignContext) error {
	return requireReadPolicyWithDependencies(ctx, d.app.auth, storage.CampaignRecord{ID: campaignContext.ID})
}

func (d *characterWorkflowDeps) ProfileToProto(campaignID, characterID string, profile projectionstore.DaggerheartCharacterProfile) *campaignv1.CharacterProfile {
	return charactertransport.DaggerheartProfileToProto(campaignID, characterID, profile)
}

func (c characterApplication) executeDaggerheartProfileReplace(ctx context.Context, campaignContext characterworkflow.CampaignContext, characterID string, profile daggerheart.CharacterProfile) error {
	policyActor, err := requireCharacterMutationPolicyWithDependencies(ctx, c.auth, storage.CampaignRecord{ID: campaignContext.ID}, characterID)
	if err != nil {
		return err
	}

	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID == "" {
		actorID = strings.TrimSpace(policyActor.ID)
	}
	commandPayload := daggerheart.CharacterProfileReplacePayload{
		CharacterID: ids.CharacterID(characterID),
		Profile:     profile,
	}
	commandPayloadJSON, err := json.Marshal(commandPayload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.System(commandbuild.SystemInput{
			CoreInput: commandbuild.CoreInput{
				CampaignID:   campaignContext.ID,
				Type:         commandTypeDaggerheartCharacterProfileReplace,
				ActorType:    actorType,
				ActorID:      actorID,
				RequestID:    grpcmeta.RequestIDFromContext(ctx),
				InvocationID: grpcmeta.InvocationIDFromContext(ctx),
				EntityType:   "character",
				EntityID:     characterID,
				PayloadJSON:  commandPayloadJSON,
			},
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
		}),
		domainwrite.Options{},
	)
	return err
}

func (c characterApplication) executeDaggerheartProfileDelete(ctx context.Context, campaignContext characterworkflow.CampaignContext, characterID string) error {
	policyActor, err := requireCharacterMutationPolicyWithDependencies(ctx, c.auth, storage.CampaignRecord{ID: campaignContext.ID}, characterID)
	if err != nil {
		return err
	}

	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID == "" {
		actorID = strings.TrimSpace(policyActor.ID)
	}

	commandPayload := daggerheart.CharacterProfileDeletePayload{CharacterID: ids.CharacterID(characterID)}
	commandPayloadJSON, err := json.Marshal(commandPayload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.System(commandbuild.SystemInput{
			CoreInput: commandbuild.CoreInput{
				CampaignID:   campaignContext.ID,
				Type:         commandTypeDaggerheartCharacterProfileDelete,
				ActorType:    actorType,
				ActorID:      actorID,
				RequestID:    grpcmeta.RequestIDFromContext(ctx),
				InvocationID: grpcmeta.InvocationIDFromContext(ctx),
				EntityType:   "character",
				EntityID:     characterID,
				PayloadJSON:  commandPayloadJSON,
			},
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
		}),
		domainwrite.Options{},
	)
	return err
}
