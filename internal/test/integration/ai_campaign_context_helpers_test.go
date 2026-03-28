//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaignartifact"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/referencecorpus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	daggerheartReferenceRoot            = "/home/louis/code/daggerheart/reference-corpus/v1/reference"
	integrationOpenAIAPIKeyEnv          = "INTEGRATION_OPENAI_API_KEY"
	integrationAIModelEnv               = "INTEGRATION_AI_MODEL"
	integrationAIReasoningEffortEnv     = "INTEGRATION_AI_REASONING_EFFORT"
	integrationAIWriteFixtureEnv        = "INTEGRATION_AI_WRITE_FIXTURE"
	integrationOpenAIResponsesTargetEnv = "INTEGRATION_OPENAI_RESPONSES_URL"
	defaultOpenAIResponsesTargetURL     = "https://api.openai.com/v1/responses"
)

// grpcArtifactAdapter bridges a gRPC CampaignArtifactServiceClient to the
// gametools.ArtifactManager interface for integration testing.
type grpcArtifactAdapter struct {
	client aiv1.CampaignArtifactServiceClient
}

func (a *grpcArtifactAdapter) ListArtifacts(ctx context.Context, campaignID string) ([]campaignartifact.Artifact, error) {
	resp, err := a.client.ListCampaignArtifacts(ctx, &aiv1.ListCampaignArtifactsRequest{CampaignId: campaignID})
	if err != nil {
		return nil, err
	}
	out := make([]campaignartifact.Artifact, len(resp.GetArtifacts()))
	for i, art := range resp.GetArtifacts() {
		out[i] = artifactProtoToRecord(art)
	}
	return out, nil
}

func (a *grpcArtifactAdapter) GetArtifact(ctx context.Context, campaignID, path string) (campaignartifact.Artifact, error) {
	resp, err := a.client.GetCampaignArtifact(ctx, &aiv1.GetCampaignArtifactRequest{CampaignId: campaignID, Path: path})
	if err != nil {
		return campaignartifact.Artifact{}, err
	}
	return artifactProtoToRecord(resp.GetArtifact()), nil
}

func (a *grpcArtifactAdapter) UpsertArtifact(ctx context.Context, campaignID, path, content string) (campaignartifact.Artifact, error) {
	resp, err := a.client.UpsertCampaignArtifact(ctx, &aiv1.UpsertCampaignArtifactRequest{CampaignId: campaignID, Path: path, Content: content})
	if err != nil {
		return campaignartifact.Artifact{}, err
	}
	return artifactProtoToRecord(resp.GetArtifact()), nil
}

func artifactProtoToRecord(art *aiv1.CampaignArtifact) campaignartifact.Artifact {
	r := campaignartifact.Artifact{
		CampaignID: art.GetCampaignId(),
		Path:       art.GetPath(),
		Content:    art.GetContent(),
		ReadOnly:   art.GetReadOnly(),
	}
	if ts := art.GetCreatedAt(); ts != nil {
		r.CreatedAt = ts.AsTime()
	}
	if ts := art.GetUpdatedAt(); ts != nil {
		r.UpdatedAt = ts.AsTime()
	}
	return r
}

// grpcReferenceAdapter bridges a gRPC SystemReferenceServiceClient to the
// gametools.ReferenceCorpus interface for integration testing.
type grpcReferenceAdapter struct {
	client aiv1.SystemReferenceServiceClient
}

func (a *grpcReferenceAdapter) Search(ctx context.Context, system, query string, maxResults int) ([]referencecorpus.SearchResult, error) {
	resp, err := a.client.SearchSystemReference(ctx, &aiv1.SearchSystemReferenceRequest{System: system, Query: query, MaxResults: int32(maxResults)})
	if err != nil {
		return nil, err
	}
	out := make([]referencecorpus.SearchResult, len(resp.GetResults()))
	for i, r := range resp.GetResults() {
		out[i] = referencecorpus.SearchResult{
			System:     r.GetSystem(),
			DocumentID: r.GetDocumentId(),
			Title:      r.GetTitle(),
			Kind:       r.GetKind(),
			Path:       r.GetPath(),
			Aliases:    r.GetAliases(),
			Snippet:    r.GetSnippet(),
		}
	}
	return out, nil
}

func (a *grpcReferenceAdapter) Read(ctx context.Context, system, documentID string) (referencecorpus.Document, error) {
	resp, err := a.client.ReadSystemReferenceDocument(ctx, &aiv1.ReadSystemReferenceDocumentRequest{System: system, DocumentId: documentID})
	if err != nil {
		return referencecorpus.Document{}, err
	}
	doc := resp.GetDocument()
	return referencecorpus.Document{
		System:     doc.GetSystem(),
		DocumentID: doc.GetDocumentId(),
		Title:      doc.GetTitle(),
		Kind:       doc.GetKind(),
		Path:       doc.GetPath(),
		Aliases:    doc.GetAliases(),
		Content:    doc.GetContent(),
	}, nil
}

// aiGMBootstrapSetup exposes the run-specific IDs that a caller may need to bind into a recorder before execution.
type aiGMBootstrapSetup struct {
	CampaignID        string
	SessionID         string
	CharacterID       string
	AIGMParticipantID string
}

// aiGMBootstrapResult exposes only the durable scenario outcomes that replay and live lanes both assert.
type aiGMBootstrapResult struct {
	CampaignID      string
	SessionID       string
	CharacterID     string
	AIGMParticipant string
	OutputText      string
	MemoryContent   string
	SkillsReadOnly  bool
	ActiveSceneID   string
	SceneCount      int
	SceneIsActive   bool
	PlayerPhaseOpen bool
}

// aiGMBootstrapScenarioOptions keeps the bootstrap harness configurable without duplicating setup logic.
type aiGMBootstrapScenarioOptions struct {
	ResponsesURL     string
	Model            string
	ReasoningEffort  string
	CredentialSecret string
	AgentLabel       string
	BeforeRun        func(aiGMBootstrapSetup)
}

// runAIGMCampaignContextBootstrapScenario exercises the full GM bootstrap seam against real game, AI, and MCP services.
func runAIGMCampaignContextBootstrapScenario(t *testing.T, opts aiGMBootstrapScenarioOptions) aiGMBootstrapResult {
	t.Helper()
	result := runAIGMCampaignContextScenario(t, aiGMBootstrapScenario, aiGMCampaignScenarioOptions{
		ResponsesURL:     opts.ResponsesURL,
		Model:            opts.Model,
		ReasoningEffort:  opts.ReasoningEffort,
		CredentialSecret: opts.CredentialSecret,
		AgentLabel:       opts.AgentLabel,
		BeforeRun: func(setup aiGMCampaignScenarioSetup) {
			if opts.BeforeRun != nil {
				opts.BeforeRun(aiGMBootstrapSetup{
					CampaignID:        setup.CampaignID,
					SessionID:         setup.SessionID,
					CharacterID:       setup.CharacterID,
					AIGMParticipantID: setup.AIGMParticipantID,
				})
			}
		},
	})

	return aiGMBootstrapResult{
		CampaignID:      result.CampaignID,
		SessionID:       result.SessionID,
		CharacterID:     result.CharacterID,
		AIGMParticipant: result.AIGMParticipantID,
		OutputText:      result.OutputText,
		MemoryContent:   result.MemoryContent,
		SkillsReadOnly:  result.SkillsReadOnly,
		ActiveSceneID:   activeSceneID(result.InteractionState),
		SceneCount:      len(result.Scenes),
		SceneIsActive:   sceneOpenByID(result.Scenes, activeSceneID(result.InteractionState)),
		PlayerPhaseOpen: playerPhaseOpen(result.InteractionState),
	}
}

// dialGRPCForIntegration centralizes blocking dial behavior so the integration harness stays consistent.
func dialGRPCForIntegration(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		t.Fatalf("dial grpc %s: %v", addr, err)
	}
	return conn
}

// loadOpenAIReplayFixture reads committed replay fixtures from the canonical integration-fixture location.
func loadOpenAIReplayFixture(t *testing.T, name string) openAIReplayFixture {
	t.Helper()
	path := filepath.Join(repoRoot(t), "internal/test/integration/fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read replay fixture: %v", err)
	}
	var fixture openAIReplayFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("parse replay fixture: %v", err)
	}
	return fixture
}

// writeOpenAIReplayFixture updates the canonical replay fixture only when the live lane opts in explicitly.
func writeOpenAIReplayFixture(t *testing.T, name string, fixture openAIReplayFixture) string {
	t.Helper()
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatalf("marshal replay fixture: %v", err)
	}
	data = append(data, '\n')
	path := filepath.Join(repoRoot(t), "internal/test/integration/fixtures", name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write replay fixture: %v", err)
	}
	return path
}

// bootstrapPromptContains captures the minimum context strings that must survive prompt assembly.
// These must be emitted by the default (degraded) prompt builder, which has no
// pre-loaded instruction content — so they come from artifacts, context sources,
// and the inline interaction contract fallback.
// artifactListContains keeps artifact presence assertions readable in the integration tests.
func artifactListContains(artifacts []*aiv1.CampaignArtifact, path string) bool {
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.GetPath()) == strings.TrimSpace(path) {
			return true
		}
	}
	return false
}

// openAIReplayFixtureToolNames extracts the unique tool names seen in a replay fixture for coverage assertions.
func openAIReplayFixtureToolNames(fixture openAIReplayFixture) []string {
	names := make([]string, 0, len(fixture.Steps))
	for _, step := range fixture.Steps {
		for _, toolCall := range step.ToolCalls {
			name := strings.TrimSpace(toolCall.Name)
			if name == "" || slices.Contains(names, name) {
				continue
			}
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

func replayFixtureToolCalls(fixture openAIReplayFixture) []openAIReplayToolCall {
	calls := make([]openAIReplayToolCall, 0)
	for _, step := range fixture.Steps {
		calls = append(calls, step.ToolCalls...)
	}
	return calls
}

func mustReplayFixtureToolCall(t *testing.T, fixture openAIReplayFixture, name string, ordinal int) openAIReplayToolCall {
	t.Helper()
	if ordinal < 1 {
		t.Fatalf("ordinal must be >= 1, got %d", ordinal)
	}
	count := 0
	for _, call := range replayFixtureToolCalls(fixture) {
		if strings.TrimSpace(call.Name) != strings.TrimSpace(name) {
			continue
		}
		count++
		if count == ordinal {
			return call
		}
	}
	t.Fatalf("tool %q occurrence %d not found in fixture sequence %v", name, ordinal, openAIReplayFixtureToolNames(fixture))
	return openAIReplayToolCall{}
}

// replayFixtureFinalOutputText returns the final narrated output captured in the replay fixture.
func replayFixtureFinalOutputText(t *testing.T, fixture openAIReplayFixture) string {
	t.Helper()
	if len(fixture.Steps) == 0 {
		t.Fatal("replay fixture has no steps")
	}
	text := strings.TrimSpace(fixture.Steps[len(fixture.Steps)-1].OutputText)
	if text == "" {
		t.Fatal("replay fixture final step is missing output_text")
	}
	return text
}

// replayFixtureMemoryContent returns the most recent memory.md write encoded in the replay fixture.
// It accepts either campaign_artifact_upsert (full doc write) or campaign_memory_section_update
// (section-level write) as valid memory write tools.
func replayFixtureMemoryContent(t *testing.T, fixture openAIReplayFixture) string {
	t.Helper()
	for stepIndex := len(fixture.Steps) - 1; stepIndex >= 0; stepIndex-- {
		step := fixture.Steps[stepIndex]
		for callIndex := len(step.ToolCalls) - 1; callIndex >= 0; callIndex-- {
			call := step.ToolCalls[callIndex]
			name := strings.TrimSpace(call.Name)
			switch name {
			case "campaign_artifact_upsert":
				if strings.TrimSpace(asString(call.Arguments["path"])) != "memory.md" {
					continue
				}
				content := strings.TrimSpace(asString(call.Arguments["content"]))
				if content == "" {
					t.Fatal("replay fixture memory.md write is missing content")
				}
				return content
			case "campaign_memory_section_update":
				content := strings.TrimSpace(asString(call.Arguments["content"]))
				if content == "" {
					t.Fatal("replay fixture memory section update is missing content")
				}
				return content
			}
		}
	}
	t.Fatal("replay fixture is missing a memory.md write (artifact_upsert or memory_section_update)")
	return ""
}

// replayFixtureMemoryWriteIsSectionUpdate reports whether the replay fixture
// used a section-level memory write rather than a full-document upsert.
func replayFixtureMemoryWriteIsSectionUpdate(fixture openAIReplayFixture) bool {
	for stepIndex := len(fixture.Steps) - 1; stepIndex >= 0; stepIndex-- {
		step := fixture.Steps[stepIndex]
		for callIndex := len(step.ToolCalls) - 1; callIndex >= 0; callIndex-- {
			call := step.ToolCalls[callIndex]
			name := strings.TrimSpace(call.Name)
			switch name {
			case "campaign_memory_section_update":
				return true
			case "campaign_artifact_upsert":
				if strings.TrimSpace(asString(call.Arguments["path"])) == "memory.md" {
					return false
				}
			}
		}
	}
	return false
}

// newHTTPClient returns a reasonably-configured client for the live capture proxy.
func newHTTPClient(t *testing.T) *http.Client {
	t.Helper()
	return &http.Client{Timeout: 60 * time.Second}
}

// envEnabled standardizes the small opt-in flags used by the manual live-capture lane.
func envEnabled(name string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return value == "1" || value == "true" || value == "yes"
}

// liveAIModel pins the capture lane to one default model while still allowing intentional overrides.
func liveAIModel() string {
	model := strings.TrimSpace(os.Getenv(integrationAIModelEnv))
	if model == "" {
		return "gpt-5-mini"
	}
	return model
}

// liveAIReasoningEffort pins the live capture lane to one default reasoning effort while allowing intentional overrides.
func liveAIReasoningEffort() string {
	effort := strings.TrimSpace(os.Getenv(integrationAIReasoningEffortEnv))
	if effort == "" {
		return ""
	}
	return effort
}

// liveOpenAIResponsesTargetURL lets the recorder proxy point at alternate OpenAI-compatible endpoints when needed.
func liveOpenAIResponsesTargetURL() string {
	target := strings.TrimSpace(os.Getenv(integrationOpenAIResponsesTargetEnv))
	if target == "" {
		return defaultOpenAIResponsesTargetURL
	}
	return target
}

// requiredToolSetPresent fails fast when a live capture did not exercise the minimum GM bootstrap tool surface.
func requiredToolSetPresent(names []string, required ...string) error {
	for _, name := range required {
		if !slices.Contains(names, name) {
			return fmt.Errorf("missing required tool %q", name)
		}
	}
	return nil
}
