package domain

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// RngRequest represents optional RNG configuration for deterministic rolls.
type RngRequest struct {
	Seed     *uint64 `json:"seed,omitempty" jsonschema:"optional seed for deterministic rolls"`
	RollMode string  `json:"roll_mode,omitempty" jsonschema:"roll mode (LIVE or REPLAY)"`
}

// RngResult represents RNG details used for a roll.
type RngResult struct {
	SeedUsed   uint64 `json:"seed_used" jsonschema:"seed value used by the server"`
	RngAlgo    string `json:"rng_algo" jsonschema:"rng algorithm identifier"`
	SeedSource string `json:"seed_source" jsonschema:"seed source (CLIENT or SERVER)"`
	RollMode   string `json:"roll_mode" jsonschema:"roll mode applied"`
}

// rollModeToProto maps a roll mode label to the protobuf enum.
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

// rollModeLabel maps a protobuf roll mode to a label for MCP output.
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

// ActionRollResult represents the MCP tool output for an action roll.
type ActionRollResult struct {
	Hope            int        `json:"hope" jsonschema:"hope die result"`
	Fear            int        `json:"fear" jsonschema:"fear die result"`
	Modifier        int        `json:"modifier" jsonschema:"modifier applied to the total"`
	Difficulty      *int       `json:"difficulty,omitempty" jsonschema:"difficulty target, if provided"`
	Total           int        `json:"total" jsonschema:"sum of dice and modifier"`
	IsCrit          bool       `json:"is_crit" jsonschema:"whether the roll is a critical success"`
	MeetsDifficulty bool       `json:"meets_difficulty" jsonschema:"whether total meets difficulty"`
	Outcome         string     `json:"outcome" jsonschema:"categorized roll outcome"`
	Rng             *RngResult `json:"rng,omitempty" jsonschema:"rng details"`
}

// ActionRollInput represents the MCP tool input for an action roll.
type ActionRollInput struct {
	Modifier   int         `json:"modifier" jsonschema:"modifier applied to the roll"`
	Difficulty *int        `json:"difficulty" jsonschema:"optional difficulty target"`
	Rng        *RngRequest `json:"rng,omitempty" jsonschema:"optional rng configuration"`
}

// DualityOutcomeInput represents the MCP tool input for deterministic outcomes.
type DualityOutcomeInput struct {
	Hope       int  `json:"hope" jsonschema:"hope die result"`
	Fear       int  `json:"fear" jsonschema:"fear die result"`
	Modifier   int  `json:"modifier" jsonschema:"modifier applied to the roll"`
	Difficulty *int `json:"difficulty" jsonschema:"optional difficulty target"`
}

// DualityOutcomeResult represents the MCP tool output for deterministic outcomes.
type DualityOutcomeResult = ActionRollResult

// DualityExplainInput represents the MCP tool input for explanations.
type DualityExplainInput struct {
	Hope       int     `json:"hope" jsonschema:"hope die result"`
	Fear       int     `json:"fear" jsonschema:"fear die result"`
	Modifier   int     `json:"modifier" jsonschema:"modifier applied to the roll"`
	Difficulty *int    `json:"difficulty" jsonschema:"optional difficulty target"`
	RequestID  *string `json:"request_id,omitempty" jsonschema:"optional correlation identifier"`
}

// DualityExplainIntermediates represents derived evaluation values.
type DualityExplainIntermediates struct {
	BaseTotal       int  `json:"base_total" jsonschema:"sum of hope and fear"`
	Total           int  `json:"total" jsonschema:"sum of base total and modifier"`
	IsCrit          bool `json:"is_crit" jsonschema:"whether the roll is a critical success"`
	MeetsDifficulty bool `json:"meets_difficulty" jsonschema:"whether total meets difficulty"`
	HopeGtFear      bool `json:"hope_gt_fear" jsonschema:"whether hope exceeds fear"`
	FearGtHope      bool `json:"fear_gt_hope" jsonschema:"whether fear exceeds hope"`
}

// DualityExplainStep represents a deterministic evaluation step.
type DualityExplainStep struct {
	Code    string         `json:"code" jsonschema:"stable step identifier"`
	Message string         `json:"message" jsonschema:"human-readable step description"`
	Data    map[string]any `json:"data" jsonschema:"structured step payload"`
}

// DualityExplainResult represents the MCP tool output for explanations.
type DualityExplainResult struct {
	Hope            int                         `json:"hope" jsonschema:"hope die result"`
	Fear            int                         `json:"fear" jsonschema:"fear die result"`
	Modifier        int                         `json:"modifier" jsonschema:"modifier applied to the roll"`
	Difficulty      *int                        `json:"difficulty,omitempty" jsonschema:"difficulty target, if provided"`
	Total           int                         `json:"total" jsonschema:"sum of dice and modifier"`
	IsCrit          bool                        `json:"is_crit" jsonschema:"whether the roll is a critical success"`
	MeetsDifficulty bool                        `json:"meets_difficulty" jsonschema:"whether total meets difficulty"`
	Outcome         string                      `json:"outcome" jsonschema:"categorized roll outcome"`
	RulesVersion    string                      `json:"rules_version" jsonschema:"semantic ruleset version"`
	Intermediates   DualityExplainIntermediates `json:"intermediates" jsonschema:"derived evaluation values"`
	Steps           []DualityExplainStep        `json:"steps" jsonschema:"ordered evaluation steps"`
}

// DualityProbabilityInput represents the MCP tool input for probabilities.
type DualityProbabilityInput struct {
	Modifier   int `json:"modifier" jsonschema:"modifier applied to the roll"`
	Difficulty int `json:"difficulty" jsonschema:"difficulty target"`
}

// ProbabilityOutcomeCount represents a counted outcome for probabilities.
type ProbabilityOutcomeCount struct {
	Outcome string `json:"outcome" jsonschema:"outcome enum name"`
	Count   int    `json:"count" jsonschema:"number of outcomes"`
}

// DualityProbabilityResult represents the MCP tool output for probabilities.
type DualityProbabilityResult struct {
	TotalOutcomes int                       `json:"total_outcomes" jsonschema:"total number of outcomes"`
	CritCount     int                       `json:"crit_count" jsonschema:"number of critical outcomes"`
	SuccessCount  int                       `json:"success_count" jsonschema:"number of success outcomes"`
	FailureCount  int                       `json:"failure_count" jsonschema:"number of failure outcomes"`
	OutcomeCounts []ProbabilityOutcomeCount `json:"outcome_counts" jsonschema:"counts per outcome"`
}

// RulesVersionInput represents the MCP tool input for ruleset metadata.
type RulesVersionInput struct{}

// RulesVersionResult represents the MCP tool output for ruleset metadata.
type RulesVersionResult struct {
	System         string   `json:"system" jsonschema:"game system name"`
	Module         string   `json:"module" jsonschema:"ruleset module name"`
	RulesVersion   string   `json:"rules_version" jsonschema:"semantic ruleset version"`
	DiceModel      string   `json:"dice_model" jsonschema:"dice model description"`
	TotalFormula   string   `json:"total_formula" jsonschema:"total calculation expression"`
	CritRule       string   `json:"crit_rule" jsonschema:"critical success rule"`
	DifficultyRule string   `json:"difficulty_rule" jsonschema:"difficulty handling rule"`
	Outcomes       []string `json:"outcomes" jsonschema:"supported outcome enums"`
}

// RollDiceSpec represents an MCP die specification for a roll.
type RollDiceSpec struct {
	Sides int `json:"sides" jsonschema:"number of sides for the die"`
	Count int `json:"count" jsonschema:"number of dice to roll"`
}

// RollDiceInput represents the MCP tool input for rolling dice.
type RollDiceInput struct {
	Dice []RollDiceSpec `json:"dice" jsonschema:"dice specifications to roll"`
	Rng  *RngRequest    `json:"rng,omitempty" jsonschema:"optional rng configuration"`
}

// RollDiceRoll represents the results for a single dice spec.
type RollDiceRoll struct {
	Sides   int   `json:"sides" jsonschema:"number of sides for the die"`
	Results []int `json:"results" jsonschema:"individual roll results"`
	Total   int   `json:"total" jsonschema:"sum of the roll results"`
}

// RollDiceResult represents the MCP tool output for rolling dice.
type RollDiceResult struct {
	Rolls []RollDiceRoll `json:"rolls" jsonschema:"results for each dice spec"`
	Total int            `json:"total" jsonschema:"sum of all roll totals"`
	Rng   *RngResult     `json:"rng,omitempty" jsonschema:"rng details"`
}

// ActionRollTool defines the MCP tool schema for action rolls.
func ActionRollTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_action_roll",
		Description: "Rolls Duality dice for an action",
	}
}

// RollDiceTool defines the MCP tool schema for rolling dice.
func RollDiceTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "roll_dice",
		Description: "Rolls arbitrary dice pools",
	}
}

// DualityOutcomeTool defines the MCP tool schema for deterministic outcomes.
func DualityOutcomeTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_outcome",
		Description: "Evaluates a duality outcome from known dice",
	}
}

// DualityExplainTool defines the MCP tool schema for explanations.
func DualityExplainTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_explain",
		Description: "Explains a duality outcome from known dice",
	}
}

// DualityProbabilityTool defines the MCP tool schema for probabilities.
func DualityProbabilityTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_probability",
		Description: "Computes outcome probabilities across duality dice",
	}
}

// RulesVersionTool defines the MCP tool schema for ruleset metadata.
func RulesVersionTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_rules_version",
		Description: "Describes the Duality ruleset semantics",
	}
}

// ActionRollHandler executes a duality action roll.
func ActionRollHandler(client pb.DaggerheartServiceClient) mcp.ToolHandlerFor[ActionRollInput, ActionRollResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ActionRollInput) (*mcp.CallToolResult, ActionRollResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, ActionRollResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		modifier := input.Modifier

		var difficulty *int32
		if input.Difficulty != nil {
			value := int32(*input.Difficulty)
			difficulty = &value
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, ActionRollResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD
		var rngRequest *commonv1.RngRequest
		if input.Rng != nil {
			rngRequest = &commonv1.RngRequest{RollMode: rollModeToProto(input.Rng.RollMode)}
			if input.Rng.Seed != nil {
				rngRequest.Seed = input.Rng.Seed
			}
		}

		response, err := client.ActionRoll(callCtx, &pb.ActionRollRequest{
			Modifier:   int32(modifier),
			Difficulty: difficulty,
			Rng:        rngRequest,
		}, grpc.Header(&header))
		if err != nil {
			return nil, ActionRollResult{}, fmt.Errorf("action roll failed: %w", err)
		}

		var rngResult *RngResult
		if response.GetRng() != nil {
			rng := response.GetRng()
			rngResult = &RngResult{
				SeedUsed:   rng.GetSeedUsed(),
				RngAlgo:    rng.GetRngAlgo(),
				SeedSource: rng.GetSeedSource(),
				RollMode:   rollModeLabel(rng.GetRollMode()),
			}
		}

		result := ActionRollResult{
			Hope:            int(response.GetHope()),
			Fear:            int(response.GetFear()),
			Modifier:        int(response.GetModifier()),
			Total:           int(response.GetTotal()),
			IsCrit:          response.GetIsCrit(),
			MeetsDifficulty: response.GetMeetsDifficulty(),
			Outcome:         response.GetOutcome().String(),
			Rng:             rngResult,
		}
		if response.Difficulty != nil {
			value := int(response.GetDifficulty())
			result.Difficulty = &value
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// DualityOutcomeHandler executes a deterministic outcome evaluation.
func DualityOutcomeHandler(client pb.DaggerheartServiceClient) mcp.ToolHandlerFor[DualityOutcomeInput, DualityOutcomeResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DualityOutcomeInput) (*mcp.CallToolResult, DualityOutcomeResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, DualityOutcomeResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		var difficulty *int32
		if input.Difficulty != nil {
			value := int32(*input.Difficulty)
			difficulty = &value
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, DualityOutcomeResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		response, err := client.DualityOutcome(callCtx, &pb.DualityOutcomeRequest{
			Hope:       int32(input.Hope),
			Fear:       int32(input.Fear),
			Modifier:   int32(input.Modifier),
			Difficulty: difficulty,
		}, grpc.Header(&header))
		if err != nil {
			return nil, DualityOutcomeResult{}, fmt.Errorf("duality outcome failed: %w", err)
		}

		result := DualityOutcomeResult{
			Hope:            int(response.GetHope()),
			Fear:            int(response.GetFear()),
			Modifier:        int(response.GetModifier()),
			Total:           int(response.GetTotal()),
			IsCrit:          response.GetIsCrit(),
			MeetsDifficulty: response.GetMeetsDifficulty(),
			Outcome:         response.GetOutcome().String(),
		}
		if response.Difficulty != nil {
			value := int(response.GetDifficulty())
			result.Difficulty = &value
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// DualityExplainHandler executes a deterministic explanation request.
func DualityExplainHandler(client pb.DaggerheartServiceClient) mcp.ToolHandlerFor[DualityExplainInput, DualityExplainResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DualityExplainInput) (*mcp.CallToolResult, DualityExplainResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, DualityExplainResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		modifier := input.Modifier

		var difficulty *int32
		if input.Difficulty != nil {
			value := int32(*input.Difficulty)
			difficulty = &value
		}

		var requestID *string
		if input.RequestID != nil {
			requestID = input.RequestID
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, DualityExplainResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		response, err := client.DualityExplain(callCtx, &pb.DualityExplainRequest{
			Hope:       int32(input.Hope),
			Fear:       int32(input.Fear),
			Modifier:   int32(modifier),
			Difficulty: difficulty,
			RequestId:  requestID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, DualityExplainResult{}, fmt.Errorf("duality explain failed: %w", err)
		}
		if response == nil {
			return nil, DualityExplainResult{}, fmt.Errorf("duality explain response is missing")
		}
		if response.GetIntermediates() == nil {
			return nil, DualityExplainResult{}, fmt.Errorf("duality explain intermediates are missing")
		}

		result := DualityExplainResult{
			Hope:            int(response.GetHope()),
			Fear:            int(response.GetFear()),
			Modifier:        int(response.GetModifier()),
			Total:           int(response.GetTotal()),
			IsCrit:          response.GetIsCrit(),
			MeetsDifficulty: response.GetMeetsDifficulty(),
			Outcome:         response.GetOutcome().String(),
			RulesVersion:    response.GetRulesVersion(),
			Intermediates: DualityExplainIntermediates{
				BaseTotal:       int(response.GetIntermediates().GetBaseTotal()),
				Total:           int(response.GetIntermediates().GetTotal()),
				IsCrit:          response.GetIntermediates().GetIsCrit(),
				MeetsDifficulty: response.GetIntermediates().GetMeetsDifficulty(),
				HopeGtFear:      response.GetIntermediates().GetHopeGtFear(),
				FearGtHope:      response.GetIntermediates().GetFearGtHope(),
			},
			Steps: make([]DualityExplainStep, 0, len(response.GetSteps())),
		}
		if response.Difficulty != nil {
			value := int(response.GetDifficulty())
			result.Difficulty = &value
		}

		for _, step := range response.GetSteps() {
			data := map[string]any{}
			if step.GetData() != nil {
				data = step.GetData().AsMap()
			}
			result.Steps = append(result.Steps, DualityExplainStep{
				Code:    step.GetCode(),
				Message: step.GetMessage(),
				Data:    data,
			})
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// DualityProbabilityHandler executes the deterministic probability evaluation.
func DualityProbabilityHandler(client pb.DaggerheartServiceClient) mcp.ToolHandlerFor[DualityProbabilityInput, DualityProbabilityResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DualityProbabilityInput) (*mcp.CallToolResult, DualityProbabilityResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, DualityProbabilityResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, DualityProbabilityResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		response, err := client.DualityProbability(callCtx, &pb.DualityProbabilityRequest{
			Modifier:   int32(input.Modifier),
			Difficulty: int32(input.Difficulty),
		}, grpc.Header(&header))
		if err != nil {
			return nil, DualityProbabilityResult{}, fmt.Errorf("duality probability failed: %w", err)
		}
		if response == nil {
			return nil, DualityProbabilityResult{}, fmt.Errorf("duality probability response is missing")
		}

		counts := make([]ProbabilityOutcomeCount, 0, len(response.GetOutcomeCounts()))
		for _, count := range response.GetOutcomeCounts() {
			counts = append(counts, ProbabilityOutcomeCount{
				Outcome: count.GetOutcome().String(),
				Count:   int(count.GetCount()),
			})
		}

		result := DualityProbabilityResult{
			TotalOutcomes: int(response.GetTotalOutcomes()),
			CritCount:     int(response.GetCritCount()),
			SuccessCount:  int(response.GetSuccessCount()),
			FailureCount:  int(response.GetFailureCount()),
			OutcomeCounts: counts,
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// RulesVersionHandler returns static ruleset metadata from the gRPC service.
func RulesVersionHandler(client pb.DaggerheartServiceClient) mcp.ToolHandlerFor[RulesVersionInput, RulesVersionResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ RulesVersionInput) (*mcp.CallToolResult, RulesVersionResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, RulesVersionResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, RulesVersionResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		response, err := client.RulesVersion(callCtx, &pb.RulesVersionRequest{}, grpc.Header(&header))
		if err != nil {
			return nil, RulesVersionResult{}, fmt.Errorf("rules version failed: %w", err)
		}
		if response == nil {
			return nil, RulesVersionResult{}, fmt.Errorf("rules version response is missing")
		}

		outcomes := make([]string, 0, len(response.GetOutcomes()))
		for _, outcome := range response.GetOutcomes() {
			outcomes = append(outcomes, outcome.String())
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), RulesVersionResult{
			System:         response.GetSystem(),
			Module:         response.GetModule(),
			RulesVersion:   response.GetRulesVersion(),
			DiceModel:      response.GetDiceModel(),
			TotalFormula:   response.GetTotalFormula(),
			CritRule:       response.GetCritRule(),
			DifficultyRule: response.GetDifficultyRule(),
			Outcomes:       outcomes,
		}, nil
	}
}

// RollDiceHandler executes a generic dice roll.
func RollDiceHandler(client pb.DaggerheartServiceClient) mcp.ToolHandlerFor[RollDiceInput, RollDiceResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input RollDiceInput) (*mcp.CallToolResult, RollDiceResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, RollDiceResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		diceSpecs := make([]*pb.DiceSpec, 0, len(input.Dice))
		for _, spec := range input.Dice {
			diceSpecs = append(diceSpecs, &pb.DiceSpec{
				Sides: int32(spec.Sides),
				Count: int32(spec.Count),
			})
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, RollDiceResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD
		var rngRequest *commonv1.RngRequest
		if input.Rng != nil {
			rngRequest = &commonv1.RngRequest{RollMode: rollModeToProto(input.Rng.RollMode)}
			if input.Rng.Seed != nil {
				rngRequest.Seed = input.Rng.Seed
			}
		}

		response, err := client.RollDice(callCtx, &pb.RollDiceRequest{
			Dice: diceSpecs,
			Rng:  rngRequest,
		}, grpc.Header(&header))
		if err != nil {
			return nil, RollDiceResult{}, fmt.Errorf("dice roll failed: %w", err)
		}
		if response == nil {
			return nil, RollDiceResult{}, fmt.Errorf("dice roll response is missing")
		}

		rolls := make([]RollDiceRoll, 0, len(response.GetRolls()))
		for _, roll := range response.GetRolls() {
			rolls = append(rolls, RollDiceRoll{
				Sides:   int(roll.GetSides()),
				Results: intSlice(roll.GetResults()),
				Total:   int(roll.GetTotal()),
			})
		}

		var rngResult *RngResult
		if response.GetRng() != nil {
			rng := response.GetRng()
			rngResult = &RngResult{
				SeedUsed:   rng.GetSeedUsed(),
				RngAlgo:    rng.GetRngAlgo(),
				SeedSource: rng.GetSeedSource(),
				RollMode:   rollModeLabel(rng.GetRollMode()),
			}
		}

		result := RollDiceResult{
			Rolls: rolls,
			Total: int(response.GetTotal()),
			Rng:   rngResult,
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// intSlice converts a slice of int32 to a slice of int.
func intSlice(values []int32) []int {
	converted := make([]int, len(values))
	for i, value := range values {
		converted[i] = int(value)
	}
	return converted
}
