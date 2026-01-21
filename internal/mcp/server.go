// Package mcp provides the MCP server for duality rolls.
package mcp

import (
	"context"
	"fmt"
	"os"
	"time"

	pb "github.com/louisbranch/duality-protocol/api/gen/go/duality/v1"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
	mcpServer *server.MCPServer
}

// ActionRollResult represents the MCP tool output for an action roll.
type ActionRollResult struct {
	Hope       int    `json:"hope"`
	Fear       int    `json:"fear"`
	Total      int    `json:"total"`
	Modifier   int    `json:"modifier"`
	Outcome    string `json:"outcome"`
	Difficulty *int   `json:"difficulty,omitempty"`
}

// ActionRollInput represents the MCP tool input for an action roll.
type ActionRollInput struct {
	Modifier   int  `json:"modifier"`
	Difficulty *int `json:"difficulty"`
}

// New creates a configured MCP server that connects to the gRPC dice service.
func New(addr string) (*Server, error) {
	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(false),
	)

	grpcAddr := grpcAddress(addr)
	grpcClient, err := newDiceRollClient(grpcAddr)
	if err != nil {
		return nil, fmt.Errorf("connect to gRPC server at %s: %w", grpcAddr, err)
	}

	mcpServer.AddTool(actionRollTool(), actionRollHandler(grpcClient))

	return &Server{mcpServer: mcpServer}, nil
}

// Serve starts the MCP server on stdio.
func (s *Server) Serve() error {
	if s == nil || s.mcpServer == nil {
		return fmt.Errorf("MCP server is not configured")
	}
	if err := server.ServeStdio(s.mcpServer); err != nil {
		return fmt.Errorf("serve MCP: %w", err)
	}
	return nil
}

// actionRollTool defines the MCP tool schema for action rolls.
func actionRollTool() mcp.Tool {
	return mcp.NewTool(
		"duality_action_roll",
		mcp.WithDescription("Rolls Duality dice for an action"),
		mcp.WithNumber("modifier",
			mcp.Description("Additive modifier applied to the dice total"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("difficulty",
			mcp.Description("Optional difficulty target for success"),
			mcp.Min(0),
		),
		mcp.WithInputSchema[ActionRollInput](),
		mcp.WithOutputSchema[ActionRollResult](),
	)
}

// actionRollHandler executes a duality action roll.
func actionRollHandler(client pb.DiceRollServiceClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input ActionRollInput
		if err := request.BindArguments(&input); err != nil {
			return mcp.NewToolResultErrorFromErr("invalid action roll arguments", err), nil
		}

		modifier := input.Modifier
		if input.Difficulty != nil && *input.Difficulty < 0 {
			return mcp.NewToolResultError("difficulty must be non-negative"), nil
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
			return mcp.NewToolResultErrorFromErr("action roll failed", err), nil
		}
		if response.GetDuality() == nil {
			return mcp.NewToolResultError("action roll response missing dice"), nil
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

		return mcp.NewToolResultStructuredOnly(result), nil
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
