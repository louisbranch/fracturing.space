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
	Hope       int    `json:"hope" jsonschema:"hope die result"`
	Fear       int    `json:"fear" jsonschema:"fear die result"`
	Total      int    `json:"total" jsonschema:"sum of dice and modifier"`
	Modifier   int    `json:"modifier" jsonschema:"modifier applied to the total"`
	Outcome    string `json:"outcome" jsonschema:"categorized roll outcome"`
	Difficulty *int   `json:"difficulty,omitempty" jsonschema:"difficulty target, if provided"`
}

// ActionRollInput represents the MCP tool input for an action roll.
type ActionRollInput struct {
	Modifier   int  `json:"modifier" jsonschema:"modifier applied to the roll"`
	Difficulty *int `json:"difficulty" jsonschema:"optional difficulty target"`
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
		if response.GetDuality() == nil {
			return nil, ActionRollResult{}, fmt.Errorf("action roll response missing dice")
		}

		result := ActionRollResult{
			Hope:     int(response.GetDuality().GetHopeD12()),
			Fear:     int(response.GetDuality().GetFearD12()),
			Total:    int(response.GetTotal()),
			Modifier: modifier,
			Outcome:  response.GetOutcome().String(),
		}
		if response.Difficulty != nil {
			value := int(response.GetDifficulty())
			result.Difficulty = &value
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
