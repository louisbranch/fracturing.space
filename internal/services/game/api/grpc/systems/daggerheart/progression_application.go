package daggerheart

import (
	"context"
	"encoding/json"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type progressionApplication struct {
	service *DaggerheartService
}

func newProgressionApplication(service *DaggerheartService) progressionApplication {
	return progressionApplication{service: service}
}

func (a progressionApplication) runApplyLevelUp(ctx context.Context, in *pb.DaggerheartApplyLevelUpRequest) (*pb.DaggerheartApplyLevelUpResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply level up request is required")
	}
	if a.service.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if a.service.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if a.service.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if a.service.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	c, err := a.service.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart level up")
	}

	profile, err := a.service.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	levelBefore := profile.Level
	if levelBefore == 0 {
		levelBefore = 1
	}
	levelAfter := int(in.GetLevelAfter())
	if levelAfter <= 0 {
		return nil, status.Error(codes.InvalidArgument, "level_after must be positive")
	}
	if levelAfter != levelBefore+1 {
		return nil, status.Error(codes.InvalidArgument, "level_after must be exactly one level higher than current level")
	}

	advancements := make([]daggerheart.LevelUpAdvancementPayload, 0, len(in.GetAdvancements()))
	for _, adv := range in.GetAdvancements() {
		entry := daggerheart.LevelUpAdvancementPayload{
			Type:            strings.TrimSpace(adv.GetType()),
			Trait:           strings.TrimSpace(adv.GetTrait()),
			DomainCardID:    strings.TrimSpace(adv.GetDomainCardId()),
			DomainCardLevel: int(adv.GetDomainCardLevel()),
			SubclassCardID:  strings.TrimSpace(adv.GetSubclassCardId()),
		}
		if adv.GetMulticlass() != nil {
			entry.Multiclass = &daggerheart.LevelUpMulticlassPayload{
				SecondaryClassID:    strings.TrimSpace(adv.GetMulticlass().GetSecondaryClassId()),
				SecondarySubclassID: strings.TrimSpace(adv.GetMulticlass().GetSecondarySubclassId()),
				FoundationCardID:    strings.TrimSpace(adv.GetMulticlass().GetFoundationCardId()),
				SpellcastTrait:      strings.TrimSpace(adv.GetMulticlass().GetSpellcastTrait()),
				DomainID:            strings.TrimSpace(adv.GetMulticlass().GetDomainId()),
			}
		}
		advancements = append(advancements, entry)
	}

	payload := daggerheart.LevelUpApplyPayload{
		CharacterID:        characterID,
		LevelBefore:        levelBefore,
		LevelAfter:         levelAfter,
		Advancements:       advancements,
		NewDomainCardID:    strings.TrimSpace(in.GetNewDomainCardId()),
		NewDomainCardLevel: int(in.GetNewDomainCardLevel()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(a.service.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = a.service.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartLevelUpApply,
		ActorType:     command.ActorTypeSystem,
		SessionID:     strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "level up did not emit an event",
		applyErrMessage: "apply level up event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}

	updatedProfile, err := a.service.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart profile: %v", err)
	}

	return &pb.DaggerheartApplyLevelUpResponse{
		CharacterId: characterID,
		Level:       int32(updatedProfile.Level),
		Tier:        int32(tierForLevel(updatedProfile.Level)),
	}, nil
}

// tierForLevel derives the tier from a character level, mirroring the SRD tier table.
func tierForLevel(level int) int {
	switch {
	case level <= 1:
		return 1
	case level <= 4:
		return 2
	case level <= 7:
		return 3
	default:
		return 4
	}
}
