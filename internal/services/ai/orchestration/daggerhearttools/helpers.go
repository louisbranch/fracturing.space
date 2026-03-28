package daggerhearttools

import (
	"encoding/json"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

type rngRequest struct {
	Seed     *uint64 `json:"seed,omitempty"`
	RollMode string  `json:"roll_mode,omitempty"`
}

type rngResult struct {
	SeedUsed   uint64 `json:"seed_used"`
	RngAlgo    string `json:"rng_algo"`
	SeedSource string `json:"seed_source"`
	RollMode   string `json:"roll_mode"`
}

type experienceSummary struct {
	Name     string `json:"name,omitempty"`
	Modifier int    `json:"modifier,omitempty"`
}

type conditionEntry struct {
	Label         string   `json:"label"`
	ClearTriggers []string `json:"clear_triggers,omitempty"`
}

type adversaryFeatureSummary struct {
	FeatureID       string `json:"feature_id,omitempty"`
	Status          string `json:"status,omitempty"`
	FocusedTargetID string `json:"focused_target_id,omitempty"`
}

type adversarySummary struct {
	ID                string                    `json:"id,omitempty"`
	Name              string                    `json:"name,omitempty"`
	Kind              string                    `json:"kind,omitempty"`
	SceneID           string                    `json:"scene_id,omitempty"`
	Notes             string                    `json:"notes,omitempty"`
	HP                int                       `json:"hp,omitempty"`
	HPMax             int                       `json:"hp_max,omitempty"`
	Stress            int                       `json:"stress,omitempty"`
	StressMax         int                       `json:"stress_max,omitempty"`
	Evasion           int                       `json:"evasion,omitempty"`
	MajorThreshold    int                       `json:"major_threshold,omitempty"`
	SevereThreshold   int                       `json:"severe_threshold,omitempty"`
	Armor             int                       `json:"armor,omitempty"`
	SpotlightGateID   string                    `json:"spotlight_gate_id,omitempty"`
	SpotlightCount    int                       `json:"spotlight_count,omitempty"`
	Conditions        []conditionEntry          `json:"conditions,omitempty"`
	Features          []adversaryFeatureSummary `json:"features,omitempty"`
	PendingExperience *experienceSummary        `json:"pending_experience,omitempty"`
}

type countdownSummary struct {
	ID                string `json:"id,omitempty"`
	Name              string `json:"name,omitempty"`
	Tone              string `json:"tone,omitempty"`
	AdvancementPolicy string `json:"advancement_policy,omitempty"`
	StartingValue     int    `json:"starting_value,omitempty"`
	RemainingValue    int    `json:"remaining_value,omitempty"`
	LoopBehavior      string `json:"loop_behavior,omitempty"`
	Status            string `json:"status,omitempty"`
	LinkedCountdownID string `json:"linked_countdown_id,omitempty"`
}

func toolResultJSON(v any) (orchestration.ToolResult, error) {
	data, _ := json.Marshal(v)
	return orchestration.ToolResult{Output: string(data)}, nil
}

func intSlice(values []int32) []int {
	converted := make([]int, len(values))
	for i, value := range values {
		converted[i] = int(value)
	}
	return converted
}

func rollModeToProto(value string) commonv1.RollMode {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "REPLAY":
		return commonv1.RollMode_REPLAY
	case "LIVE":
		return commonv1.RollMode_LIVE
	default:
		return commonv1.RollMode_ROLL_MODE_UNSPECIFIED
	}
}

func rollModeLabel(value commonv1.RollMode) string {
	switch value {
	case commonv1.RollMode_REPLAY:
		return "REPLAY"
	case commonv1.RollMode_LIVE:
		return "LIVE"
	default:
		return ""
	}
}

func rngRequestToProto(value *rngRequest) *commonv1.RngRequest {
	if value == nil {
		return nil
	}
	req := &commonv1.RngRequest{RollMode: rollModeToProto(value.RollMode)}
	if value.Seed != nil {
		req.Seed = value.Seed
	}
	return req
}

func rngResultFromProto(value *commonv1.RngResponse) *rngResult {
	if value == nil {
		return nil
	}
	return &rngResult{
		SeedUsed:   value.GetSeedUsed(),
		RngAlgo:    value.GetRngAlgo(),
		SeedSource: value.GetSeedSource(),
		RollMode:   rollModeLabel(value.GetRollMode()),
	}
}

func countdownToneToProto(value string) pb.DaggerheartCountdownTone {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "NEUTRAL", "DAGGERHEART_COUNTDOWN_TONE_NEUTRAL":
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_NEUTRAL
	case "PROGRESS", "DAGGERHEART_COUNTDOWN_TONE_PROGRESS":
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS
	case "CONSEQUENCE", "DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE":
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE
	default:
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_UNSPECIFIED
	}
}

func countdownToneToString(value pb.DaggerheartCountdownTone) string {
	switch value {
	case pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_NEUTRAL:
		return "NEUTRAL"
	case pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS:
		return "PROGRESS"
	case pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE:
		return "CONSEQUENCE"
	default:
		return "UNSPECIFIED"
	}
}

func countdownPolicyToProto(value string) pb.DaggerheartCountdownAdvancementPolicy {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MANUAL", "DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL":
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL
	case "ACTION_STANDARD", "DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD":
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD
	case "ACTION_DYNAMIC", "DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC":
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC
	case "LONG_REST", "DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST":
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST
	default:
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_UNSPECIFIED
	}
}

func countdownPolicyToString(value pb.DaggerheartCountdownAdvancementPolicy) string {
	switch value {
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL:
		return "MANUAL"
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD:
		return "ACTION_STANDARD"
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC:
		return "ACTION_DYNAMIC"
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST:
		return "LONG_REST"
	default:
		return "UNSPECIFIED"
	}
}

func countdownLoopBehaviorToProto(value string) pb.DaggerheartCountdownLoopBehavior {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "NONE", "DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE":
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE
	case "RESET", "DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET":
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET
	case "RESET_INCREASE_START", "DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START":
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START
	case "RESET_DECREASE_START", "DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START":
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START
	default:
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_UNSPECIFIED
	}
}

func countdownLoopBehaviorToString(value pb.DaggerheartCountdownLoopBehavior) string {
	switch value {
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE:
		return "NONE"
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET:
		return "RESET"
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START:
		return "RESET_INCREASE_START"
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START:
		return "RESET_DECREASE_START"
	default:
		return "UNSPECIFIED"
	}
}

func countdownStatusToString(value pb.DaggerheartCountdownStatus) string {
	switch value {
	case pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE:
		return "ACTIVE"
	case pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING:
		return "TRIGGER_PENDING"
	default:
		return "UNSPECIFIED"
	}
}

func actionRollContextToProto(value string) pb.ActionRollContext {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MOVE_SILENTLY", "ACTION_ROLL_CONTEXT_MOVE_SILENTLY":
		return pb.ActionRollContext_ACTION_ROLL_CONTEXT_MOVE_SILENTLY
	default:
		return pb.ActionRollContext_ACTION_ROLL_CONTEXT_UNSPECIFIED
	}
}

func daggerheartLifeStateToString(state pb.DaggerheartLifeState) string {
	switch state {
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE:
		return "ALIVE"
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS:
		return "UNCONSCIOUS"
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY:
		return "BLAZE_OF_GLORY"
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD:
		return "DEAD"
	default:
		return "UNSPECIFIED"
	}
}

func daggerheartAttackRangeToProto(value string) pb.DaggerheartAttackRange {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MELEE", "DAGGERHEART_ATTACK_RANGE_MELEE":
		return pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE
	case "RANGED", "DAGGERHEART_ATTACK_RANGE_RANGED":
		return pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_RANGED
	default:
		return pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED
	}
}

func daggerheartDamageTypeToProto(value string) pb.DaggerheartDamageType {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PHYSICAL", "DAGGERHEART_DAMAGE_TYPE_PHYSICAL":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL
	case "MAGIC", "DAGGERHEART_DAMAGE_TYPE_MAGIC":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC
	case "MIXED", "DAGGERHEART_DAMAGE_TYPE_MIXED":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED
	default:
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED
	}
}

func gmMoveKindToProto(value string) pb.DaggerheartGmMoveKind {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "INTERRUPT_AND_MOVE", "DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE":
		return pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE
	case "ADDITIONAL_MOVE", "DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE":
		return pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE
	default:
		return pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_UNSPECIFIED
	}
}

func gmMoveShapeToProto(value string) pb.DaggerheartGmMoveShape {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SHOW_WORLD_REACTION", "DAGGERHEART_GM_MOVE_SHAPE_SHOW_WORLD_REACTION":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHOW_WORLD_REACTION
	case "REVEAL_DANGER", "DAGGERHEART_GM_MOVE_SHAPE_REVEAL_DANGER":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_REVEAL_DANGER
	case "FORCE_SPLIT", "DAGGERHEART_GM_MOVE_SHAPE_FORCE_SPLIT":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_FORCE_SPLIT
	case "MARK_STRESS", "DAGGERHEART_GM_MOVE_SHAPE_MARK_STRESS":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_MARK_STRESS
	case "SHIFT_ENVIRONMENT", "DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT
	case "SPOTLIGHT_ADVERSARY", "DAGGERHEART_GM_MOVE_SHAPE_SPOTLIGHT_ADVERSARY":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SPOTLIGHT_ADVERSARY
	case "CAPTURE_IMPORTANT_TARGET", "DAGGERHEART_GM_MOVE_SHAPE_CAPTURE_IMPORTANT_TARGET":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CAPTURE_IMPORTANT_TARGET
	case "CUSTOM", "DAGGERHEART_GM_MOVE_SHAPE_CUSTOM":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CUSTOM
	default:
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_UNSPECIFIED
	}
}

func sessionActionOutcomeLabel(resp *pb.SessionActionRollResponse) string {
	if resp == nil {
		return ""
	}
	if resp.GetCrit() {
		return pb.Outcome_CRITICAL_SUCCESS.String()
	}
	flavor := strings.ToUpper(strings.TrimSpace(resp.GetFlavor()))
	switch {
	case resp.GetSuccess() && flavor == "HOPE":
		return pb.Outcome_SUCCESS_WITH_HOPE.String()
	case resp.GetSuccess() && flavor == "FEAR":
		return pb.Outcome_SUCCESS_WITH_FEAR.String()
	case !resp.GetSuccess() && flavor == "HOPE":
		return pb.Outcome_FAILURE_WITH_HOPE.String()
	case !resp.GetSuccess() && flavor == "FEAR":
		return pb.Outcome_FAILURE_WITH_FEAR.String()
	default:
		return ""
	}
}

func resolvedOutcomeUpdateFromProto(value *pb.OutcomeUpdated) *resolvedOutcomeUpdate {
	if value == nil {
		return nil
	}
	update := &resolvedOutcomeUpdate{
		CharacterStates: make([]resolvedOutcomeCharacterState, 0, len(value.GetCharacterStates())),
	}
	for _, state := range value.GetCharacterStates() {
		update.CharacterStates = append(update.CharacterStates, resolvedOutcomeCharacterState{
			CharacterID: strings.TrimSpace(state.GetCharacterId()),
			Hope:        int(state.GetHope()),
			Stress:      int(state.GetStress()),
			HP:          int(state.GetHp()),
		})
	}
	if value.GmFear != nil {
		gmFear := int(value.GetGmFear())
		update.GMFear = &gmFear
	}
	if len(update.CharacterStates) == 0 && update.GMFear == nil {
		return nil
	}
	return update
}

func adversarySummaryFromProto(value *pb.DaggerheartAdversary) adversarySummary {
	summary := adversarySummary{
		ID:              value.GetId(),
		Name:            value.GetName(),
		Kind:            strings.TrimSpace(value.GetKind()),
		SceneID:         value.GetSceneId(),
		Notes:           value.GetNotes(),
		HP:              int(value.GetHp()),
		HPMax:           int(value.GetHpMax()),
		Stress:          int(value.GetStress()),
		StressMax:       int(value.GetStressMax()),
		Evasion:         int(value.GetEvasion()),
		MajorThreshold:  int(value.GetMajorThreshold()),
		SevereThreshold: int(value.GetSevereThreshold()),
		Armor:           int(value.GetArmor()),
		SpotlightGateID: strings.TrimSpace(value.GetSpotlightGateId()),
		SpotlightCount:  int(value.GetSpotlightCount()),
		Conditions:      conditionsFromProto(value.GetConditionStates()),
		Features:        adversaryFeaturesFromProto(value.GetFeatureStates()),
	}
	if pending := value.GetPendingExperience(); pending != nil && strings.TrimSpace(pending.GetName()) != "" {
		summary.PendingExperience = &experienceSummary{
			Name:     pending.GetName(),
			Modifier: int(pending.GetModifier()),
		}
	}
	return summary
}

func countdownSummaryFromSceneProto(value *pb.DaggerheartSceneCountdown) countdownSummary {
	if value == nil {
		return countdownSummary{}
	}
	return countdownSummary{
		ID:                strings.TrimSpace(value.GetCountdownId()),
		Name:              strings.TrimSpace(value.GetName()),
		Tone:              countdownToneToString(value.GetTone()),
		AdvancementPolicy: countdownPolicyToString(value.GetAdvancementPolicy()),
		StartingValue:     int(value.GetStartingValue()),
		RemainingValue:    int(value.GetRemainingValue()),
		LoopBehavior:      countdownLoopBehaviorToString(value.GetLoopBehavior()),
		Status:            countdownStatusToString(value.GetStatus()),
		LinkedCountdownID: strings.TrimSpace(value.GetLinkedCountdownId()),
	}
}

func countdownSummaryFromCampaignProto(value *pb.DaggerheartCampaignCountdown) countdownSummary {
	if value == nil {
		return countdownSummary{}
	}
	return countdownSummary{
		ID:                strings.TrimSpace(value.GetCountdownId()),
		Name:              strings.TrimSpace(value.GetName()),
		Tone:              countdownToneToString(value.GetTone()),
		AdvancementPolicy: countdownPolicyToString(value.GetAdvancementPolicy()),
		StartingValue:     int(value.GetStartingValue()),
		RemainingValue:    int(value.GetRemainingValue()),
		LoopBehavior:      countdownLoopBehaviorToString(value.GetLoopBehavior()),
		Status:            countdownStatusToString(value.GetStatus()),
		LinkedCountdownID: strings.TrimSpace(value.GetLinkedCountdownId()),
	}
}

func daggerheartConditionClearTriggerToString(trigger pb.DaggerheartConditionClearTrigger) string {
	switch trigger {
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST:
		return "SHORT_REST"
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST:
		return "LONG_REST"
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END:
		return "SESSION_END"
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN:
		return "DAMAGE_TAKEN"
	default:
		return "UNSPECIFIED"
	}
}

func daggerheartAdversaryFeatureStateStatusToString(value pb.DaggerheartAdversaryFeatureStateStatus) string {
	switch value {
	case pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_READY:
		return "READY"
	case pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_ACTIVE:
		return "ACTIVE"
	case pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_COOLDOWN:
		return "COOLDOWN"
	case pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_SPENT:
		return "SPENT"
	case pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_STAGED:
		return "STAGED"
	default:
		return "UNSPECIFIED"
	}
}

func adversaryFeaturesFromProto(values []*pb.DaggerheartAdversaryFeatureState) []adversaryFeatureSummary {
	result := make([]adversaryFeatureSummary, 0, len(values))
	for _, value := range values {
		result = append(result, adversaryFeatureSummary{
			FeatureID:       strings.TrimSpace(value.GetFeatureId()),
			Status:          daggerheartAdversaryFeatureStateStatusToString(value.GetStatus()),
			FocusedTargetID: strings.TrimSpace(value.GetFocusedTargetId()),
		})
	}
	return result
}

func conditionsFromProto(values []*pb.DaggerheartConditionState) []conditionEntry {
	result := make([]conditionEntry, 0, len(values))
	for _, value := range values {
		label := strings.TrimSpace(value.GetLabel())
		if label == "" {
			label = strings.TrimSpace(value.GetCode())
		}
		if label == "" {
			continue
		}
		clearTriggers := make([]string, 0, len(value.GetClearTriggers()))
		for _, trigger := range value.GetClearTriggers() {
			clearTriggers = append(clearTriggers, daggerheartConditionClearTriggerToString(trigger))
		}
		result = append(result, conditionEntry{Label: label, ClearTriggers: clearTriggers})
	}
	return result
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
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

func rangeInputToRNG(value *rangeInput) *rngRequest {
	if value == nil || value.Seed == nil {
		return nil
	}
	return &rngRequest{Seed: value.Seed}
}

func gmMoveApplyRequestFromInput(campaignID, sessionID, sceneID string, input gmMoveApplyInput) (*pb.DaggerheartApplyGmMoveRequest, error) {
	req := &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		FearSpent:  int32(input.FearSpent),
		SceneId:    sceneID,
	}

	selected := 0
	if input.DirectMove != nil && (strings.TrimSpace(input.DirectMove.Kind) != "" || strings.TrimSpace(input.DirectMove.Shape) != "" || strings.TrimSpace(input.DirectMove.Description) != "" || strings.TrimSpace(input.DirectMove.AdversaryID) != "") {
		selected++
		req.SpendTarget = &pb.DaggerheartApplyGmMoveRequest_DirectMove{
			DirectMove: &pb.DaggerheartDirectGmMoveTarget{
				Kind:        gmMoveKindToProto(input.DirectMove.Kind),
				Shape:       gmMoveShapeToProto(input.DirectMove.Shape),
				Description: strings.TrimSpace(input.DirectMove.Description),
				AdversaryId: strings.TrimSpace(input.DirectMove.AdversaryID),
			},
		}
	}
	if input.AdversaryFeature != nil && (strings.TrimSpace(input.AdversaryFeature.AdversaryID) != "" || strings.TrimSpace(input.AdversaryFeature.FeatureID) != "" || strings.TrimSpace(input.AdversaryFeature.Description) != "") {
		selected++
		req.SpendTarget = &pb.DaggerheartApplyGmMoveRequest_AdversaryFeature{
			AdversaryFeature: &pb.DaggerheartAdversaryFearFeatureTarget{
				AdversaryId: strings.TrimSpace(input.AdversaryFeature.AdversaryID),
				FeatureId:   strings.TrimSpace(input.AdversaryFeature.FeatureID),
				Description: strings.TrimSpace(input.AdversaryFeature.Description),
			},
		}
	}
	if input.EnvironmentFeature != nil && (strings.TrimSpace(input.EnvironmentFeature.EnvironmentEntityID) != "" || strings.TrimSpace(input.EnvironmentFeature.FeatureID) != "" || strings.TrimSpace(input.EnvironmentFeature.Description) != "") {
		selected++
		req.SpendTarget = &pb.DaggerheartApplyGmMoveRequest_EnvironmentFeature{
			EnvironmentFeature: &pb.DaggerheartEnvironmentFearFeatureTarget{
				EnvironmentEntityId: strings.TrimSpace(input.EnvironmentFeature.EnvironmentEntityID),
				FeatureId:           strings.TrimSpace(input.EnvironmentFeature.FeatureID),
				Description:         strings.TrimSpace(input.EnvironmentFeature.Description),
			},
		}
	}
	if input.AdversaryExperience != nil && (strings.TrimSpace(input.AdversaryExperience.AdversaryID) != "" || strings.TrimSpace(input.AdversaryExperience.ExperienceName) != "" || strings.TrimSpace(input.AdversaryExperience.Description) != "") {
		selected++
		req.SpendTarget = &pb.DaggerheartApplyGmMoveRequest_AdversaryExperience{
			AdversaryExperience: &pb.DaggerheartAdversaryExperienceTarget{
				AdversaryId:    strings.TrimSpace(input.AdversaryExperience.AdversaryID),
				ExperienceName: strings.TrimSpace(input.AdversaryExperience.ExperienceName),
				Description:    strings.TrimSpace(input.AdversaryExperience.Description),
			},
		}
	}

	switch {
	case selected == 0:
		return nil, fmt.Errorf("one gm move spend target is required")
	case selected > 1:
		return nil, fmt.Errorf("only one gm move spend target may be provided")
	default:
		return req, nil
	}
}
