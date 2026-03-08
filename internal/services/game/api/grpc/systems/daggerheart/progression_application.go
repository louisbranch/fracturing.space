package daggerheart

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) runApplyLevelUp(ctx context.Context, in *pb.DaggerheartApplyLevelUpRequest) (*pb.DaggerheartApplyLevelUpResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply level up request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart level up"); err != nil {
		return nil, err
	}

	profile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
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
		CharacterID:        ids.CharacterID(characterID),
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

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    ids.CampaignID(campaignID),
		Type:          commandTypeDaggerheartLevelUpApply,
		ActorType:     command.ActorTypeSystem,
		SessionID:     ids.SessionID(strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))),
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics("level up did not emit an event", "apply level up event"))
	if err != nil {
		return nil, err
	}

	updatedProfile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
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
