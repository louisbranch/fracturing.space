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
