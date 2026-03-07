package daggerheart

import (
	"context"
	"encoding/json"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// sessionOutcomePrelude holds the validated state shared by all session
// outcome handlers (attack, adversary attack, reaction).
type sessionOutcomePrelude struct {
	campaignID    string
	sessionID     string
	rollPayload   action.RollResolvePayload
	rollMetadata  rollSystemMetadata
	rollRequestID string
}

func (s *DaggerheartService) validateSessionOutcome(
	ctx context.Context,
	sessionID string,
	rollSeq uint64,
) (sessionOutcomePrelude, error) {
	if err := s.requireDependencies(dependencyCampaignStore, dependencySessionStore, dependencyEventStore); err != nil {
		return sessionOutcomePrelude{}, err
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "session id is required")
	}
	if rollSeq == 0 {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return sessionOutcomePrelude{}, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return sessionOutcomePrelude{}, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return sessionOutcomePrelude{}, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart outcomes")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return sessionOutcomePrelude{}, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return sessionOutcomePrelude{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return sessionOutcomePrelude{}, err
	}

	rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, rollSeq)
	if err != nil {
		return sessionOutcomePrelude{}, handleDomainError(err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return sessionOutcomePrelude{}, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}
	rollMetadata, err := decodeRollSystemMetadata(rollPayload.SystemData)
	if err != nil {
		return sessionOutcomePrelude{}, status.Errorf(codes.InvalidArgument, "invalid roll system_data: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	return sessionOutcomePrelude{
		campaignID:    campaignID,
		sessionID:     sessionID,
		rollPayload:   rollPayload,
		rollMetadata:  rollMetadata,
		rollRequestID: rollRequestID,
	}, nil
}

func (s *DaggerheartService) runApplyAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply attack outcome request is required")
	}
	if len(in.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	pre, err := s.validateSessionOutcome(ctx, in.GetSessionId(), in.GetRollSeq())
	if err != nil {
		return nil, err
	}

	rollKind := pre.rollMetadata.rollKindOrDefault()
	if rollKind == pb.RollKind_ROLL_KIND_REACTION {
		return nil, status.Error(codes.FailedPrecondition, "roll seq references a reaction roll")
	}
	rollOutcome := pre.rollMetadata.outcomeOrFallback(pre.rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	crit := boolPointerValue(pre.rollMetadata.Crit, strings.TrimSpace(rollOutcome) == pb.Outcome_CRITICAL_SUCCESS.String())
	flavor := outcomeFlavorFromCode(rollOutcome)
	if !boolPointerValue(pre.rollMetadata.HopeFear, true) {
		flavor = ""
	}
	rollSuccess, ok := outcomeSuccessFromCode(rollOutcome)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is invalid")
	}
	attackerID := strings.TrimSpace(pre.rollMetadata.CharacterID)
	if attackerID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	targets := normalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	return &pb.DaggerheartApplyAttackOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		CharacterId: attackerID,
		Targets:     targets,
		Result: &pb.DaggerheartAttackOutcomeResult{
			Outcome: outcomeCodeToProto(rollOutcome),
			Success: rollSuccess,
			Crit:    crit,
			Flavor:  flavor,
		},
	}, nil
}

func (s *DaggerheartService) runApplyAdversaryAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply adversary attack outcome request is required")
	}
	if len(in.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}

	pre, err := s.validateSessionOutcome(ctx, in.GetSessionId(), in.GetRollSeq())
	if err != nil {
		return nil, err
	}

	rollKind := pre.rollMetadata.rollKindCode()
	if rollKind != "adversary_roll" {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference an adversary roll")
	}
	adversaryID := strings.TrimSpace(pre.rollMetadata.CharacterID)
	if adversaryID == "" {
		adversaryID = strings.TrimSpace(pre.rollMetadata.AdversaryID)
	}
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	targets := normalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	roll, rollHasValue := intPointerValue(pre.rollMetadata.Roll)
	if !rollHasValue {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing roll")
	}
	_, hasModifier := intPointerValue(pre.rollMetadata.Modifier)
	if !hasModifier {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing modifier")
	}
	total, hasTotal := intPointerValue(pre.rollMetadata.Total)
	if !hasTotal {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing total")
	}
	difficulty := int(in.GetDifficulty())
	success := total >= difficulty
	crit := roll == 20

	return &pb.DaggerheartApplyAdversaryAttackOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		AdversaryId: adversaryID,
		Targets:     targets,
		Result: &pb.DaggerheartAdversaryAttackOutcomeResult{
			Success:    success,
			Crit:       crit,
			Roll:       int32(roll),
			Total:      int32(total),
			Difficulty: int32(difficulty),
		},
	}, nil
}

func (s *DaggerheartService) runApplyReactionOutcome(ctx context.Context, in *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply reaction outcome request is required")
	}
	pre, err := s.validateSessionOutcome(ctx, in.GetSessionId(), in.GetRollSeq())
	if err != nil {
		return nil, err
	}

	rollKind := pre.rollMetadata.rollKindOrDefault()
	if rollKind != pb.RollKind_ROLL_KIND_REACTION {
		return nil, status.Error(codes.FailedPrecondition, "roll seq does not reference a reaction roll")
	}
	rollOutcome := pre.rollMetadata.outcomeOrFallback(pre.rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	crit := boolPointerValue(pre.rollMetadata.Crit, strings.TrimSpace(rollOutcome) == pb.Outcome_CRITICAL_SUCCESS.String())
	rollSuccess, ok := outcomeSuccessFromCode(rollOutcome)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is invalid")
	}
	critNegates := boolPointerValue(pre.rollMetadata.CritNegates, crit)
	effectsNegated := crit && critNegates
	actorID := strings.TrimSpace(pre.rollMetadata.CharacterID)
	if actorID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	return &pb.DaggerheartApplyReactionOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		CharacterId: actorID,
		Result: &pb.DaggerheartReactionOutcomeResult{
			Outcome:            outcomeCodeToProto(rollOutcome),
			Success:            rollSuccess,
			Crit:               crit,
			CritNegatesEffects: critNegates,
			EffectsNegated:     effectsNegated,
		},
	}, nil
}
