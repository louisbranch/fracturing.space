// Package mcp provides the MCP server for duality rolls.
package mcp

import (
	"context"
	"fmt"
	"os"
	"time"

	pb "github.com/louisbranch/duality-protocol/api/gen/go/duality/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// serverName identifies this MCP server to clients.
	serverName = "Duality Engine MCP"
	// serverVersion identifies the MCP server version.
	serverVersion = "0.1.0"
)

// Server hosts the MCP server.
type Server struct {
	mcpServer *mcp.Server
}

// ActionRollResult represents the MCP tool output for an action roll.
type ActionRollResult struct {
	Hope            int    `json:"hope" jsonschema:"hope die result"`
	Fear            int    `json:"fear" jsonschema:"fear die result"`
	Modifier        int    `json:"modifier" jsonschema:"modifier applied to the total"`
	Difficulty      *int   `json:"difficulty,omitempty" jsonschema:"difficulty target, if provided"`
	Total           int    `json:"total" jsonschema:"sum of dice and modifier"`
	IsCrit          bool   `json:"is_crit" jsonschema:"whether the roll is a critical success"`
	MeetsDifficulty bool   `json:"meets_difficulty" jsonschema:"whether total meets difficulty"`
	Outcome         string `json:"outcome" jsonschema:"categorized roll outcome"`
}

// ActionRollInput represents the MCP tool input for an action roll.
type ActionRollInput struct {
	Modifier   int  `json:"modifier" jsonschema:"modifier applied to the roll"`
	Difficulty *int `json:"difficulty" jsonschema:"optional difficulty target"`
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
	Outcome int `json:"outcome" jsonschema:"numeric outcome enum value"`
	Count   int `json:"count" jsonschema:"number of outcomes"`
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
}

// New creates a configured MCP server that connects to the gRPC dice service.
func New(addr string) (*Server, error) {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)

	grpcAddr := grpcAddress(addr)
	grpcClient, err := newDualityClient(grpcAddr)
	if err != nil {
		return nil, fmt.Errorf("connect to gRPC server at %s: %w", grpcAddr, err)
	}

	mcp.AddTool(mcpServer, actionRollTool(), actionRollHandler(grpcClient))
	mcp.AddTool(mcpServer, dualityOutcomeTool(), dualityOutcomeHandler(grpcClient))
	mcp.AddTool(mcpServer, dualityExplainTool(), dualityExplainHandler(grpcClient))
	mcp.AddTool(mcpServer, dualityProbabilityTool(), dualityProbabilityHandler(grpcClient))
	mcp.AddTool(mcpServer, rulesVersionTool(), rulesVersionHandler(grpcClient))
	mcp.AddTool(mcpServer, rollDiceTool(), rollDiceHandler(grpcClient))

	return &Server{mcpServer: mcpServer}, nil
}

// Serve starts the MCP server on stdio.
func (s *Server) Serve() error {
	if s == nil || s.mcpServer == nil {
		return fmt.Errorf("MCP server is not configured")
	}
	if err := s.mcpServer.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("serve MCP: %w", err)
	}
	return nil
}

// actionRollTool defines the MCP tool schema for action rolls.
func actionRollTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_action_roll",
		Description: "Rolls Duality dice for an action",
	}
}

// rollDiceTool defines the MCP tool schema for rolling dice.
func rollDiceTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "roll_dice",
		Description: "Rolls arbitrary dice pools",
	}
}

// dualityOutcomeTool defines the MCP tool schema for deterministic outcomes.
func dualityOutcomeTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_outcome",
		Description: "Evaluates a duality outcome from known dice",
	}
}

// dualityExplainTool defines the MCP tool schema for explanations.
func dualityExplainTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_explain",
		Description: "Explains a duality outcome from known dice",
	}
}

// dualityProbabilityTool defines the MCP tool schema for probabilities.
func dualityProbabilityTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_probability",
		Description: "Computes outcome probabilities across duality dice",
	}
}

// rulesVersionTool defines the MCP tool schema for ruleset metadata.
func rulesVersionTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_rules_version",
		Description: "Describes the Duality ruleset semantics",
	}
}

// actionRollHandler executes a duality action roll.
func actionRollHandler(client pb.DualityServiceClient) mcp.ToolHandlerFor[ActionRollInput, ActionRollResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ActionRollInput) (*mcp.CallToolResult, ActionRollResult, error) {
		modifier := input.Modifier

		var difficulty *int32
		if input.Difficulty != nil {
			value := int32(*input.Difficulty)
			difficulty = &value
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		response, err := client.ActionRoll(runCtx, &pb.ActionRollRequest{
			Modifier:   int32(modifier),
			Difficulty: difficulty,
		})
		if err != nil {
			return nil, ActionRollResult{}, fmt.Errorf("action roll failed: %w", err)
		}

		result := ActionRollResult{
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

		return nil, result, nil
	}
}

// dualityOutcomeHandler executes a deterministic outcome evaluation.
func dualityOutcomeHandler(client pb.DualityServiceClient) mcp.ToolHandlerFor[DualityOutcomeInput, DualityOutcomeResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DualityOutcomeInput) (*mcp.CallToolResult, DualityOutcomeResult, error) {
		var difficulty *int32
		if input.Difficulty != nil {
			value := int32(*input.Difficulty)
			difficulty = &value
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		response, err := client.DualityOutcome(runCtx, &pb.DualityOutcomeRequest{
			Hope:       int32(input.Hope),
			Fear:       int32(input.Fear),
			Modifier:   int32(input.Modifier),
			Difficulty: difficulty,
		})
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

		return nil, result, nil
	}
}

// dualityExplainHandler executes a deterministic explanation request.
func dualityExplainHandler(client pb.DualityServiceClient) mcp.ToolHandlerFor[DualityExplainInput, DualityExplainResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DualityExplainInput) (*mcp.CallToolResult, DualityExplainResult, error) {
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

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		response, err := client.DualityExplain(runCtx, &pb.DualityExplainRequest{
			Hope:       int32(input.Hope),
			Fear:       int32(input.Fear),
			Modifier:   int32(modifier),
			Difficulty: difficulty,
			RequestId:  requestID,
		})
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

		return nil, result, nil
	}
}

// dualityProbabilityHandler executes the deterministic probability evaluation.
func dualityProbabilityHandler(client pb.DualityServiceClient) mcp.ToolHandlerFor[DualityProbabilityInput, DualityProbabilityResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DualityProbabilityInput) (*mcp.CallToolResult, DualityProbabilityResult, error) {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		response, err := client.DualityProbability(runCtx, &pb.DualityProbabilityRequest{
			Modifier:   int32(input.Modifier),
			Difficulty: int32(input.Difficulty),
		})
		if err != nil {
			return nil, DualityProbabilityResult{}, fmt.Errorf("duality probability failed: %w", err)
		}
		if response == nil {
			return nil, DualityProbabilityResult{}, fmt.Errorf("duality probability response is missing")
		}

		counts := make([]ProbabilityOutcomeCount, 0, len(response.GetOutcomeCounts()))
		for _, count := range response.GetOutcomeCounts() {
			counts = append(counts, ProbabilityOutcomeCount{
				Outcome: int(count.GetOutcome()),
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

		return nil, result, nil
	}
}

// rulesVersionHandler returns static ruleset metadata from the gRPC service.
func rulesVersionHandler(client pb.DualityServiceClient) mcp.ToolHandlerFor[RulesVersionInput, RulesVersionResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ RulesVersionInput) (*mcp.CallToolResult, RulesVersionResult, error) {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		response, err := client.RulesVersion(runCtx, &pb.RulesVersionRequest{})
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

		return nil, RulesVersionResult{
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

// rollDiceHandler executes a generic dice roll.
func rollDiceHandler(client pb.DualityServiceClient) mcp.ToolHandlerFor[RollDiceInput, RollDiceResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input RollDiceInput) (*mcp.CallToolResult, RollDiceResult, error) {
		diceSpecs := make([]*pb.DiceSpec, 0, len(input.Dice))
		for _, spec := range input.Dice {
			diceSpecs = append(diceSpecs, &pb.DiceSpec{
				Sides: int32(spec.Sides),
				Count: int32(spec.Count),
			})
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		response, err := client.RollDice(runCtx, &pb.RollDiceRequest{
			Dice: diceSpecs,
		})
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

		result := RollDiceResult{
			Rolls: rolls,
			Total: int(response.GetTotal()),
		}

		return nil, result, nil
	}
}

// newDualityClient connects to the gRPC Duality service.
func newDualityClient(addr string) (pb.DualityServiceClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return pb.NewDualityServiceClient(conn), nil
}

// grpcAddress resolves the gRPC address from env or defaults.
func grpcAddress(fallback string) string {
	if value := os.Getenv("DUALITY_GRPC_ADDR"); value != "" {
		return value
	}
	return fallback
}

// intSlice converts a slice of int32 to a slice of int.
func intSlice(values []int32) []int {
	converted := make([]int, len(values))
	for i, value := range values {
		converted[i] = int(value)
	}
	return converted
}
