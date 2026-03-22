package gametools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
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

// daggerheartActionRollResolve runs the authoritative Daggerheart action-roll
// workflow as one AI-facing tool call so the model does not have to sequence
// roll creation and outcome application manually.
func (s *DirectSession) daggerheartActionRollResolve(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input actionRollResolveInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := s.resolveSessionID("")
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
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = s.resolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}

	rollResp, err := s.clients.Daggerheart.SessionActionRoll(callCtx, &pb.SessionActionRollRequest{
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

	outcomeResp, err := s.clients.Daggerheart.ApplyRollOutcome(callCtx, &pb.ApplyRollOutcomeRequest{
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

// daggerheartGmMoveApply spends Fear through the authoritative GM-move RPC so
// the AI can change the board state without inventing custom state mutation.
func (s *DirectSession) daggerheartGmMoveApply(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input gmMoveApplyInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := s.resolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = s.resolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}

	req, err := gmMoveApplyRequestFromInput(campaignID, sessionID, sceneID, input)
	if err != nil {
		return orchestration.ToolResult{}, err
	}

	resp, err := s.clients.Daggerheart.ApplyGmMove(callCtx, req)
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

// daggerheartAdversaryCreate creates one adversary on the current session
// scene through the authoritative Daggerheart adversary workflow.
func (s *DirectSession) daggerheartAdversaryCreate(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input adversaryCreateInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := s.resolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	adversaryEntryID := strings.TrimSpace(input.AdversaryEntryID)
	if adversaryEntryID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("adversary_entry_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = s.resolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}
	resp, err := s.clients.Daggerheart.CreateAdversary(callCtx, &pb.DaggerheartCreateAdversaryRequest{
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

// daggerheartCountdownCreate creates one Daggerheart countdown on the current
// session scene so the AI can externalize visible pressure with canonical state.
func (s *DirectSession) daggerheartCountdownCreate(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input countdownCreateInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := s.resolveSessionID("")
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
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = s.resolveSceneID(callCtx, campaignID, "")
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
		req.StartingValue = &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{FixedStartingValue: int32(input.FixedStartingValue)}
	}
	resp, err := s.clients.Daggerheart.CreateSceneCountdown(callCtx, req)
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

// daggerheartCountdownUpdate advances one Daggerheart countdown so
// the AI can keep visible pressure synchronized with the current fiction.
func (s *DirectSession) daggerheartCountdownUpdate(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input countdownAdvanceInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := s.resolveSessionID("")
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
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = s.resolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}

	req := &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CountdownId: countdownID,
		Amount:      int32(input.Amount),
		Reason:      strings.TrimSpace(input.Reason),
	}
	resp, err := s.clients.Daggerheart.AdvanceSceneCountdown(callCtx, req)
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

func (s *DirectSession) daggerheartCountdownResolveTrigger(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input countdownResolveTriggerInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID("")
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	sessionID := s.resolveSessionID("")
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}
	countdownID := strings.TrimSpace(input.CountdownID)
	if countdownID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("countdown_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = s.resolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}
	resp, err := s.clients.Daggerheart.ResolveSceneCountdownTrigger(callCtx, &pb.DaggerheartResolveSceneCountdownTriggerRequest{
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

// daggerheartAdversaryUpdate updates one existing adversary on the current
// scene board without exposing lower-level projection details to the AI.
func (s *DirectSession) daggerheartAdversaryUpdate(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input adversaryUpdateInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID("")
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
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID := strings.TrimSpace(input.SceneID)
	if sceneID == "" {
		var err error
		sceneID, err = s.resolveSceneID(callCtx, campaignID, "")
		if err != nil {
			return orchestration.ToolResult{}, err
		}
	}
	resp, err := s.clients.Daggerheart.UpdateAdversary(callCtx, &pb.DaggerheartUpdateAdversaryRequest{
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
		updated := int(value.GetGmFear())
		update.GMFear = &updated
	}
	if len(update.CharacterStates) == 0 && update.GMFear == nil {
		return nil
	}
	return update
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
		summary.PendingExperience = &experienceSummary{Name: pending.GetName(), Modifier: int(pending.GetModifier())}
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

func rangeInputToRNG(value *rangeInput) *rngRequest {
	if value == nil || value.Seed == nil {
		return nil
	}
	return &rngRequest{Seed: value.Seed}
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
