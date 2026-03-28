package daggerhearttools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

type actionRollResolveInput struct {
	CharacterID            string               `json:"character_id"`
	Trait                  string               `json:"trait"`
	Difficulty             int                  `json:"difficulty"`
	Modifiers              []actionRollModifier `json:"modifiers"`
	Advantage              int                  `json:"advantage"`
	Disadvantage           int                  `json:"disadvantage"`
	Underwater             bool                 `json:"underwater"`
	BreathSceneCountdownID string               `json:"breath_scene_countdown_id,omitempty"`
	SceneID                string               `json:"scene_id,omitempty"`
	ReplaceHopeWithArmor   bool                 `json:"replace_hope_with_armor,omitempty"`
	Context                string               `json:"context,omitempty"`
	Targets                []string             `json:"targets,omitempty"`
	SwapHopeFear           bool                 `json:"swap_hope_fear,omitempty"`
	Rng                    *rngRequest          `json:"rng,omitempty"`
}

type actionRollModifier struct {
	Source string `json:"source,omitempty"`
	Value  int    `json:"value"`
}

type gmMoveApplyInput struct {
	FearSpent           int                             `json:"fear_spent"`
	SceneID             string                          `json:"scene_id,omitempty"`
	DirectMove          *gmMoveDirectMoveInput          `json:"direct_move,omitempty"`
	AdversaryFeature    *gmMoveAdversaryFeatureInput    `json:"adversary_feature,omitempty"`
	EnvironmentFeature  *gmMoveEnvironmentFeatureInput  `json:"environment_feature,omitempty"`
	AdversaryExperience *gmMoveAdversaryExperienceInput `json:"adversary_experience,omitempty"`
}

type gmMoveDirectMoveInput struct {
	Kind        string `json:"kind"`
	Shape       string `json:"shape"`
	Description string `json:"description,omitempty"`
	AdversaryID string `json:"adversary_id,omitempty"`
}

type gmMoveAdversaryFeatureInput struct {
	AdversaryID string `json:"adversary_id"`
	FeatureID   string `json:"feature_id"`
	Description string `json:"description,omitempty"`
}

type gmMoveEnvironmentFeatureInput struct {
	EnvironmentEntityID string `json:"environment_entity_id"`
	FeatureID           string `json:"feature_id"`
	Description         string `json:"description,omitempty"`
}

type gmMoveAdversaryExperienceInput struct {
	AdversaryID    string `json:"adversary_id"`
	ExperienceName string `json:"experience_name"`
	Description    string `json:"description,omitempty"`
}

type adversaryCreateInput struct {
	SceneID          string `json:"scene_id,omitempty"`
	AdversaryEntryID string `json:"adversary_entry_id"`
	Notes            string `json:"notes,omitempty"`
}

type countdownCreateInput struct {
	SceneID            string      `json:"scene_id,omitempty"`
	CountdownID        string      `json:"countdown_id,omitempty"`
	Name               string      `json:"name"`
	Tone               string      `json:"tone"`
	AdvancementPolicy  string      `json:"advancement_policy"`
	FixedStartingValue int         `json:"fixed_starting_value,omitempty"`
	RandomizedStart    *rangeInput `json:"randomized_start,omitempty"`
	LoopBehavior       string      `json:"loop_behavior"`
	LinkedCountdownID  string      `json:"linked_countdown_id,omitempty"`
}

type countdownAdvanceInput struct {
	SceneID     string `json:"scene_id,omitempty"`
	CountdownID string `json:"countdown_id"`
	Amount      int    `json:"amount"`
	Reason      string `json:"reason,omitempty"`
}

type countdownResolveTriggerInput struct {
	SceneID     string `json:"scene_id,omitempty"`
	CountdownID string `json:"countdown_id"`
	Reason      string `json:"reason,omitempty"`
}

type rangeInput struct {
	Min  int     `json:"min"`
	Max  int     `json:"max"`
	Seed *uint64 `json:"seed,omitempty"`
}

type adversaryUpdateInput struct {
	AdversaryID string  `json:"adversary_id"`
	SceneID     string  `json:"scene_id,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

type actionRollResolveResult struct {
	ActionRoll  resolvedActionRollSummary  `json:"action_roll"`
	RollOutcome resolvedRollOutcomeSummary `json:"roll_outcome"`
}

type resolvedActionRollSummary struct {
	RollSeq    uint64     `json:"roll_seq"`
	HopeDie    int        `json:"hope_die"`
	FearDie    int        `json:"fear_die"`
	Total      int        `json:"total"`
	Difficulty int        `json:"difficulty"`
	Success    bool       `json:"success"`
	Flavor     string     `json:"flavor,omitempty"`
	Crit       bool       `json:"crit"`
	Outcome    string     `json:"outcome,omitempty"`
	Rng        *rngResult `json:"rng,omitempty"`
}

type resolvedRollOutcomeSummary struct {
	RollSeq              uint64                 `json:"roll_seq"`
	RequiresComplication bool                   `json:"requires_complication"`
	Updated              *resolvedOutcomeUpdate `json:"updated,omitempty"`
}

type resolvedOutcomeUpdate struct {
	CharacterStates []resolvedOutcomeCharacterState `json:"character_states,omitempty"`
	GMFear          *int                            `json:"gm_fear,omitempty"`
}

type resolvedOutcomeCharacterState struct {
	CharacterID string `json:"character_id"`
	Hope        int    `json:"hope"`
	Stress      int    `json:"stress"`
	HP          int    `json:"hp"`
}

type gmMoveApplyResult struct {
	GMFearBefore int `json:"gm_fear_before"`
	GMFearAfter  int `json:"gm_fear_after"`
}

type countdownAdvanceResult struct {
	Countdown countdownSummary        `json:"countdown"`
	Advance   countdownAdvanceSummary `json:"advance"`
}

type countdownAdvanceSummary struct {
	BeforeRemaining int    `json:"before_remaining"`
	AfterRemaining  int    `json:"after_remaining"`
	AdvancedBy      int    `json:"advanced_by"`
	StatusBefore    string `json:"status_before,omitempty"`
	StatusAfter     string `json:"status_after,omitempty"`
	Triggered       bool   `json:"triggered"`
	Reason          string `json:"reason,omitempty"`
}

// ActionRollResolve runs the authoritative Daggerheart action-roll workflow in one tool call.
func ActionRollResolve(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input actionRollResolveInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	characterID := strings.TrimSpace(input.CharacterID)
	if characterID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("character_id is required")
	}
	trait := strings.TrimSpace(input.Trait)
	if trait == "" {
		return orchestration.ToolResult{}, fmt.Errorf("trait is required")
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}

	rollResp, err := runtime.DaggerheartClient().SessionActionRoll(callCtx, &pb.SessionActionRollRequest{
		CampaignId:             campaignID,
		SessionId:              sessionID,
		SceneId:                sceneID,
		CharacterId:            characterID,
		Trait:                  trait,
		RollKind:               pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:             int32(input.Difficulty),
		Modifiers:              actionRollModifiersToProto(input.Modifiers),
		Advantage:              int32(input.Advantage),
		Disadvantage:           int32(input.Disadvantage),
		Underwater:             input.Underwater,
		BreathSceneCountdownId: strings.TrimSpace(input.BreathSceneCountdownID),
		Rng:                    rngRequestToProto(input.Rng),
		ReplaceHopeWithArmor:   input.ReplaceHopeWithArmor,
		Context:                actionRollContextToProto(input.Context),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("session action roll failed: %w", err)
	}
	if rollResp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("session action roll response is missing")
	}

	outcomeResp, err := runtime.DaggerheartClient().ApplyRollOutcome(callCtx, &pb.ApplyRollOutcomeRequest{
		SessionId:    sessionID,
		RollSeq:      rollResp.GetRollSeq(),
		Targets:      compactStrings(input.Targets),
		SceneId:      sceneID,
		SwapHopeFear: input.SwapHopeFear,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("apply roll outcome failed: %w", err)
	}
	if outcomeResp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("apply roll outcome response is missing")
	}

	return toolResultJSON(actionRollResolveResult{
		ActionRoll: resolvedActionRollSummary{
			RollSeq:    rollResp.GetRollSeq(),
			HopeDie:    int(rollResp.GetHopeDie()),
			FearDie:    int(rollResp.GetFearDie()),
			Total:      int(rollResp.GetTotal()),
			Difficulty: int(rollResp.GetDifficulty()),
			Success:    rollResp.GetSuccess(),
			Flavor:     strings.TrimSpace(rollResp.GetFlavor()),
			Crit:       rollResp.GetCrit(),
			Outcome:    sessionActionOutcomeLabel(rollResp),
			Rng:        rngResultFromProto(rollResp.GetRng()),
		},
		RollOutcome: resolvedRollOutcomeSummary{
			RollSeq:              outcomeResp.GetRollSeq(),
			RequiresComplication: outcomeResp.GetRequiresComplication(),
			Updated:              resolvedOutcomeUpdateFromProto(outcomeResp.GetUpdated()),
		},
	})
}

// GmMoveApply spends Fear through the authoritative GM-move RPC.
func GmMoveApply(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input gmMoveApplyInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}

	req, err := gmMoveApplyRequestFromInput(campaignID, sessionID, sceneID, input)
	if err != nil {
		return orchestration.ToolResult{}, err
	}

	resp, err := runtime.DaggerheartClient().ApplyGmMove(callCtx, req)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("apply gm move failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("apply gm move response is missing")
	}
	return toolResultJSON(gmMoveApplyResult{
		GMFearBefore: int(resp.GetGmFearBefore()),
		GMFearAfter:  int(resp.GetGmFearAfter()),
	})
}

// AdversaryCreate creates one Daggerheart adversary on the current scene.
func AdversaryCreate(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input adversaryCreateInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	adversaryEntryID := strings.TrimSpace(input.AdversaryEntryID)
	if adversaryEntryID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("adversary_entry_id is required")
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}

	resp, err := runtime.DaggerheartClient().CreateAdversary(callCtx, &pb.DaggerheartCreateAdversaryRequest{
		CampaignId:       campaignID,
		SessionId:        sessionID,
		SceneId:          sceneID,
		AdversaryEntryId: adversaryEntryID,
		Notes:            strings.TrimSpace(input.Notes),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("create adversary failed: %w", err)
	}
	if resp == nil || resp.GetAdversary() == nil {
		return orchestration.ToolResult{}, fmt.Errorf("create adversary response is missing")
	}
	return toolResultJSON(adversarySummaryFromProto(resp.GetAdversary()))
}

// CountdownCreate creates one Daggerheart scene countdown on the current scene.
func CountdownCreate(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input countdownCreateInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return orchestration.ToolResult{}, fmt.Errorf("name is required")
	}
	tone := countdownToneToProto(input.Tone)
	if tone == pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_UNSPECIFIED {
		return orchestration.ToolResult{}, fmt.Errorf("tone must be NEUTRAL, PROGRESS, or CONSEQUENCE")
	}
	policy := countdownPolicyToProto(input.AdvancementPolicy)
	if policy == pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_UNSPECIFIED {
		return orchestration.ToolResult{}, fmt.Errorf("advancement_policy must be MANUAL, ACTION_STANDARD, ACTION_DYNAMIC, or LONG_REST")
	}
	loopBehavior := countdownLoopBehaviorToProto(input.LoopBehavior)
	if loopBehavior == pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_UNSPECIFIED {
		return orchestration.ToolResult{}, fmt.Errorf("loop_behavior must be NONE, RESET, RESET_INCREASE_START, or RESET_DECREASE_START")
	}

	input = normalizeCountdownCreateInput(input)

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}

	req := &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId:        campaignID,
		SessionId:         sessionID,
		SceneId:           sceneID,
		CountdownId:       strings.TrimSpace(input.CountdownID),
		Name:              name,
		Tone:              tone,
		AdvancementPolicy: policy,
		LoopBehavior:      loopBehavior,
		LinkedCountdownId: strings.TrimSpace(input.LinkedCountdownID),
	}
	if input.RandomizedStart != nil {
		req.StartingValue = &pb.DaggerheartCreateSceneCountdownRequest_RandomizedStart{
			RandomizedStart: &pb.DaggerheartCountdownRandomizedStart{
				Min: int32(input.RandomizedStart.Min),
				Max: int32(input.RandomizedStart.Max),
				Rng: rngRequestToProto(rangeInputToRNG(input.RandomizedStart)),
			},
		}
	} else {
		req.StartingValue = &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{
			FixedStartingValue: int32(input.FixedStartingValue),
		}
	}

	resp, err := runtime.DaggerheartClient().CreateSceneCountdown(callCtx, req)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("create countdown failed: %w", err)
	}
	if resp == nil || resp.GetCountdown() == nil {
		return orchestration.ToolResult{}, fmt.Errorf("create countdown response is missing")
	}
	return toolResultJSON(countdownSummaryFromSceneProto(resp.GetCountdown()))
}

func normalizeCountdownCreateInput(input countdownCreateInput) countdownCreateInput {
	if input.RandomizedStart == nil {
		return input
	}
	if input.FixedStartingValue > 0 {
		if input.RandomizedStart.Min <= 0 || input.RandomizedStart.Max <= 0 || input.RandomizedStart.Max < input.RandomizedStart.Min {
			input.RandomizedStart = nil
			return input
		}
	}
	if input.RandomizedStart.Min <= 0 || input.RandomizedStart.Max <= 0 || input.RandomizedStart.Max < input.RandomizedStart.Min {
		input.RandomizedStart = nil
		if input.FixedStartingValue <= 0 {
			input.FixedStartingValue = 1
		}
	}
	return input
}

// CountdownAdvance advances one Daggerheart scene countdown.
func CountdownAdvance(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input countdownAdvanceInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	countdownID := strings.TrimSpace(input.CountdownID)
	if countdownID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("countdown_id is required")
	}
	if input.Amount <= 0 {
		return orchestration.ToolResult{}, fmt.Errorf("amount must be positive")
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}

	resp, err := runtime.DaggerheartClient().AdvanceSceneCountdown(callCtx, &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CountdownId: countdownID,
		Amount:      int32(input.Amount),
		Reason:      strings.TrimSpace(input.Reason),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("advance countdown failed: %w", err)
	}
	if resp == nil || resp.GetCountdown() == nil || resp.GetAdvance() == nil {
		return orchestration.ToolResult{}, fmt.Errorf("advance countdown response is missing")
	}
	return toolResultJSON(countdownAdvanceResult{
		Countdown: countdownSummaryFromSceneProto(resp.GetCountdown()),
		Advance: countdownAdvanceSummary{
			BeforeRemaining: int(resp.GetAdvance().GetRemainingBefore()),
			AfterRemaining:  int(resp.GetAdvance().GetRemainingAfter()),
			AdvancedBy:      int(resp.GetAdvance().GetAdvancedBy()),
			StatusBefore:    countdownStatusToString(resp.GetAdvance().GetStatusBefore()),
			StatusAfter:     countdownStatusToString(resp.GetAdvance().GetStatusAfter()),
			Triggered:       resp.GetAdvance().GetTriggered(),
			Reason:          strings.TrimSpace(resp.GetAdvance().GetReason()),
		},
	})
}

// CountdownResolveTrigger resolves a pending Daggerheart countdown trigger.
func CountdownResolveTrigger(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input countdownResolveTriggerInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := runtime.ResolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	countdownID := strings.TrimSpace(input.CountdownID)
	if countdownID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("countdown_id is required")
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}

	resp, err := runtime.DaggerheartClient().ResolveSceneCountdownTrigger(callCtx, &pb.DaggerheartResolveSceneCountdownTriggerRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CountdownId: countdownID,
		Reason:      strings.TrimSpace(input.Reason),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("resolve countdown trigger failed: %w", err)
	}
	if resp == nil || resp.GetCountdown() == nil {
		return orchestration.ToolResult{}, fmt.Errorf("resolve countdown trigger response is missing")
	}
	return toolResultJSON(countdownSummaryFromSceneProto(resp.GetCountdown()))
}

// AdversaryUpdate updates one existing Daggerheart adversary on the current scene.
func AdversaryUpdate(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input adversaryUpdateInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	campaignID := runtime.ResolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	adversaryID := strings.TrimSpace(input.AdversaryID)
	if adversaryID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("adversary_id is required")
	}
	if input.Notes == nil {
		return orchestration.ToolResult{}, fmt.Errorf("notes is required")
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = runtime.ResolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}

	resp, err := runtime.DaggerheartClient().UpdateAdversary(callCtx, &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  campaignID,
		AdversaryId: adversaryID,
		SceneId:     sceneID,
		Notes:       wrapperspb.String(strings.TrimSpace(*input.Notes)),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("update adversary failed: %w", err)
	}
	if resp == nil || resp.GetAdversary() == nil {
		return orchestration.ToolResult{}, fmt.Errorf("update adversary response is missing")
	}
	return toolResultJSON(adversarySummaryFromProto(resp.GetAdversary()))
}

func actionRollModifiersToProto(values []actionRollModifier) []*pb.ActionRollModifier {
	result := make([]*pb.ActionRollModifier, 0, len(values))
	for _, value := range values {
		if value.Value == 0 && strings.TrimSpace(value.Source) == "" {
			continue
		}
		result = append(result, &pb.ActionRollModifier{
			Source: strings.TrimSpace(value.Source),
			Value:  int32(value.Value),
		})
	}
	return result
}
