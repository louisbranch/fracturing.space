package sessionflowtransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the Daggerheart session gameplay flow transport surface behind a
// narrow callback-based seam.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a session flow transport handler.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

// SessionAttackFlow runs the attack workflow by composing a roll, outcome, and
// optional damage application.
func (h *Handler) SessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session attack flow request is required")
	}
	if err := h.requireAttackFlowDeps(); err != nil {
		return nil, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	attackerID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	trait, err := validate.RequiredID(in.GetTrait(), "trait")
	if err != nil {
		return nil, err
	}
	targetID, err := validate.RequiredID(in.GetTargetId(), "target id")
	if err != nil {
		return nil, err
	}
	if in.GetDamage() == nil {
		return nil, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.GetDamage().GetDamageType() == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "damage_type is required")
	}

	rollResp, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:        campaignID,
		SessionId:         sessionID,
		SceneId:           sceneID,
		CharacterId:       attackerID,
		Trait:             trait,
		RollKind:          pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:        in.GetDifficulty(),
		Modifiers:         in.GetModifiers(),
		Underwater:        in.GetUnderwater(),
		BreathCountdownId: in.GetBreathCountdownId(),
		Rng:               in.GetActionRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	attackOutcome, err := h.deps.ApplyAttackOutcome(ctxWithMeta, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
		Targets:   []string{targetID},
	})
	if err != nil {
		return nil, err
	}

	response := &pb.SessionAttackFlowResponse{
		ActionRoll:    rollResp,
		RollOutcome:   rollOutcome,
		AttackOutcome: attackOutcome,
	}

	if attackOutcome.GetResult() == nil || !attackOutcome.GetResult().GetSuccess() {
		return response, nil
	}
	if len(in.GetDamageDice()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "damage_dice are required")
	}

	critical := attackOutcome.GetResult().GetCrit() || in.GetDamageCritical()
	damageRoll, err := h.deps.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: attackerID,
		Dice:        in.GetDamageDice(),
		Modifier:    in.GetDamageModifier(),
		Critical:    critical,
		Rng:         in.GetDamageRng(),
	})
	if err != nil {
		return nil, err
	}

	damageReq := &pb.DaggerheartDamageRequest{
		Amount:             damageRoll.GetTotal(),
		DamageType:         in.GetDamage().GetDamageType(),
		ResistPhysical:     in.GetDamage().GetResistPhysical(),
		ResistMagic:        in.GetDamage().GetResistMagic(),
		ImmunePhysical:     in.GetDamage().GetImmunePhysical(),
		ImmuneMagic:        in.GetDamage().GetImmuneMagic(),
		Direct:             in.GetDamage().GetDirect(),
		MassiveDamage:      in.GetDamage().GetMassiveDamage(),
		Source:             in.GetDamage().GetSource(),
		SourceCharacterIds: workflowtransport.NormalizeTargets(in.GetDamage().GetSourceCharacterIds()),
	}

	applyDamage, err := h.deps.ApplyDamage(ctxWithMeta, &pb.DaggerheartApplyDamageRequest{
		CampaignId:        campaignID,
		SceneId:           sceneID,
		CharacterId:       targetID,
		Damage:            damageReq,
		RollSeq:           &damageRoll.RollSeq,
		RequireDamageRoll: in.GetRequireDamageRoll(),
	})
	if err != nil {
		return nil, err
	}

	response.DamageRoll = damageRoll
	response.DamageApplied = applyDamage
	return response, nil
}

// SessionReactionFlow runs the reaction workflow by composing a reaction roll
// and its outcome handlers.
func (h *Handler) SessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session reaction flow request is required")
	}
	if h.deps.SessionActionRoll == nil || h.deps.ApplyRollOutcome == nil || h.deps.ApplyReactionOutcome == nil {
		return nil, status.Error(codes.Internal, "session workflow dependencies are not configured")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	actorID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	trait, err := validate.RequiredID(in.GetTrait(), "trait")
	if err != nil {
		return nil, err
	}

	rollResp, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:   campaignID,
		SessionId:    sessionID,
		SceneId:      sceneID,
		CharacterId:  actorID,
		Trait:        trait,
		RollKind:     pb.RollKind_ROLL_KIND_REACTION,
		Difficulty:   in.GetDifficulty(),
		Modifiers:    in.GetModifiers(),
		Advantage:    in.GetAdvantage(),
		Disadvantage: in.GetDisadvantage(),
		Rng:          in.GetReactionRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}
	reactionOutcome, err := h.deps.ApplyReactionOutcome(ctxWithMeta, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionReactionFlowResponse{
		ActionRoll:      rollResp,
		RollOutcome:     rollOutcome,
		ReactionOutcome: reactionOutcome,
	}, nil
}

// SessionGroupActionFlow runs the group action orchestration by resolving each
// supporter roll, then the leader roll and its resulting outcome.
func (h *Handler) SessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session group action flow request is required")
	}
	if h.deps.SessionActionRoll == nil || h.deps.ApplyRollOutcome == nil {
		return nil, status.Error(codes.Internal, "session workflow dependencies are not configured")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	leaderID, err := validate.RequiredID(in.GetLeaderCharacterId(), "leader character id")
	if err != nil {
		return nil, err
	}
	leaderTrait, err := validate.RequiredID(in.GetLeaderTrait(), "leader trait")
	if err != nil {
		return nil, err
	}
	if in.GetDifficulty() == 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty is required")
	}
	supporters := in.GetSupporters()
	if len(supporters) == 0 {
		return nil, status.Error(codes.InvalidArgument, "supporters are required")
	}

	supportRolls := make([]*pb.GroupActionSupporterRoll, 0, len(supporters))
	supportSuccesses := 0
	supportFailures := 0
	for _, supporter := range supporters {
		if supporter == nil {
			return nil, status.Error(codes.InvalidArgument, "supporter is required")
		}
		supporterID, err := validate.RequiredID(supporter.GetCharacterId(), "supporter character id")
		if err != nil {
			return nil, err
		}
		supporterTrait, err := validate.RequiredID(supporter.GetTrait(), "supporter trait")
		if err != nil {
			return nil, err
		}

		rollResp, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			SceneId:     sceneID,
			CharacterId: supporterID,
			Trait:       supporterTrait,
			RollKind:    pb.RollKind_ROLL_KIND_REACTION,
			Difficulty:  in.GetDifficulty(),
			Modifiers:   supporter.GetModifiers(),
			Rng:         supporter.GetRng(),
		})
		if err != nil {
			return nil, err
		}
		if rollResp.GetSuccess() {
			supportSuccesses++
		} else {
			supportFailures++
		}
		supportRolls = append(supportRolls, &pb.GroupActionSupporterRoll{
			CharacterId: supporterID,
			ActionRoll:  rollResp,
			Success:     rollResp.GetSuccess(),
		})
	}

	supportModifier := supportSuccesses - supportFailures
	leaderModifiers := append([]*pb.ActionRollModifier{}, in.GetLeaderModifiers()...)
	if supportModifier != 0 {
		leaderModifiers = append(leaderModifiers, &pb.ActionRollModifier{
			Value:  int32(supportModifier),
			Source: "group_action_support",
		})
	}

	leaderRoll, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: leaderID,
		Trait:       leaderTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   leaderModifiers,
		Rng:         in.GetLeaderRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	leaderOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   leaderRoll.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionGroupActionFlowResponse{
		LeaderRoll:       leaderRoll,
		LeaderOutcome:    leaderOutcome,
		SupporterRolls:   supportRolls,
		SupportModifier:  int32(supportModifier),
		SupportSuccesses: int32(supportSuccesses),
		SupportFailures:  int32(supportFailures),
	}, nil
}

// SessionTagTeamFlow runs the tag-team orchestration by resolving both action
// rolls and then applying the chosen result to the combined targets.
func (h *Handler) SessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session tag team flow request is required")
	}
	if h.deps.SessionActionRoll == nil || h.deps.ApplyRollOutcome == nil {
		return nil, status.Error(codes.Internal, "session workflow dependencies are not configured")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if in.GetDifficulty() == 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty is required")
	}
	first := in.GetFirst()
	if first == nil {
		return nil, status.Error(codes.InvalidArgument, "first participant is required")
	}
	second := in.GetSecond()
	if second == nil {
		return nil, status.Error(codes.InvalidArgument, "second participant is required")
	}
	firstID, err := validate.RequiredID(first.GetCharacterId(), "first character id")
	if err != nil {
		return nil, err
	}
	secondID, err := validate.RequiredID(second.GetCharacterId(), "second character id")
	if err != nil {
		return nil, err
	}
	if firstID == secondID {
		return nil, status.Error(codes.InvalidArgument, "tag team participants must be distinct")
	}
	firstTrait, err := validate.RequiredID(first.GetTrait(), "first trait")
	if err != nil {
		return nil, err
	}
	secondTrait, err := validate.RequiredID(second.GetTrait(), "second trait")
	if err != nil {
		return nil, err
	}
	selectedID, err := validate.RequiredID(in.GetSelectedCharacterId(), "selected character id")
	if err != nil {
		return nil, err
	}
	if selectedID != firstID && selectedID != secondID {
		return nil, status.Error(codes.InvalidArgument, "selected character id must match a participant")
	}

	firstRoll, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: firstID,
		Trait:       firstTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   first.GetModifiers(),
		Rng:         first.GetRng(),
	})
	if err != nil {
		return nil, err
	}
	secondRoll, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CharacterId: secondID,
		Trait:       secondTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   second.GetModifiers(),
		Rng:         second.GetRng(),
	})
	if err != nil {
		return nil, err
	}

	selectedRoll := firstRoll
	if selectedID == secondID {
		selectedRoll = secondRoll
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	selectedOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   selectedRoll.GetRollSeq(),
		Targets:   []string{firstID, secondID},
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionTagTeamFlowResponse{
		FirstRoll:           firstRoll,
		SecondRoll:          secondRoll,
		SelectedOutcome:     selectedOutcome,
		SelectedCharacterId: selectedID,
		SelectedRollSeq:     selectedRoll.GetRollSeq(),
	}, nil
}

// SessionAdversaryAttackFlow runs the adversary attack orchestration by
// composing the adversary roll, outcome, and optional damage application.
func (h *Handler) SessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest) (*pb.SessionAdversaryAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack flow request is required")
	}
	if err := h.requireAdversaryAttackFlowDeps(); err != nil {
		return nil, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}
	targetID, err := validate.RequiredID(in.GetTargetId(), "target id")
	if err != nil {
		return nil, err
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}
	if in.GetDamage() == nil {
		return nil, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.GetDamage().GetDamageType() == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "damage_type is required")
	}

	rollResp, err := h.deps.SessionAdversaryAttackRoll(ctx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:     campaignID,
		SessionId:      sessionID,
		AdversaryId:    adversaryID,
		AttackModifier: in.GetAttackModifier(),
		Advantage:      in.GetAdvantage(),
		Disadvantage:   in.GetDisadvantage(),
		Rng:            in.GetAttackRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	attackOutcome, err := h.deps.ApplyAdversaryAttackOutcome(ctxWithMeta, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  sessionID,
		RollSeq:    rollResp.GetRollSeq(),
		Targets:    []string{targetID},
		Difficulty: in.GetDifficulty(),
	})
	if err != nil {
		return nil, err
	}

	response := &pb.SessionAdversaryAttackFlowResponse{
		AttackRoll:    rollResp,
		AttackOutcome: attackOutcome,
	}
	if attackOutcome.GetResult() == nil || !attackOutcome.GetResult().GetSuccess() {
		return response, nil
	}
	if len(in.GetDamageDice()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "damage_dice are required")
	}

	critical := attackOutcome.GetResult().GetCrit() || in.GetDamageCritical()
	damageRoll, err := h.deps.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: adversaryID,
		Dice:        in.GetDamageDice(),
		Modifier:    in.GetDamageModifier(),
		Critical:    critical,
		Rng:         in.GetDamageRng(),
	})
	if err != nil {
		return nil, err
	}

	sourceCharacterIDs := workflowtransport.NormalizeTargets(in.GetDamage().GetSourceCharacterIds())
	sourceCharacterIDs = append(sourceCharacterIDs, adversaryID)
	sourceCharacterIDs = workflowtransport.NormalizeTargets(sourceCharacterIDs)

	damageReq := &pb.DaggerheartDamageRequest{
		Amount:             damageRoll.GetTotal(),
		DamageType:         in.GetDamage().GetDamageType(),
		ResistPhysical:     in.GetDamage().GetResistPhysical(),
		ResistMagic:        in.GetDamage().GetResistMagic(),
		ImmunePhysical:     in.GetDamage().GetImmunePhysical(),
		ImmuneMagic:        in.GetDamage().GetImmuneMagic(),
		Direct:             in.GetDamage().GetDirect(),
		MassiveDamage:      in.GetDamage().GetMassiveDamage(),
		Source:             in.GetDamage().GetSource(),
		SourceCharacterIds: sourceCharacterIDs,
	}

	applyDamage, err := h.deps.ApplyDamage(ctxWithMeta, &pb.DaggerheartApplyDamageRequest{
		CampaignId:        campaignID,
		CharacterId:       targetID,
		Damage:            damageReq,
		RollSeq:           &damageRoll.RollSeq,
		RequireDamageRoll: in.GetRequireDamageRoll(),
	})
	if err != nil {
		return nil, err
	}

	response.DamageRoll = damageRoll
	response.DamageApplied = applyDamage
	return response, nil
}

func (h *Handler) requireAttackFlowDeps() error {
	switch {
	case h.deps.SessionActionRoll == nil:
		return status.Error(codes.Internal, "session action roll handler is not configured")
	case h.deps.SessionDamageRoll == nil:
		return status.Error(codes.Internal, "session damage roll handler is not configured")
	case h.deps.ApplyRollOutcome == nil:
		return status.Error(codes.Internal, "roll outcome handler is not configured")
	case h.deps.ApplyAttackOutcome == nil:
		return status.Error(codes.Internal, "attack outcome handler is not configured")
	case h.deps.ApplyDamage == nil:
		return status.Error(codes.Internal, "apply damage handler is not configured")
	default:
		return nil
	}
}

func (h *Handler) requireAdversaryAttackFlowDeps() error {
	switch {
	case h.deps.SessionAdversaryAttackRoll == nil:
		return status.Error(codes.Internal, "session adversary attack roll handler is not configured")
	case h.deps.SessionDamageRoll == nil:
		return status.Error(codes.Internal, "session damage roll handler is not configured")
	case h.deps.ApplyAdversaryAttackOutcome == nil:
		return status.Error(codes.Internal, "adversary attack outcome handler is not configured")
	case h.deps.ApplyDamage == nil:
		return status.Error(codes.Internal, "apply damage handler is not configured")
	default:
		return nil
	}
}
