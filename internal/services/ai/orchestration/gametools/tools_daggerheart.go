package gametools

import (
	"context"
	"encoding/json"
	"fmt"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

// --- Input types ---

type rngRequest struct {
	Seed     *uint64 `json:"seed,omitempty"`
	RollMode string  `json:"roll_mode,omitempty"`
}

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

// --- Result types ---

type rngResult struct {
	SeedUsed   uint64 `json:"seed_used"`
	RngAlgo    string `json:"rng_algo"`
	SeedSource string `json:"seed_source"`
	RollMode   string `json:"roll_mode"`
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

// --- Handlers ---

func (s *DirectSession) dualityActionRoll(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input actionRollInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	var difficulty *int32
	if input.Difficulty != nil {
		v := int32(*input.Difficulty)
		difficulty = &v
	}
	var rng *commonv1.RngRequest
	if input.Rng != nil {
		rng = &commonv1.RngRequest{RollMode: rollModeToProto(input.Rng.RollMode)}
		if input.Rng.Seed != nil {
			rng.Seed = input.Rng.Seed
		}
	}

	resp, err := s.clients.Daggerheart.ActionRoll(callCtx, &pb.ActionRollRequest{
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
		v := int(resp.GetDifficulty())
		result.Difficulty = &v
	}
	return toolResultJSON(result)
}

func (s *DirectSession) rollDice(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input rollDiceInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
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

	resp, err := s.clients.Daggerheart.RollDice(callCtx, &pb.RollDiceRequest{
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

func (s *DirectSession) dualityOutcome(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input dualityOutcomeInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	var difficulty *int32
	if input.Difficulty != nil {
		v := int32(*input.Difficulty)
		difficulty = &v
	}
	resp, err := s.clients.Daggerheart.DualityOutcome(callCtx, &pb.DualityOutcomeRequest{
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
		v := int(resp.GetDifficulty())
		result.Difficulty = &v
	}
	return toolResultJSON(result)
}

func (s *DirectSession) dualityExplain(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input dualityExplainInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	var difficulty *int32
	if input.Difficulty != nil {
		v := int32(*input.Difficulty)
		difficulty = &v
	}
	resp, err := s.clients.Daggerheart.DualityExplain(callCtx, &pb.DualityExplainRequest{
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
		v := int(resp.GetDifficulty())
		result.Difficulty = &v
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

func (s *DirectSession) dualityProbability(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input dualityProbabilityInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Daggerheart.DualityProbability(callCtx, &pb.DualityProbabilityRequest{
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
	for _, c := range resp.GetOutcomeCounts() {
		counts = append(counts, probabilityOutcomeCount{
			Outcome: c.GetOutcome().String(),
			Count:   int(c.GetCount()),
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

func (s *DirectSession) dualityRulesVersion(ctx context.Context, _ []byte) (orchestration.ToolResult, error) {
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Daggerheart.RulesVersion(callCtx, &pb.RulesVersionRequest{})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("rules version failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("rules version response is missing")
	}

	outcomes := make([]string, 0, len(resp.GetOutcomes()))
	for _, o := range resp.GetOutcomes() {
		outcomes = append(outcomes, o.String())
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

func rngResultFromProto(r *commonv1.RngResponse) *rngResult {
	if r == nil {
		return nil
	}
	return &rngResult{
		SeedUsed:   r.GetSeedUsed(),
		RngAlgo:    r.GetRngAlgo(),
		SeedSource: r.GetSeedSource(),
		RollMode:   rollModeLabel(r.GetRollMode()),
	}
}
