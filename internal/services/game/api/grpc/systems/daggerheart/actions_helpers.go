package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) applyStressVulnerableCondition(
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
) error {
	effect, err := s.buildStressVulnerableConditionEffect(
		ctx,
		campaignID,
		sessionID,
		characterID,
		conditions,
		stressBefore,
		stressAfter,
		stressMax,
		rollSeq,
		requestID,
	)
	if err != nil {
		return err
	}
	if effect == nil {
		return nil
	}

	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	if err := s.executeAndApplyDaggerheartSystemCommand(ctx, daggerheartSystemCommandInput{
		campaignID:      campaignID,
		commandType:     commandTypeDaggerheartConditionChange,
		sessionID:       sessionID,
		requestID:       requestID,
		invocationID:    invocationID,
		entityType:      "character",
		entityID:        characterID,
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
		CharacterID:      characterID,
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
	if s.stores.Daggerheart == nil {
		return status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return status.Error(codes.Internal, "event store is not configured")
	}

	if _, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return handleDomainError(err)
		}
		_, createErr := s.CreateCountdown(ctx, &pb.DaggerheartCreateCountdownRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			CountdownId: countdownID,
			Name:        "Breath",
			Kind:        pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE,
			Current:     0,
			Max:         3,
			Direction:   pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
			Looping:     false,
		})
		if createErr != nil && status.Code(createErr) != codes.FailedPrecondition {
			return createErr
		}
	}

	delta := int32(1)
	reason := "breath_tick"
	if failed {
		delta = 2
		reason = "breath_failure"
	}
	if _, err := s.UpdateCountdown(ctx, &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CountdownId: countdownID,
		Delta:       delta,
		Reason:      reason,
	}); err != nil {
		return err
	}

	return nil
}

func (s *DaggerheartService) ensureNoOpenSessionGate(ctx context.Context, campaignID, sessionID string) error {
	if s.stores.SessionGate == nil {
		return status.Error(codes.Internal, "session gate store is not configured")
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

func normalizeActionModifiers(modifiers []*pb.ActionRollModifier) (int, []map[string]any) {
	if len(modifiers) == 0 {
		return 0, nil
	}

	entries := make([]map[string]any, 0, len(modifiers))
	total := 0
	for _, modifier := range modifiers {
		if modifier == nil {
			continue
		}
		value := int(modifier.GetValue())
		total += value
		entry := map[string]any{"value": value}
		if source := strings.TrimSpace(modifier.GetSource()); source != "" {
			entry["source"] = source
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

// applyGMFearSpend mirrors legacy snapshot checks to keep error messaging stable.
func applyGMFearSpend(current, amount int) (int, int, error) {
	if amount <= 0 {
		return 0, 0, errors.New("gm fear amount must be greater than zero")
	}
	if current < amount {
		return 0, 0, errors.New("gm fear is insufficient")
	}
	before := current
	after := before - amount
	return before, after, nil
}

// applyGMFearGain mirrors legacy snapshot checks to keep error messaging stable.
func applyGMFearGain(current, amount int) (int, int, error) {
	if amount <= 0 {
		return 0, 0, errors.New("gm fear amount must be greater than zero")
	}
	before := current
	after := before + amount
	if after > daggerheart.GMFearMax {
		return 0, 0, errors.New("gm fear exceeds cap")
	}
	return before, after, nil
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

func outcomeFlavorFromCode(code string) string {
	switch strings.TrimSpace(code) {
	case pb.Outcome_ROLL_WITH_HOPE.String(),
		pb.Outcome_SUCCESS_WITH_HOPE.String(),
		pb.Outcome_FAILURE_WITH_HOPE.String(),
		pb.Outcome_CRITICAL_SUCCESS.String():
		return "HOPE"
	case pb.Outcome_ROLL_WITH_FEAR.String(),
		pb.Outcome_SUCCESS_WITH_FEAR.String(),
		pb.Outcome_FAILURE_WITH_FEAR.String():
		return "FEAR"
	default:
		return ""
	}
}

func outcomeSuccessFromCode(code string) (bool, bool) {
	switch strings.TrimSpace(code) {
	case pb.Outcome_SUCCESS_WITH_HOPE.String(),
		pb.Outcome_SUCCESS_WITH_FEAR.String(),
		pb.Outcome_CRITICAL_SUCCESS.String():
		return true, true
	case pb.Outcome_FAILURE_WITH_HOPE.String(),
		pb.Outcome_FAILURE_WITH_FEAR.String(),
		pb.Outcome_ROLL_WITH_HOPE.String(),
		pb.Outcome_ROLL_WITH_FEAR.String():
		return false, true
	default:
		return false, false
	}
}

func outcomeCodeToProto(code string) pb.Outcome {
	switch strings.TrimSpace(code) {
	case pb.Outcome_ROLL_WITH_HOPE.String():
		return pb.Outcome_ROLL_WITH_HOPE
	case pb.Outcome_ROLL_WITH_FEAR.String():
		return pb.Outcome_ROLL_WITH_FEAR
	case pb.Outcome_SUCCESS_WITH_HOPE.String():
		return pb.Outcome_SUCCESS_WITH_HOPE
	case pb.Outcome_SUCCESS_WITH_FEAR.String():
		return pb.Outcome_SUCCESS_WITH_FEAR
	case pb.Outcome_FAILURE_WITH_HOPE.String():
		return pb.Outcome_FAILURE_WITH_HOPE
	case pb.Outcome_FAILURE_WITH_FEAR.String():
		return pb.Outcome_FAILURE_WITH_FEAR
	case pb.Outcome_CRITICAL_SUCCESS.String():
		return pb.Outcome_CRITICAL_SUCCESS
	default:
		return pb.Outcome_OUTCOME_UNSPECIFIED
	}
}

func outcomeFromSystemData(systemData map[string]any, fallback string) string {
	if systemData == nil {
		return strings.TrimSpace(fallback)
	}
	if value, ok := systemData["outcome"]; ok {
		if outcome, ok := value.(string); ok {
			return strings.TrimSpace(outcome)
		}
	}
	return strings.TrimSpace(fallback)
}

func rollKindFromSystemData(systemData map[string]any) pb.RollKind {
	if systemData == nil {
		return pb.RollKind_ROLL_KIND_ACTION
	}
	value, ok := systemData["roll_kind"]
	if !ok {
		return pb.RollKind_ROLL_KIND_ACTION
	}
	kind, ok := value.(string)
	if !ok {
		return pb.RollKind_ROLL_KIND_ACTION
	}
	switch strings.TrimSpace(kind) {
	case pb.RollKind_ROLL_KIND_REACTION.String():
		return pb.RollKind_ROLL_KIND_REACTION
	case pb.RollKind_ROLL_KIND_ACTION.String():
		return pb.RollKind_ROLL_KIND_ACTION
	default:
		return pb.RollKind_ROLL_KIND_ACTION
	}
}

func boolFromSystemData(systemData map[string]any, key string, fallback bool) bool {
	if systemData == nil {
		return fallback
	}
	value, ok := systemData[key]
	if !ok {
		return fallback
	}
	boolValue, ok := value.(bool)
	if !ok {
		return fallback
	}
	return boolValue
}

func intFromSystemData(systemData map[string]any, key string) (int, bool) {
	if systemData == nil {
		return 0, false
	}
	value, ok := systemData[key]
	if !ok {
		return 0, false
	}
	switch decoded := value.(type) {
	case float64:
		return int(decoded), true
	case int:
		return decoded, true
	case int64:
		return int(decoded), true
	case float32:
		return int(decoded), true
	case uint64:
		return int(decoded), true
	case uint:
		return int(decoded), true
	case json.Number:
		asInt, err := decoded.Int64()
		if err != nil {
			return 0, false
		}
		return int(asInt), true
	case string:
		intValue, err := strconv.Atoi(strings.TrimSpace(decoded))
		if err != nil {
			return 0, false
		}
		return intValue, true
	default:
		return 0, false
	}
}

func critFromSystemData(systemData map[string]any, outcome string) bool {
	if systemData != nil {
		if value, ok := systemData["crit"]; ok {
			if crit, ok := value.(bool); ok {
				return crit
			}
		}
	}
	return strings.TrimSpace(outcome) == pb.Outcome_CRITICAL_SUCCESS.String()
}

func stringFromSystemData(systemData map[string]any, key string) string {
	if systemData == nil {
		return ""
	}
	value, ok := systemData[key]
	if !ok {
		return ""
	}
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringValue)
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

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func applyDaggerheartDamage(req *pb.DaggerheartDamageRequest, profile storage.DaggerheartCharacterProfile, state storage.DaggerheartCharacterState) (daggerheart.DamageApplication, bool, error) {
	damageTypes := daggerheart.DamageTypes{}
	switch req.DamageType {
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL:
		damageTypes.Physical = true
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		damageTypes.Magic = true
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		damageTypes.Physical = true
		damageTypes.Magic = true
	}

	resistance := daggerheart.ResistanceProfile{
		ResistPhysical: req.ResistPhysical,
		ResistMagic:    req.ResistMagic,
		ImmunePhysical: req.ImmunePhysical,
		ImmuneMagic:    req.ImmuneMagic,
	}
	adjusted := daggerheart.ApplyResistance(int(req.Amount), damageTypes, resistance)
	mitigated := adjusted < int(req.Amount)
	options := daggerheart.DamageOptions{EnableMassiveDamage: req.MassiveDamage}
	result, err := daggerheart.EvaluateDamage(adjusted, profile.MajorThreshold, profile.SevereThreshold, options)
	if err != nil {
		return daggerheart.DamageApplication{}, mitigated, err
	}
	if req.Direct {
		app, err := daggerheart.ApplyDamage(state.Hp, adjusted, profile.MajorThreshold, profile.SevereThreshold, options)
		return app, mitigated, err
	}
	app := daggerheart.ApplyDamageWithArmor(state.Hp, state.Armor, result)
	if app.ArmorSpent > 0 {
		mitigated = true
	}
	return app, mitigated, nil
}

func applyDaggerheartAdversaryDamage(req *pb.DaggerheartDamageRequest, adversary storage.DaggerheartAdversary) (daggerheart.DamageApplication, bool, error) {
	damageTypes := daggerheart.DamageTypes{}
	switch req.DamageType {
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL:
		damageTypes.Physical = true
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		damageTypes.Magic = true
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		damageTypes.Physical = true
		damageTypes.Magic = true
	}

	resistance := daggerheart.ResistanceProfile{
		ResistPhysical: req.ResistPhysical,
		ResistMagic:    req.ResistMagic,
		ImmunePhysical: req.ImmunePhysical,
		ImmuneMagic:    req.ImmuneMagic,
	}
	adjusted := daggerheart.ApplyResistance(int(req.Amount), damageTypes, resistance)
	mitigated := adjusted < int(req.Amount)
	options := daggerheart.DamageOptions{EnableMassiveDamage: req.MassiveDamage}
	result, err := daggerheart.EvaluateDamage(adjusted, adversary.Major, adversary.Severe, options)
	if err != nil {
		return daggerheart.DamageApplication{}, mitigated, err
	}
	if req.Direct {
		app, err := daggerheart.ApplyDamage(adversary.HP, adjusted, adversary.Major, adversary.Severe, options)
		return app, mitigated, err
	}
	app := daggerheart.ApplyDamageWithArmor(adversary.HP, adversary.Armor, result)
	if app.ArmorSpent > 0 {
		mitigated = true
	}
	return app, mitigated, nil
}

func daggerheartSeverityToString(severity daggerheart.DamageSeverity) string {
	switch severity {
	case daggerheart.DamageMinor:
		return "minor"
	case daggerheart.DamageMajor:
		return "major"
	case daggerheart.DamageSevere:
		return "severe"
	case daggerheart.DamageMassive:
		return "massive"
	default:
		return "none"
	}
}

func daggerheartDamageTypeToString(t pb.DaggerheartDamageType) string {
	switch t {
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL:
		return "physical"
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		return "magic"
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		return "mixed"
	default:
		return "unknown"
	}
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

func daggerheartStateToProto(state storage.DaggerheartCharacterState) *pb.DaggerheartCharacterState {
	temporaryArmorBuckets := make([]*pb.DaggerheartTemporaryArmorBucket, 0, len(state.TemporaryArmor))
	for _, bucket := range state.TemporaryArmor {
		temporaryArmorBuckets = append(temporaryArmorBuckets, &pb.DaggerheartTemporaryArmorBucket{
			Source:   bucket.Source,
			Duration: bucket.Duration,
			SourceId: bucket.SourceID,
			Amount:   int32(bucket.Amount),
		})
	}

	return &pb.DaggerheartCharacterState{
		Hp:                    int32(state.Hp),
		Hope:                  int32(state.Hope),
		HopeMax:               int32(state.HopeMax),
		Stress:                int32(state.Stress),
		Armor:                 int32(state.Armor),
		Conditions:            daggerheartConditionsToProto(state.Conditions),
		TemporaryArmorBuckets: temporaryArmorBuckets,
		LifeState:             daggerheartLifeStateToProto(state.LifeState),
	}
}

func optionalInt32(value *int) *int32 {
	if value == nil {
		return nil
	}
	v := int32(*value)
	return &v
}

func handleDomainError(err error) error {
	return status.Errorf(codes.Internal, "%v", err)
}
