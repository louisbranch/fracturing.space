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
	grpcClient, err := newDiceRollClient(grpcAddr)
	if err != nil {
		return nil, fmt.Errorf("connect to gRPC server at %s: %w", grpcAddr, err)
	}

	mcp.AddTool(mcpServer, actionRollTool(), actionRollHandler(grpcClient))
	mcp.AddTool(mcpServer, dualityOutcomeTool(), dualityOutcomeHandler(grpcClient))
	mcp.AddTool(mcpServer, dualityProbabilityTool(), dualityProbabilityHandler(grpcClient))
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

// dualityProbabilityTool defines the MCP tool schema for probabilities.
func dualityProbabilityTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "duality_probability",
		Description: "Computes outcome probabilities across duality dice",
	}
}

// actionRollHandler executes a duality action roll.
func actionRollHandler(client pb.DiceRollServiceClient) mcp.ToolHandlerFor[ActionRollInput, ActionRollResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ActionRollInput) (*mcp.CallToolResult, ActionRollResult, error) {
		modifier := input.Modifier
		if input.Difficulty != nil && *input.Difficulty < 0 {
			return nil, ActionRollResult{}, fmt.Errorf("difficulty must be non-negative")
		}

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
func dualityOutcomeHandler(client pb.DiceRollServiceClient) mcp.ToolHandlerFor[DualityOutcomeInput, DualityOutcomeResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DualityOutcomeInput) (*mcp.CallToolResult, DualityOutcomeResult, error) {
		if input.Hope < 1 || input.Hope > 12 || input.Fear < 1 || input.Fear > 12 {
			return nil, DualityOutcomeResult{}, fmt.Errorf("hope and fear must be between 1 and 12")
		}
		if input.Difficulty != nil && *input.Difficulty < 0 {
			return nil, DualityOutcomeResult{}, fmt.Errorf("difficulty must be non-negative")
		}

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

// dualityProbabilityHandler executes the deterministic probability evaluation.
func dualityProbabilityHandler(client pb.DiceRollServiceClient) mcp.ToolHandlerFor[DualityProbabilityInput, DualityProbabilityResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DualityProbabilityInput) (*mcp.CallToolResult, DualityProbabilityResult, error) {
		if input.Difficulty < 0 {
			return nil, DualityProbabilityResult{}, fmt.Errorf("difficulty must be non-negative")
		}

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

// rollDiceHandler executes a generic dice roll.
func rollDiceHandler(client pb.DiceRollServiceClient) mcp.ToolHandlerFor[RollDiceInput, RollDiceResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input RollDiceInput) (*mcp.CallToolResult, RollDiceResult, error) {
		if len(input.Dice) == 0 {
			return nil, RollDiceResult{}, fmt.Errorf("at least one die must be provided")
		}

		diceSpecs := make([]*pb.DiceSpec, 0, len(input.Dice))
		for _, spec := range input.Dice {
			if spec.Sides <= 0 || spec.Count <= 0 {
				return nil, RollDiceResult{}, fmt.Errorf("dice must have positive sides and count")
			}
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

// newDiceRollClient connects to the gRPC dice service.
func newDiceRollClient(addr string) (pb.DiceRollServiceClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return pb.NewDiceRollServiceClient(conn), nil
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
