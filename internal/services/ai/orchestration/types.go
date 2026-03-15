package orchestration

import (
	"context"
	"net/http"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

// CampaignTurnRunner executes one MCP-augmented provider turn for GM control.
type CampaignTurnRunner interface {
	Run(ctx context.Context, input Input) (Result, error)
}

// RunnerConfig defines orchestration runtime policy for campaign turns.
type RunnerConfig struct {
	Dialer             Dialer
	PromptBuilder      PromptBuilder
	MaxSteps           int
	TurnTimeout        time.Duration
	ToolResultMaxBytes int
}

// PromptBuilder assembles the MCP-backed prompt for one campaign turn.
type PromptBuilder interface {
	Build(ctx context.Context, sess Session, input Input) (string, error)
}

// Provider executes one provider step in the campaign-turn tool loop.
type Provider interface {
	Run(ctx context.Context, input ProviderInput) (ProviderOutput, error)
}

// Dialer opens one MCP session for a single orchestration run.
type Dialer interface {
	Dial(ctx context.Context) (Session, error)
}

// MCPDialerConfig defines how the orchestration layer opens one MCP session.
type MCPDialerConfig struct {
	URL         string
	HTTPClient  *http.Client
	DialTimeout time.Duration
}

// Session exposes the MCP operations used during campaign orchestration.
type Session interface {
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, name string, args any) (ToolResult, error)
	ReadResource(ctx context.Context, uri string) (string, error)
	Close() error
}

// Input contains all data required to run one campaign turn.
type Input struct {
	CampaignID       string
	SessionID        string
	ParticipantID    string
	Input            string
	Model            string
	ReasoningEffort  string
	Instructions     string
	CredentialSecret string
	Provider         Provider
}

// Result contains the final narrated output for a campaign turn.
type Result struct {
	OutputText string
	Usage      provider.Usage
}

// Tool mirrors the provider-facing subset of one MCP tool definition.
type Tool struct {
	Name        string
	Description string
	InputSchema any
}

// ToolResult captures one MCP tool result for model feedback.
type ToolResult struct {
	Output  string
	IsError bool
}

// ProviderInput contains provider input for either the initial prompt or a
// follow-up batch of tool outputs.
type ProviderInput struct {
	Model            string
	ReasoningEffort  string
	Instructions     string
	Prompt           string
	FollowUpPrompt   string
	CredentialSecret string
	Tools            []Tool
	ConversationID   string
	Results          []ProviderToolResult
}

// ProviderOutput contains either tool calls or final model output.
type ProviderOutput struct {
	ConversationID string
	OutputText     string
	ToolCalls      []ProviderToolCall
	Usage          provider.Usage
}

// ProviderToolCall captures one provider-requested tool invocation.
type ProviderToolCall struct {
	CallID    string
	Name      string
	Arguments string
}

// ProviderToolResult maps one MCP tool result back to the provider.
type ProviderToolResult struct {
	CallID  string
	Output  string
	IsError bool
}
