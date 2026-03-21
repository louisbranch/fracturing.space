package orchestration

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

// CampaignTurnRunner executes one provider turn for GM control.
type CampaignTurnRunner interface {
	Run(ctx context.Context, input Input) (Result, error)
}

// DefaultCommitToolName is the tool name that signals a committed narration
// output during a campaign turn. Callers may override via RunnerConfig.
const DefaultCommitToolName = "interaction_scene_gm_output_commit"

// RunnerConfig defines orchestration runtime policy for campaign turns.
type RunnerConfig struct {
	Dialer             Dialer
	PromptBuilder      PromptBuilder
	ToolPolicy         ToolPolicy
	MaxSteps           int
	TurnTimeout        time.Duration
	ToolResultMaxBytes int
	// CommitToolName identifies which tool call signals a committed narration.
	// Defaults to DefaultCommitToolName when empty.
	CommitToolName string
}

// PromptBuilder assembles the prompt for one campaign turn.
type PromptBuilder interface {
	Build(ctx context.Context, sess Session, input PromptInput) (string, error)
}

// SessionBriefCollector gathers the typed session brief used for one prompt.
type SessionBriefCollector interface {
	CollectBrief(ctx context.Context, sess Session, input PromptInput) (SessionBrief, error)
}

// PromptRenderer renders one prompt from a collected brief plus prompt input.
type PromptRenderer interface {
	Render(brief SessionBrief, input PromptInput) string
}

// ContextSource contributes to the typed session brief used for prompt
// assembly. Game systems implement this interface to inject system-specific
// context alongside the core campaign context.
type ContextSource interface {
	Collect(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error)
}

// Provider executes one provider step in the campaign-turn tool loop.
type Provider interface {
	Run(ctx context.Context, input ProviderInput) (ProviderOutput, error)
}

// Dialer opens one session for a single orchestration run.
type Dialer interface {
	Dial(ctx context.Context) (Session, error)
}

// Session exposes the tool/resource operations used during campaign orchestration.
type Session interface {
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, name string, args any) (ToolResult, error)
	ReadResource(ctx context.Context, uri string) (string, error)
	Close() error
}

// PromptInput contains only the prompt-relevant fields needed to assemble one
// campaign turn brief.
type PromptInput struct {
	CampaignID    string
	SessionID     string
	ParticipantID string
	TurnInput     string
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

// Tool mirrors the provider-facing subset of one tool definition.
type Tool struct {
	Name        string
	Description string
	InputSchema any
}

// ToolResult captures one tool result for model feedback.
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

// ProviderToolResult maps one tool result back to the provider.
type ProviderToolResult struct {
	CallID  string
	Output  string
	IsError bool
}
