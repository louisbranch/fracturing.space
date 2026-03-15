package sessionflowtransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
