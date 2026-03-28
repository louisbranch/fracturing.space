package charactermutationtransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyLevelUp validates the next level transition, shapes the advancement
// payload, emits the command, and reloads the resulting profile tier.
func (h *Handler) ApplyLevelUp(ctx context.Context, in *pb.DaggerheartApplyLevelUpRequest) (*pb.DaggerheartApplyLevelUpResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply level up request is required")
	}
	if err := h.requireDependencies(); err != nil {
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

	profile, err := h.validateLevelUpPreconditions(ctx, campaignID, characterID)
	if err != nil {
		return nil, err
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

	advancements := make([]daggerheartpayload.LevelUpAdvancementPayload, 0, len(in.GetAdvancements()))
	for _, adv := range in.GetAdvancements() {
		entry := daggerheartpayload.LevelUpAdvancementPayload{
			Type:            strings.TrimSpace(adv.GetType()),
			Trait:           strings.TrimSpace(adv.GetTrait()),
			DomainCardID:    strings.TrimSpace(adv.GetDomainCardId()),
			DomainCardLevel: int(adv.GetDomainCardLevel()),
		}
		if adv.GetMulticlass() != nil {
			entry.Multiclass = &daggerheartpayload.LevelUpMulticlassPayload{
				SecondaryClassID:    strings.TrimSpace(adv.GetMulticlass().GetSecondaryClassId()),
				SecondarySubclassID: strings.TrimSpace(adv.GetMulticlass().GetSecondarySubclassId()),
				SpellcastTrait:      strings.TrimSpace(adv.GetMulticlass().GetSpellcastTrait()),
				DomainID:            strings.TrimSpace(adv.GetMulticlass().GetDomainId()),
			}
		}
		advancements = append(advancements, entry)
	}
	rewards := make([]daggerheartpayload.LevelUpRewardPayload, 0, len(in.GetRewards()))
	for _, reward := range in.GetRewards() {
		rewards = append(rewards, daggerheartpayload.LevelUpRewardPayload{
			Type:                  strings.TrimSpace(reward.GetType()),
			DomainCardID:          strings.TrimSpace(reward.GetDomainCardId()),
			DomainCardLevel:       int(reward.GetDomainCardLevel()),
			CompanionBonusChoices: int(reward.GetCompanionBonusChoices()),
		})
	}
	subclassTracksAfter, subclassBonuses, err := h.deriveLevelUpSubclassProgression(ctx, profile, advancements)
	if err != nil {
		return nil, err
	}

	payloadJSON, err := json.Marshal(daggerheartpayload.LevelUpApplyPayload{
		CharacterID:                  ids.CharacterID(characterID),
		LevelBefore:                  levelBefore,
		LevelAfter:                   levelAfter,
		Advancements:                 advancements,
		Rewards:                      rewards,
		SubclassTracksAfter:          subclassTracksAfter,
		SubclassHpMaxDelta:           subclassBonuses.HpMaxDelta,
		SubclassStressMaxDelta:       subclassBonuses.StressMaxDelta,
		SubclassEvasionDelta:         subclassBonuses.EvasionDelta,
		SubclassMajorThresholdDelta:  subclassBonuses.MajorThresholdDelta,
		SubclassSevereThresholdDelta: subclassBonuses.SevereThresholdDelta,
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartLevelUpApply,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "level up did not emit an event",
		ApplyErrMessage: "apply level up event",
	}); err != nil {
		return nil, err
	}

	updatedProfile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart profile", err)
	}
	return &pb.DaggerheartApplyLevelUpResponse{
		CharacterId: characterID,
		Level:       int32(updatedProfile.Level),
		Tier:        int32(tierForLevel(updatedProfile.Level)),
	}, nil
}
