package daggerheart

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) runSessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session attack flow request is required")
	}
	if err := s.requireDependencies(
		dependencyCampaignStore,
		dependencySessionStore,
		dependencyDaggerheartStore,
		dependencyEventStore,
		dependencySeedGenerator,
	); err != nil {
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

	rollResp, err := s.runSessionActionRoll(ctx, &pb.SessionActionRollRequest{
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

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := s.runApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	attackOutcome, err := s.runApplyAttackOutcome(ctxWithMeta, &pb.DaggerheartApplyAttackOutcomeRequest{
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
	damageRoll, err := s.runSessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
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
		SourceCharacterIds: normalizeTargets(in.GetDamage().GetSourceCharacterIds()),
	}

	applyDamage, err := s.runApplyDamage(ctxWithMeta, &pb.DaggerheartApplyDamageRequest{
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

func (s *DaggerheartService) runSessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session reaction flow request is required")
	}
	if err := s.requireDependencies(
		dependencyCampaignStore,
		dependencySessionStore,
		dependencyDaggerheartStore,
		dependencyEventStore,
		dependencySeedGenerator,
	); err != nil {
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
	actorID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	trait, err := validate.RequiredID(in.GetTrait(), "trait")
	if err != nil {
		return nil, err
	}

	rollResp, err := s.runSessionActionRoll(ctx, &pb.SessionActionRollRequest{
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

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := s.runApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	reactionOutcome, err := s.runApplyReactionOutcome(ctxWithMeta, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	response := &pb.SessionReactionFlowResponse{
		ActionRoll:      rollResp,
		RollOutcome:     rollOutcome,
		ReactionOutcome: reactionOutcome,
	}

	return response, nil
}
