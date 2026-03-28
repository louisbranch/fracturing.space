package daggerhearttools

import (
	"context"
	"encoding/json"
	"fmt"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

type actionRollInput struct {
	Modifier   int         `json:"modifier"`
	Difficulty *int        `json:"difficulty"`
	Rng        *rngRequest `json:"rng,omitempty"`
}

type dualityOutcomeInput struct {
	Hope       int  `json:"hope"`
	Fear       int  `json:"fear"`
	Modifier   int  `json:"modifier"`
	Difficulty *int `json:"difficulty"`
}

type dualityExplainInput struct {
	Hope       int     `json:"hope"`
	Fear       int     `json:"fear"`
	Modifier   int     `json:"modifier"`
	Difficulty *int    `json:"difficulty"`
	RequestID  *string `json:"request_id,omitempty"`
}

type dualityProbabilityInput struct {
	Modifier   int `json:"modifier"`
	Difficulty int `json:"difficulty"`
}

type rollDiceSpec struct {
	Sides int `json:"sides"`
	Count int `json:"count"`
}

type rollDiceInput struct {
	Dice []rollDiceSpec `json:"dice"`
	Rng  *rngRequest    `json:"rng,omitempty"`
}

type actionRollResult struct {
	Hope            int        `json:"hope"`
	Fear            int        `json:"fear"`
	Modifier        int        `json:"modifier"`
	Difficulty      *int       `json:"difficulty,omitempty"`
	Total           int        `json:"total"`
	IsCrit          bool       `json:"is_crit"`
	MeetsDifficulty bool       `json:"meets_difficulty"`
	Outcome         string     `json:"outcome"`
	Rng             *rngResult `json:"rng,omitempty"`
}

type dualityExplainIntermediates struct {
	BaseTotal       int  `json:"base_total"`
	Total           int  `json:"total"`
	IsCrit          bool `json:"is_crit"`
	MeetsDifficulty bool `json:"meets_difficulty"`
	HopeGtFear      bool `json:"hope_gt_fear"`
	FearGtHope      bool `json:"fear_gt_hope"`
}

type dualityExplainStep struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type dualityExplainResult struct {
	Hope            int                         `json:"hope"`
	Fear            int                         `json:"fear"`
	Modifier        int                         `json:"modifier"`
	Difficulty      *int                        `json:"difficulty,omitempty"`
	Total           int                         `json:"total"`
	IsCrit          bool                        `json:"is_crit"`
	MeetsDifficulty bool                        `json:"meets_difficulty"`
	Outcome         string                      `json:"outcome"`
	RulesVersion    string                      `json:"rules_version"`
	Intermediates   dualityExplainIntermediates `json:"intermediates"`
	Steps           []dualityExplainStep        `json:"steps"`
}

type probabilityOutcomeCount struct {
	Outcome string `json:"outcome"`
	Count   int    `json:"count"`
}

type dualityProbabilityResult struct {
	TotalOutcomes int                       `json:"total_outcomes"`
	CritCount     int                       `json:"crit_count"`
	SuccessCount  int                       `json:"success_count"`
	FailureCount  int                       `json:"failure_count"`
	OutcomeCounts []probabilityOutcomeCount `json:"outcome_counts"`
}

type rulesVersionResult struct {
	System         string   `json:"system"`
	Module         string   `json:"module"`
	RulesVersion   string   `json:"rules_version"`
	DiceModel      string   `json:"dice_model"`
	TotalFormula   string   `json:"total_formula"`
	CritRule       string   `json:"crit_rule"`
	DifficultyRule string   `json:"difficulty_rule"`
	Outcomes       []string `json:"outcomes"`
}

type rollDiceRoll struct {
	Sides   int   `json:"sides"`
	Results []int `json:"results"`
	Total   int   `json:"total"`
}

type rollDiceResult struct {
	Rolls []rollDiceRoll `json:"rolls"`
	Total int            `json:"total"`
	Rng   *rngResult     `json:"rng,omitempty"`
}

// DualityActionRoll runs the authoritative Daggerheart duality action-roll RPC.
func DualityActionRoll(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input actionRollInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	var difficulty *int32
	if input.Difficulty != nil {
		value := int32(*input.Difficulty)
		difficulty = &value
	}
	var rng *commonv1.RngRequest
	if input.Rng != nil {
		rng = &commonv1.RngRequest{RollMode: rollModeToProto(input.Rng.RollMode)}
		if input.Rng.Seed != nil {
			rng.Seed = input.Rng.Seed
		}
	}

	resp, err := runtime.DaggerheartClient().ActionRoll(callCtx, &pb.ActionRollRequest{
		Modifier:   int32(input.Modifier),
		Difficulty: difficulty,
		Rng:        rng,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("action roll failed: %w", err)
	}

	result := actionRollResult{
		Hope:            int(resp.GetHope()),
		Fear:            int(resp.GetFear()),
		Modifier:        int(resp.GetModifier()),
		Total:           int(resp.GetTotal()),
		IsCrit:          resp.GetIsCrit(),
		MeetsDifficulty: resp.GetMeetsDifficulty(),
		Outcome:         resp.GetOutcome().String(),
		Rng:             rngResultFromProto(resp.GetRng()),
	}
	if resp.Difficulty != nil {
		value := int(resp.GetDifficulty())
		result.Difficulty = &value
	}
	return toolResultJSON(result)
}

// RollDice runs the authoritative arbitrary-dice RPC used by AI tooling.
func RollDice(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input rollDiceInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	diceSpecs := make([]*pb.DiceSpec, 0, len(input.Dice))
	for _, spec := range input.Dice {
		diceSpecs = append(diceSpecs, &pb.DiceSpec{
			Sides: int32(spec.Sides),
			Count: int32(spec.Count),
		})
	}

	var rng *commonv1.RngRequest
	if input.Rng != nil {
		rng = &commonv1.RngRequest{RollMode: rollModeToProto(input.Rng.RollMode)}
		if input.Rng.Seed != nil {
			rng.Seed = input.Rng.Seed
		}
	}

	resp, err := runtime.DaggerheartClient().RollDice(callCtx, &pb.RollDiceRequest{
		Dice: diceSpecs,
		Rng:  rng,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("dice roll failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("dice roll response is missing")
	}

	rolls := make([]rollDiceRoll, 0, len(resp.GetRolls()))
	for _, roll := range resp.GetRolls() {
		rolls = append(rolls, rollDiceRoll{
			Sides:   int(roll.GetSides()),
			Results: intSlice(roll.GetResults()),
			Total:   int(roll.GetTotal()),
		})
	}
	return toolResultJSON(rollDiceResult{
		Rolls: rolls,
		Total: int(resp.GetTotal()),
		Rng:   rngResultFromProto(resp.GetRng()),
	})
}

// DualityOutcome evaluates a known duality roll through the authoritative service.
func DualityOutcome(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input dualityOutcomeInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	var difficulty *int32
	if input.Difficulty != nil {
		value := int32(*input.Difficulty)
		difficulty = &value
	}

	resp, err := runtime.DaggerheartClient().DualityOutcome(callCtx, &pb.DualityOutcomeRequest{
		Hope:       int32(input.Hope),
		Fear:       int32(input.Fear),
		Modifier:   int32(input.Modifier),
		Difficulty: difficulty,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("duality outcome failed: %w", err)
	}

	result := actionRollResult{
		Hope:            int(resp.GetHope()),
		Fear:            int(resp.GetFear()),
		Modifier:        int(resp.GetModifier()),
		Total:           int(resp.GetTotal()),
		IsCrit:          resp.GetIsCrit(),
		MeetsDifficulty: resp.GetMeetsDifficulty(),
		Outcome:         resp.GetOutcome().String(),
	}
	if resp.Difficulty != nil {
		value := int(resp.GetDifficulty())
		result.Difficulty = &value
	}
	return toolResultJSON(result)
}

// DualityExplain asks the authoritative service to explain a known duality roll.
func DualityExplain(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input dualityExplainInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	var difficulty *int32
	if input.Difficulty != nil {
		value := int32(*input.Difficulty)
		difficulty = &value
	}

	resp, err := runtime.DaggerheartClient().DualityExplain(callCtx, &pb.DualityExplainRequest{
		Hope:       int32(input.Hope),
		Fear:       int32(input.Fear),
		Modifier:   int32(input.Modifier),
		Difficulty: difficulty,
		RequestId:  input.RequestID,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("duality explain failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("duality explain response is missing")
	}
	if resp.GetIntermediates() == nil {
		return orchestration.ToolResult{}, fmt.Errorf("duality explain intermediates are missing")
	}

	result := dualityExplainResult{
		Hope:            int(resp.GetHope()),
		Fear:            int(resp.GetFear()),
		Modifier:        int(resp.GetModifier()),
		Total:           int(resp.GetTotal()),
		IsCrit:          resp.GetIsCrit(),
		MeetsDifficulty: resp.GetMeetsDifficulty(),
		Outcome:         resp.GetOutcome().String(),
		RulesVersion:    resp.GetRulesVersion(),
		Intermediates: dualityExplainIntermediates{
			BaseTotal:       int(resp.GetIntermediates().GetBaseTotal()),
			Total:           int(resp.GetIntermediates().GetTotal()),
			IsCrit:          resp.GetIntermediates().GetIsCrit(),
			MeetsDifficulty: resp.GetIntermediates().GetMeetsDifficulty(),
			HopeGtFear:      resp.GetIntermediates().GetHopeGtFear(),
			FearGtHope:      resp.GetIntermediates().GetFearGtHope(),
		},
		Steps: make([]dualityExplainStep, 0, len(resp.GetSteps())),
	}
	if resp.Difficulty != nil {
		value := int(resp.GetDifficulty())
		result.Difficulty = &value
	}
	for _, step := range resp.GetSteps() {
		data := map[string]any{}
		if step.GetData() != nil {
			data = step.GetData().AsMap()
		}
		result.Steps = append(result.Steps, dualityExplainStep{
			Code:    step.GetCode(),
			Message: step.GetMessage(),
			Data:    data,
		})
	}
	return toolResultJSON(result)
}

// DualityProbability computes duality outcome probabilities through the authoritative service.
func DualityProbability(runtime Runtime, ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input dualityProbabilityInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}

	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	resp, err := runtime.DaggerheartClient().DualityProbability(callCtx, &pb.DualityProbabilityRequest{
		Modifier:   int32(input.Modifier),
		Difficulty: int32(input.Difficulty),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("duality probability failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("duality probability response is missing")
	}

	counts := make([]probabilityOutcomeCount, 0, len(resp.GetOutcomeCounts()))
	for _, count := range resp.GetOutcomeCounts() {
		counts = append(counts, probabilityOutcomeCount{
			Outcome: count.GetOutcome().String(),
			Count:   int(count.GetCount()),
		})
	}
	return toolResultJSON(dualityProbabilityResult{
		TotalOutcomes: int(resp.GetTotalOutcomes()),
		CritCount:     int(resp.GetCritCount()),
		SuccessCount:  int(resp.GetSuccessCount()),
		FailureCount:  int(resp.GetFailureCount()),
		OutcomeCounts: counts,
	})
}

// DualityRulesVersion returns the current authoritative ruleset metadata.
func DualityRulesVersion(runtime Runtime, ctx context.Context, _ []byte) (orchestration.ToolResult, error) {
	callCtx, cancel := runtime.CallContext(ctx)
	defer cancel()

	resp, err := runtime.DaggerheartClient().RulesVersion(callCtx, &pb.RulesVersionRequest{})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("rules version failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("rules version response is missing")
	}

	outcomes := make([]string, 0, len(resp.GetOutcomes()))
	for _, outcome := range resp.GetOutcomes() {
		outcomes = append(outcomes, outcome.String())
	}
	return toolResultJSON(rulesVersionResult{
		System:         resp.GetSystem(),
		Module:         resp.GetModule(),
		RulesVersion:   resp.GetRulesVersion(),
		DiceModel:      resp.GetDiceModel(),
		TotalFormula:   resp.GetTotalFormula(),
		CritRule:       resp.GetCritRule(),
		DifficultyRule: resp.GetDifficultyRule(),
		Outcomes:       outcomes,
	})
}
