package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// applyStressVulnerableConditionInput groups arguments for applyStressVulnerableCondition.
type applyStressVulnerableConditionInput struct {
	campaignID    string
	sessionID     string
	characterID   string
	conditions    []string
	stressBefore  int
	stressAfter   int
	stressMax     int
	rollSeq       *uint64
	requestID     string
	correlationID string
}

func (s *DaggerheartService) applyStressVulnerableCondition(
	ctx context.Context,
	in applyStressVulnerableConditionInput,
) error {
	effect, err := s.buildStressVulnerableConditionEffect(
		ctx,
		in.campaignID,
		in.sessionID,
		in.characterID,
		in.conditions,
		in.stressBefore,
		in.stressAfter,
		in.stressMax,
		in.rollSeq,
		in.requestID,
	)
	if err != nil {
		return err
	}
	if effect == nil {
		return nil
	}

	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	if err := s.executeAndApplyDaggerheartSystemCommand(ctx, daggerheartSystemCommandInput{
		campaignID:      in.campaignID,
		commandType:     commandTypeDaggerheartConditionChange,
		sessionID:       in.sessionID,
		requestID:       in.requestID,
		invocationID:    invocationID,
		correlationID:   in.correlationID,
		entityType:      "character",
		entityID:        in.characterID,
		payloadJSON:     effect.PayloadJSON,
		missingEventMsg: "condition change did not emit an event",
		applyErrMessage: "apply condition event",
	}); err != nil {
		return err
	}

	return nil
}

func (s *DaggerheartService) buildStressVulnerableConditionEffect(
	ctx context.Context,
	campaignID string,
	sessionID string,
	characterID string,
	conditions []string,
	stressBefore int,
	stressAfter int,
	stressMax int,
	rollSeq *uint64,
	requestID string,
) (*action.OutcomeAppliedEffect, error) {
	if stressMax <= 0 {
		return nil, nil
	}
	if stressBefore == stressAfter {
		return nil, nil
	}
	shouldAdd := stressBefore < stressMax && stressAfter == stressMax
	shouldRemove := stressBefore == stressMax && stressAfter < stressMax
	if !shouldAdd && !shouldRemove {
		return nil, nil
	}

	normalized, err := daggerheart.NormalizeConditions(conditions)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid stored conditions: %v", err)
	}
	hasVulnerable := containsString(normalized, daggerheart.ConditionVulnerable)
	if shouldAdd && hasVulnerable {
		return nil, nil
	}
	if shouldRemove && !hasVulnerable {
		return nil, nil
	}

	afterSet := make(map[string]struct{}, len(normalized)+1)
	for _, value := range normalized {
		afterSet[value] = struct{}{}
	}
	if shouldAdd {
		afterSet[daggerheart.ConditionVulnerable] = struct{}{}
	}
	if shouldRemove {
		delete(afterSet, daggerheart.ConditionVulnerable)
	}
	afterList := make([]string, 0, len(afterSet))
	for value := range afterSet {
		afterList = append(afterList, value)
	}
	after, err := daggerheart.NormalizeConditions(afterList)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid condition set: %v", err)
	}
	added, removed := daggerheart.DiffConditions(normalized, after)
	if len(added) == 0 && len(removed) == 0 {
		return nil, nil
	}
	if rollSeq != nil && *rollSeq > 0 {
		exists, err := s.sessionRequestEventExists(
			ctx,
			campaignID,
			sessionID,
			*rollSeq,
			requestID,
			eventTypeDaggerheartConditionChanged,
			characterID,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "check condition change applied: %v", err)
		}
		if exists {
			return nil, nil
		}
	}

	payload := daggerheart.ConditionChangePayload{
		CharacterID:      ids.CharacterID(characterID),
		ConditionsBefore: normalized,
		ConditionsAfter:  after,
		Added:            added,
		Removed:          removed,
		RollSeq:          rollSeq,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode condition payload: %v", err)
	}
	return &action.OutcomeAppliedEffect{
		Type:          "sys.daggerheart.condition_changed",
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, nil
}

func (s *DaggerheartService) advanceBreathCountdown(
	ctx context.Context,
	campaignID string,
	sessionID string,
	countdownID string,
	failed bool,
) error {
	countdownID = strings.TrimSpace(countdownID)
	if countdownID == "" {
		return nil
	}
	if err := s.requireDependencies(dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return err
	}

	if _, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return handleDomainError(err)
		}
		_, createErr := s.runCreateCountdown(ctx, &pb.DaggerheartCreateCountdownRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			CountdownId: countdownID,
			Name:        daggerheart.BreathCountdownName,
			Kind:        daggerheartCountdownKindToProto(daggerheart.CountdownKindConsequence),
			Current:     daggerheart.BreathCountdownInitial,
			Max:         daggerheart.BreathCountdownMax,
			Direction:   daggerheartCountdownDirectionToProto(daggerheart.CountdownDirectionIncrease),
			Looping:     false,
		})
		if createErr != nil && status.Code(createErr) != codes.FailedPrecondition {
			return createErr
		}
	}

	advance := daggerheart.ResolveBreathCountdownAdvance(failed)
	if _, err := s.runUpdateCountdown(ctx, &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CountdownId: countdownID,
		Delta:       int32(advance.Delta),
		Reason:      advance.Reason,
	}); err != nil {
		return err
	}

	return nil
}

func (s *DaggerheartService) ensureNoOpenSessionGate(ctx context.Context, campaignID, sessionID string) error {
	if err := s.requireDependencies(dependencySessionGateStore); err != nil {
		return err
	}
	if strings.TrimSpace(campaignID) == "" || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	gate, err := s.stores.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID)
	if err == nil {
		return status.Errorf(codes.FailedPrecondition, "session gate is open: %s", gate.GateID)
	}
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	return status.Errorf(codes.Internal, "load session gate: %v", err)
}

func normalizeActionModifiers(modifiers []*pb.ActionRollModifier) (int, []rollModifierMetadata) {
	if len(modifiers) == 0 {
		return 0, nil
	}

	entries := make([]rollModifierMetadata, 0, len(modifiers))
	total := 0
	for _, modifier := range modifiers {
		if modifier == nil {
			continue
		}
		value := int(modifier.GetValue())
		total += value
		entry := rollModifierMetadata{Value: value}
		if source := strings.TrimSpace(modifier.GetSource()); source != "" {
			entry.Source = source
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return total, nil
	}
	return total, entries
}

func normalizeRollKind(kind pb.RollKind) pb.RollKind {
	if kind == pb.RollKind_ROLL_KIND_UNSPECIFIED {
		return pb.RollKind_ROLL_KIND_ACTION
	}
	return kind
}

func withCampaignSessionMetadata(ctx context.Context, campaignID, sessionID string) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	md = metadata.Join(md, metadata.Pairs(grpcmeta.CampaignIDHeader, campaignID, grpcmeta.SessionIDHeader, sessionID))
	return metadata.NewIncomingContext(ctx, md)
}

func containsString(values []string, target string) bool {
	if target == "" {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func damageDiceFromProto(specs []*pb.DiceSpec) ([]daggerheart.DamageDieSpec, error) {
	if len(specs) == 0 {
		return nil, dice.ErrMissingDice
	}
	converted := make([]daggerheart.DamageDieSpec, 0, len(specs))
	for _, spec := range specs {
		if spec == nil {
			return nil, dice.ErrInvalidDiceSpec
		}
		sides := int(spec.GetSides())
		count := int(spec.GetCount())
		if sides <= 0 || count <= 0 {
			return nil, dice.ErrInvalidDiceSpec
		}
		converted = append(converted, daggerheart.DamageDieSpec{Sides: sides, Count: count})
	}
	return converted, nil
}

func diceRollsToProto(rolls []dice.Roll) []*pb.DiceRoll {
	if len(rolls) == 0 {
		return nil
	}
	converted := make([]*pb.DiceRoll, 0, len(rolls))
	for _, roll := range rolls {
		converted = append(converted, &pb.DiceRoll{
			Sides:   int32(roll.Sides),
			Results: int32Slice(roll.Results),
			Total:   int32(roll.Total),
		})
	}
	return converted
}

func resolveRoll(kind pb.RollKind, request daggerheartdomain.ActionRequest) (daggerheartdomain.ActionResult, bool, bool, bool, error) {
	switch normalizeRollKind(kind) {
	case pb.RollKind_ROLL_KIND_REACTION:
		result, err := daggerheartdomain.RollReaction(daggerheartdomain.ReactionRequest{
			Modifier:   request.Modifier,
			Difficulty: request.Difficulty,
			Seed:       request.Seed,
		})
		if err != nil {
			return daggerheartdomain.ActionResult{}, false, false, false, err
		}
		return result.ActionResult, result.GeneratesHopeFear, result.TriggersGMMove, result.CritNegatesEffects, nil
	default:
		result, err := daggerheartdomain.RollAction(request)
		if err != nil {
			return daggerheartdomain.ActionResult{}, true, true, false, err
		}
		return result, true, true, false, nil
	}
}

type hopeSpend struct {
	Source string
	Amount int
}

func hopeSpendsFromModifiers(modifiers []*pb.ActionRollModifier) []hopeSpend {
	if len(modifiers) == 0 {
		return nil
	}

	spends := make([]hopeSpend, 0)
	for _, modifier := range modifiers {
		if modifier == nil {
			continue
		}
		sourceKey := normalizeHopeSpendSource(modifier.GetSource())
		amount := 0
		switch sourceKey {
		case "experience", "help":
			amount = 1
		case "tag_team", "hope_feature":
			amount = 3
		default:
			continue
		}
		spends = append(spends, hopeSpend{Source: sourceKey, Amount: amount})
	}

	if len(spends) == 0 {
		return nil
	}
	return spends
}

func normalizeHopeSpendSource(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ToLower(trimmed)
	replacer := strings.NewReplacer(" ", "_", "-", "_")
	return replacer.Replace(normalized)
}

func normalizeTargets(targets []string) []string {
	if len(targets) == 0 {
		return nil
	}

	result := make([]string, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		trimmed := strings.TrimSpace(target)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

// stringsToCharacterIDs converts a []string to []ids.CharacterID.
func stringsToCharacterIDs(ss []string) []ids.CharacterID {
	if len(ss) == 0 {
		return nil
	}
	result := make([]ids.CharacterID, len(ss))
	for i, s := range ss {
		result[i] = ids.CharacterID(s)
	}
	return result
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func daggerheartRestTypeFromProto(t pb.DaggerheartRestType) (daggerheart.RestType, error) {
	switch t {
	case pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT:
		return daggerheart.RestTypeShort, nil
	case pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG:
		return daggerheart.RestTypeLong, nil
	default:
		return daggerheart.RestTypeShort, errors.New("rest_type is required")
	}
}

func daggerheartRestTypeToString(t daggerheart.RestType) string {
	if t == daggerheart.RestTypeLong {
		return "long"
	}
	return "short"
}

func daggerheartDowntimeMoveFromProto(m pb.DaggerheartDowntimeMove) (daggerheart.DowntimeMove, error) {
	switch m {
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS:
		return daggerheart.DowntimeClearAllStress, nil
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_REPAIR_ALL_ARMOR:
		return daggerheart.DowntimeRepairAllArmor, nil
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_PREPARE:
		return daggerheart.DowntimePrepare, nil
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_WORK_ON_PROJECT:
		return daggerheart.DowntimeWorkOnProject, nil
	default:
		return daggerheart.DowntimePrepare, errors.New("downtime move is required")
	}
}

func daggerheartDowntimeMoveToString(m daggerheart.DowntimeMove) string {
	switch m {
	case daggerheart.DowntimeClearAllStress:
		return "clear_all_stress"
	case daggerheart.DowntimeRepairAllArmor:
		return "repair_all_armor"
	case daggerheart.DowntimePrepare:
		return "prepare"
	case daggerheart.DowntimeWorkOnProject:
		return "work_on_project"
	default:
		return "unknown"
	}
}

// handleDomainError maps domain errors to gRPC status errors with proper codes
// (NotFound, InvalidArgument, FailedPrecondition, etc.) instead of flattening
// everything to codes.Internal.
func handleDomainError(err error) error {
	return grpcerror.HandleDomainError(err)
}
