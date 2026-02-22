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

func (a outcomeApplication) runApplyAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply attack outcome request is required")
	}
	if a.service.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if a.service.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if a.service.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}
	if len(in.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	c, err := a.service.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart attack outcomes")
	}

	sess, err := a.service.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := a.service.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	rollEvent, err := a.service.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	rollKind := rollKindFromSystemData(rollPayload.SystemData)
	if rollKind == pb.RollKind_ROLL_KIND_REACTION {
		return nil, status.Error(codes.FailedPrecondition, "roll seq references a reaction roll")
	}
	rollOutcome := outcomeFromSystemData(rollPayload.SystemData, rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	crit := critFromSystemData(rollPayload.SystemData, rollOutcome)
	flavor := outcomeFlavorFromCode(rollOutcome)
	if !boolFromSystemData(rollPayload.SystemData, "hope_fear", true) {
		flavor = ""
	}
	rollSuccess, ok := outcomeSuccessFromCode(rollOutcome)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is invalid")
	}
	attackerID := stringFromSystemData(rollPayload.SystemData, "character_id")
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

func (a outcomeApplication) runApplyAdversaryAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply adversary attack outcome request is required")
	}
	if a.service.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if a.service.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if a.service.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}
	if len(in.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}

	c, err := a.service.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversary attack outcomes")
	}

	sess, err := a.service.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := a.service.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	rollEvent, err := a.service.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}
	rollKind := strings.TrimSpace(stringFromSystemData(rollPayload.SystemData, "roll_kind"))
	if rollKind != "adversary_roll" {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference an adversary roll")
	}
	adversaryID := strings.TrimSpace(stringFromSystemData(rollPayload.SystemData, "character_id"))
	if adversaryID == "" {
		adversaryID = strings.TrimSpace(stringFromSystemData(rollPayload.SystemData, "adversary_id"))
	}
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	rollRequestID := strings.TrimSpace(rollEvent.RequestID)
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	targets := normalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	roll, rollHasValue := intFromSystemData(rollPayload.SystemData, "roll")
	if !rollHasValue {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing roll")
	}
	_, hasModifier := intFromSystemData(rollPayload.SystemData, "modifier")
	if !hasModifier {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing modifier")
	}
	total, hasTotal := intFromSystemData(rollPayload.SystemData, "total")
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

func (a outcomeApplication) runApplyReactionOutcome(ctx context.Context, in *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply reaction outcome request is required")
	}
	if a.service.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if a.service.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if a.service.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if a.service.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	c, err := a.service.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart reaction outcomes")
	}

	sess, err := a.service.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := a.service.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	rollEvent, err := a.service.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	rollKind := rollKindFromSystemData(rollPayload.SystemData)
	if rollKind != pb.RollKind_ROLL_KIND_REACTION {
		return nil, status.Error(codes.FailedPrecondition, "roll seq does not reference a reaction roll")
	}
	rollOutcome := outcomeFromSystemData(rollPayload.SystemData, rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	crit := critFromSystemData(rollPayload.SystemData, rollOutcome)
	rollSuccess, ok := outcomeSuccessFromCode(rollOutcome)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is invalid")
	}
	critNegates := boolFromSystemData(rollPayload.SystemData, "crit_negates", crit)
	effectsNegated := crit && critNegates
	actorID := stringFromSystemData(rollPayload.SystemData, "character_id")
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
