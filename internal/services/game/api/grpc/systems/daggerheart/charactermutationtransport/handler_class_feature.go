package charactermutationtransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/statetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyClassFeature resolves one class feature activation into a character
// state patch and emits the corresponding domain command.
func (h *Handler) ApplyClassFeature(ctx context.Context, in *pb.DaggerheartApplyClassFeatureRequest) (*pb.DaggerheartApplyClassFeatureResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply class feature request is required")
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

	profile, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "class feature")
	if err != nil {
		return nil, err
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}

	classState := classStateFromProjection(state.ClassState)
	payload, err := h.resolveClassFeaturePayload(ctx, campaignID, profile, state, classState, in)
	if err != nil {
		return nil, err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode class feature payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartClassFeatureApply,
		SessionID:       strings.TrimSpace(in.GetSessionId()),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "class feature did not emit an event",
		ApplyErrMessage: "apply class feature event",
	}); err != nil {
		return nil, err
	}

	updatedState, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart state", err)
	}
	return &pb.DaggerheartApplyClassFeatureResponse{
		CharacterId: characterID,
		State:       statetransport.CharacterStateToProto(updatedState),
	}, nil
}

// resolveClassFeaturePayload maps the proto oneof variant into the domain
// ClassFeatureApplyPayload that the class-feature decider validates.
func (h *Handler) resolveClassFeaturePayload(
	ctx context.Context,
	campaignID string,
	profile projectionstore.DaggerheartCharacterProfile,
	state projectionstore.DaggerheartCharacterState,
	classState daggerheart.CharacterClassState,
	in *pb.DaggerheartApplyClassFeatureRequest,
) (daggerheart.ClassFeatureApplyPayload, error) {
	payload := daggerheart.ClassFeatureApplyPayload{
		ActorCharacterID: ids.CharacterID(strings.TrimSpace(in.GetCharacterId())),
	}

	switch feature := in.GetFeature().(type) {
	case *pb.DaggerheartApplyClassFeatureRequest_FrontlineTank:
		_ = feature
		classEntry, loadErr := h.deps.Content.GetDaggerheartClass(ctx, profile.ClassID)
		if loadErr != nil {
			return daggerheart.ClassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
		}
		rule := classEntry.HopeFeature.HopeFeatureRule
		if rule == nil {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.Internal, "class has no frontline_tank rule")
		}
		hopeCost := rule.HopeCost
		if state.Hope < hopeCost {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "insufficient hope")
		}
		newArmor := min(state.Armor+rule.Bonus, profile.ArmorMax)
		payload.Feature = "frontline_tank"
		payload.Targets = []daggerheart.ClassFeatureTargetPatchPayload{{
			CharacterID: ids.CharacterID(in.GetCharacterId()),
			HopeBefore:  intPtr(state.Hope),
			HopeAfter:   intPtr(state.Hope - hopeCost),
			ArmorBefore: intPtr(state.Armor),
			ArmorAfter:  intPtr(newArmor),
		}}

	case *pb.DaggerheartApplyClassFeatureRequest_RoguesDodge:
		_ = feature
		next := classState
		next.EvasionBonusUntilHitOrRest = max(next.EvasionBonusUntilHitOrRest, profile.Proficiency)
		payload.Feature = "rogues_dodge"
		payload.Targets = []daggerheart.ClassFeatureTargetPatchPayload{{
			CharacterID:      ids.CharacterID(in.GetCharacterId()),
			ClassStateBefore: classStatePtr(classState),
			ClassStateAfter:  classStatePtr(next),
		}}

	case *pb.DaggerheartApplyClassFeatureRequest_NoMercy:
		_ = feature
		next := classState
		next.AttackBonusUntilRest = max(next.AttackBonusUntilRest, profile.Proficiency)
		payload.Feature = "no_mercy"
		payload.Targets = []daggerheart.ClassFeatureTargetPatchPayload{{
			CharacterID:      ids.CharacterID(in.GetCharacterId()),
			ClassStateBefore: classStatePtr(classState),
			ClassStateAfter:  classStatePtr(next),
		}}

	case *pb.DaggerheartApplyClassFeatureRequest_StrangePatternsChoice:
		number := int(feature.StrangePatternsChoice.GetNumber())
		if number < 1 || number > 12 {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "strange_patterns_choice.number must be in range 1..12")
		}
		next := classState
		next.StrangePatternsNumber = number
		payload.Feature = "strange_patterns_choice"
		payload.Targets = []daggerheart.ClassFeatureTargetPatchPayload{{
			CharacterID:      ids.CharacterID(in.GetCharacterId()),
			ClassStateBefore: classStatePtr(classState),
			ClassStateAfter:  classStatePtr(next),
		}}

	case *pb.DaggerheartApplyClassFeatureRequest_Unstoppable:
		_ = feature
		if classState.Unstoppable.Active {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "unstoppable is already active")
		}
		if classState.Unstoppable.UsedThisLongRest {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "unstoppable already used this long rest")
		}
		next := classState
		next.Unstoppable.Active = true
		next.Unstoppable.UsedThisLongRest = true
		payload.Feature = "unstoppable"
		payload.Targets = []daggerheart.ClassFeatureTargetPatchPayload{{
			CharacterID:      ids.CharacterID(in.GetCharacterId()),
			ClassStateBefore: classStatePtr(classState),
			ClassStateAfter:  classStatePtr(next),
		}}

	case *pb.DaggerheartApplyClassFeatureRequest_Rally:
		targetIDs := uniqueTrimmedIDs(feature.Rally.GetTargetCharacterIds())
		if len(targetIDs) == 0 {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "rally requires at least one target_character_id")
		}
		if len(classState.RallyDice) == 0 {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "no rally dice available")
		}
		next := classState
		next.RallyDice = nil
		payload.Feature = "rally"
		payload.Targets = []daggerheart.ClassFeatureTargetPatchPayload{{
			CharacterID:      ids.CharacterID(in.GetCharacterId()),
			ClassStateBefore: classStatePtr(classState),
			ClassStateAfter:  classStatePtr(next),
		}}
		for _, targetID := range targetIDs {
			targetState, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
			if loadErr != nil {
				return daggerheart.ClassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
			}
			targetProfile, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, targetID)
			if loadErr != nil {
				return daggerheart.ClassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
			}
			hpGain := 0
			for _, die := range classState.RallyDice {
				hpGain += die
			}
			payload.Targets = append(payload.Targets, daggerheart.ClassFeatureTargetPatchPayload{
				CharacterID: ids.CharacterID(targetID),
				HPBefore:    intPtr(targetState.Hp),
				HPAfter:     intPtr(min(targetState.Hp+hpGain, targetProfile.HpMax)),
			})
		}

	case *pb.DaggerheartApplyClassFeatureRequest_MakeAScene:
		targetID := strings.TrimSpace(feature.MakeAScene.GetTargetCharacterId())
		if targetID == "" {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "make_a_scene.target_character_id is required")
		}
		if state.Hope < 1 {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "insufficient hope")
		}
		targetState, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
		if loadErr != nil {
			return daggerheart.ClassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
		}
		payload.Feature = "make_a_scene"
		payload.Targets = []daggerheart.ClassFeatureTargetPatchPayload{
			{
				CharacterID: ids.CharacterID(in.GetCharacterId()),
				HopeBefore:  intPtr(state.Hope),
				HopeAfter:   intPtr(state.Hope - 1),
			},
			{
				CharacterID: ids.CharacterID(targetID),
				HopeBefore:  intPtr(targetState.Hope),
				HopeAfter:   intPtr(min(targetState.Hope+1, targetState.HopeMax)),
			},
		}

	case *pb.DaggerheartApplyClassFeatureRequest_HuntersFocus:
		targetID := strings.TrimSpace(feature.HuntersFocus.GetTargetId())
		if targetID == "" {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "hunters_focus.target_id is required")
		}
		next := classState
		next.FocusTargetID = targetID
		payload.Feature = "hunters_focus"
		payload.Targets = []daggerheart.ClassFeatureTargetPatchPayload{{
			CharacterID:      ids.CharacterID(in.GetCharacterId()),
			ClassStateBefore: classStatePtr(classState),
			ClassStateAfter:  classStatePtr(next),
		}}

	case *pb.DaggerheartApplyClassFeatureRequest_LifeSupport:
		targetID := strings.TrimSpace(feature.LifeSupport.GetTargetCharacterId())
		if targetID == "" {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "life_support.target_character_id is required")
		}
		if state.Hope < 1 {
			return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.FailedPrecondition, "insufficient hope")
		}
		targetState, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, targetID)
		if loadErr != nil {
			return daggerheart.ClassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
		}
		targetProfile, loadErr := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, targetID)
		if loadErr != nil {
			return daggerheart.ClassFeatureApplyPayload{}, grpcerror.HandleDomainError(loadErr)
		}
		payload.Feature = "life_support"
		payload.Targets = []daggerheart.ClassFeatureTargetPatchPayload{
			{
				CharacterID: ids.CharacterID(in.GetCharacterId()),
				HopeBefore:  intPtr(state.Hope),
				HopeAfter:   intPtr(state.Hope - 1),
			},
			{
				CharacterID: ids.CharacterID(targetID),
				HPBefore:    intPtr(targetState.Hp),
				HPAfter:     intPtr(min(targetState.Hp+2, targetProfile.HpMax)),
			},
		}

	default:
		return daggerheart.ClassFeatureApplyPayload{}, status.Error(codes.InvalidArgument, "class feature is required")
	}

	return payload, nil
}
