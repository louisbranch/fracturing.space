//go:build integration

package integration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	aiservice "github.com/louisbranch/fracturing.space/internal/services/ai/api/grpc/ai"
	aiapp "github.com/louisbranch/fracturing.space/internal/services/ai/app"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	mcpservice "github.com/louisbranch/fracturing.space/internal/services/mcp/service"
	grpcauthctx "github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	daggerheartReferenceRoot            = "/home/louis/code/daggerheart/reference-corpus/v1/reference"
	aiGMBootstrapScenarioName           = "ai_gm_campaign_context_bootstrap"
	aiGMBootstrapFixtureFile            = "ai_gm_campaign_context_bootstrap_replay.json"
	aiGMBootstrapPrompt                 = "Open the session, consult the Fear reference first, and remember the harbor debt."
	aiGMBootstrapStorySeed              = "Starter seed: The Black Lantern warns of a debt collected at dawn."
	aiGMBootstrapMemorySeed             = "Remember: the harbor master owes the party a favor."
	integrationOpenAIAPIKeyEnv          = "INTEGRATION_OPENAI_API_KEY"
	integrationAIModelEnv               = "INTEGRATION_AI_MODEL"
	integrationAIReasoningEffortEnv     = "INTEGRATION_AI_REASONING_EFFORT"
	integrationAIWriteFixtureEnv        = "INTEGRATION_AI_WRITE_FIXTURE"
	integrationOpenAIResponsesTargetEnv = "INTEGRATION_OPENAI_RESPONSES_URL"
	defaultOpenAIResponsesTargetURL     = "https://api.openai.com/v1/responses"
)

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
	aiAddr := pickUnusedAddress(t)
	t.Setenv("FRACTURING_SPACE_AI_ADDR", aiAddr)
	fixture := newSuiteFixture(t)
	userID := fixture.newUserID(t, uniqueTestUsername(t, "ai-gm-context"))

	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", fixture.grpcAddr)
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_RESPONSES_URL", strings.TrimSpace(opts.ResponsesURL))
	t.Setenv("FRACTURING_SPACE_AI_DAGGERHEART_REFERENCE_ROOT", daggerheartReferenceRoot)

	mcpAddr := pickUnusedAddress(t)
	t.Setenv("FRACTURING_SPACE_AI_MCP_URL", "http://"+mcpAddr+"/mcp")

	aiCtx, cancelAI := context.WithCancel(context.Background())
	aiServer, err := aiapp.NewWithAddrContext(aiCtx, aiAddr)
	if err != nil {
		cancelAI()
		t.Fatalf("new ai server: %v", err)
	}
	aiServeErr := make(chan error, 1)
	go func() {
		aiServeErr <- aiServer.Serve(aiCtx)
	}()
	t.Cleanup(func() {
		cancelAI()
		select {
		case err := <-aiServeErr:
			if err != nil {
				t.Fatalf("ai server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for ai server to stop")
		}
	})
	waitForGRPCHealth(t, aiAddr)

	mcpCtx, cancelMCP := context.WithCancel(context.Background())
	mcpErr := make(chan error, 1)
	go func() {
		mcpErr <- mcpservice.Run(mcpCtx, mcpservice.Config{
			GRPCAddr:  fixture.grpcAddr,
			AIAddr:    aiAddr,
			Transport: mcpservice.TransportHTTP,
			HTTPAddr:  mcpAddr,
		})
	}()
	t.Cleanup(func() {
		cancelMCP()
		select {
		case err := <-mcpErr:
			if err != nil && !strings.Contains(strings.ToLower(err.Error()), "context canceled") {
				t.Fatalf("mcp server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for mcp server to stop")
		}
	})
	waitForHTTPHealth(t, newHTTPClient(t), "http://"+mcpAddr+"/mcp/health")

	gameConn := dialGRPCForIntegration(t, fixture.grpcAddr)
	defer gameConn.Close()
	aiConn := dialGRPCForIntegration(t, aiAddr)
	defer aiConn.Close()

	credentialClient := aiv1.NewCredentialServiceClient(aiConn)
	agentClient := aiv1.NewAgentServiceClient(aiConn)
	artifactClient := aiv1.NewCampaignArtifactServiceClient(aiConn)
	campaignClient := gamev1.NewCampaignServiceClient(gameConn)
	participantClient := gamev1.NewParticipantServiceClient(gameConn)
	characterClient := gamev1.NewCharacterServiceClient(gameConn)
	sessionClient := gamev1.NewSessionServiceClient(gameConn)
	sceneClient := gamev1.NewSceneServiceClient(gameConn)
	interactionClient := gamev1.NewInteractionServiceClient(gameConn)

	ctxWithUser := grpcauthctx.WithUserID(context.Background(), userID)

	credentialResp, err := credentialClient.CreateCredential(ctxWithUser, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    "Replay credential",
		Secret:   strings.TrimSpace(opts.CredentialSecret),
	})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	agentResp, err := agentClient.CreateAgent(ctxWithUser, &aiv1.CreateAgentRequest{
		Label:        strings.TrimSpace(opts.AgentLabel),
		Provider:     aiv1.Provider_PROVIDER_OPENAI,
		Model:        strings.TrimSpace(opts.Model),
		CredentialId: credentialResp.GetCredential().GetId(),
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	campaignResp, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:        "Replay Harbor",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_AI,
		ThemePrompt: "A debt comes due at the harbor.",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	campaignID := campaignResp.GetCampaign().GetId()
	if _, err := campaignClient.SetCampaignAIBinding(ctxWithUser, &gamev1.SetCampaignAIBindingRequest{
		CampaignId: campaignID,
		AiAgentId:  agentResp.GetAgent().GetId(),
	}); err != nil {
		t.Fatalf("set campaign ai binding: %v", err)
	}
	participantsResp, err := participantClient.ListParticipants(ctxWithUser, &gamev1.ListParticipantsRequest{
		CampaignId: campaignID,
		PageSize:   50,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	aiGMParticipantID := ""
	for _, participant := range participantsResp.GetParticipants() {
		if participant.GetRole() == gamev1.ParticipantRole_GM && participant.GetController() == gamev1.Controller_CONTROLLER_AI {
			aiGMParticipantID = strings.TrimSpace(participant.GetId())
			break
		}
	}
	if aiGMParticipantID == "" {
		t.Fatal("expected ai gm participant")
	}
	ensureSessionStartReadiness(t, ctxWithUser, participantClient, characterClient, campaignID, campaignResp.GetOwnerParticipant().GetId())
	charactersResp, err := characterClient.ListCharacters(ctxWithUser, &gamev1.ListCharactersRequest{
		CampaignId: campaignID,
		PageSize:   20,
	})
	if err != nil {
		t.Fatalf("list characters: %v", err)
	}
	if len(charactersResp.GetCharacters()) == 0 || strings.TrimSpace(charactersResp.GetCharacters()[0].GetId()) == "" {
		t.Fatal("expected at least one campaign character for replay bootstrap")
	}
	characterID := strings.TrimSpace(charactersResp.GetCharacters()[0].GetId())
	if _, err := artifactClient.EnsureCampaignArtifacts(ctxWithUser, &aiv1.EnsureCampaignArtifactsRequest{
		CampaignId:        campaignID,
		StorySeedMarkdown: aiGMBootstrapStorySeed,
	}); err != nil {
		t.Fatalf("ensure campaign artifacts: %v", err)
	}
	if _, err := artifactClient.UpsertCampaignArtifact(ctxWithUser, &aiv1.UpsertCampaignArtifactRequest{
		CampaignId: campaignID,
		Path:       "memory.md",
		Content:    aiGMBootstrapMemorySeed,
	}); err != nil {
		t.Fatalf("seed memory artifact: %v", err)
	}

	startResp, err := sessionClient.StartSession(ctxWithUser, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Opening Night",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	sessionID := startResp.GetSession().GetId()

	if opts.BeforeRun != nil {
		opts.BeforeRun(aiGMBootstrapSetup{
			CampaignID:        campaignID,
			SessionID:         sessionID,
			CharacterID:       characterID,
			AIGMParticipantID: aiGMParticipantID,
		})
	}

	provider := aiservice.NewOpenAIInvokeAdapter(aiservice.OpenAIInvokeConfig{
		ResponsesURL: strings.TrimSpace(opts.ResponsesURL),
	})
	runner := orchestration.NewRunner(orchestration.NewMCPDialer("http://"+mcpAddr+"/mcp", newHTTPClient(t)), 12)
	runResp, err := runner.Run(context.Background(), orchestration.Input{
		CampaignID:       campaignID,
		SessionID:        sessionID,
		ParticipantID:    aiGMParticipantID,
		Input:            aiGMBootstrapPrompt,
		Model:            strings.TrimSpace(opts.Model),
		ReasoningEffort:  strings.TrimSpace(opts.ReasoningEffort),
		CredentialSecret: strings.TrimSpace(opts.CredentialSecret),
		Provider:         provider.(orchestration.Provider),
	})
	if err != nil {
		t.Fatalf("run campaign turn: %v", err)
	}

	skillsResp, err := artifactClient.GetCampaignArtifact(ctxWithUser, &aiv1.GetCampaignArtifactRequest{
		CampaignId: campaignID,
		Path:       "skills.md",
	})
	if err != nil {
		t.Fatalf("get skills artifact: %v", err)
	}
	memoryResp, err := artifactClient.GetCampaignArtifact(ctxWithUser, &aiv1.GetCampaignArtifactRequest{
		CampaignId: campaignID,
		Path:       "memory.md",
	})
	if err != nil {
		t.Fatalf("get memory artifact: %v", err)
	}
	scenesResp, err := sceneClient.ListScenes(ctxWithUser, &gamev1.ListScenesRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list scenes: %v", err)
	}
	interactionResp, err := interactionClient.GetInteractionState(ctxWithUser, &gamev1.GetInteractionStateRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		t.Fatalf("get interaction state: %v", err)
	}
	activeSceneID := ""
	if interactionResp.GetState() != nil {
		activeSceneID = strings.TrimSpace(interactionResp.GetState().GetActiveScene().GetSceneId())
	}
	if activeSceneID == "" {
		t.Fatal("expected active interaction scene")
	}
	sceneIsActive := false
	if len(scenesResp.GetScenes()) != 0 {
		sceneIsActive = scenesResp.GetScenes()[0].GetActive()
	}
	playerPhaseOpen := interactionResp.GetState() != nil &&
		interactionResp.GetState().GetPlayerPhase().GetStatus() == gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS

	return aiGMBootstrapResult{
		CampaignID:      campaignID,
		SessionID:       sessionID,
		CharacterID:     characterID,
		AIGMParticipant: aiGMParticipantID,
		OutputText:      strings.TrimSpace(runResp.OutputText),
		MemoryContent:   strings.TrimSpace(memoryResp.GetArtifact().GetContent()),
		SkillsReadOnly:  skillsResp.GetArtifact().GetReadOnly(),
		ActiveSceneID:   activeSceneID,
		SceneCount:      len(scenesResp.GetScenes()),
		SceneIsActive:   sceneIsActive,
		PlayerPhaseOpen: playerPhaseOpen,
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
func bootstrapPromptContains() []string {
	return []string{
		"story.md:",
		aiGMBootstrapStorySeed,
		"memory.md:",
		aiGMBootstrapMemorySeed,
	}
}

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
func replayFixtureMemoryContent(t *testing.T, fixture openAIReplayFixture) string {
	t.Helper()
	for stepIndex := len(fixture.Steps) - 1; stepIndex >= 0; stepIndex-- {
		step := fixture.Steps[stepIndex]
		for callIndex := len(step.ToolCalls) - 1; callIndex >= 0; callIndex-- {
			call := step.ToolCalls[callIndex]
			if strings.TrimSpace(call.Name) != "campaign_artifact_upsert" {
				continue
			}
			if strings.TrimSpace(asString(call.Arguments["path"])) != "memory.md" {
				continue
			}
			content := strings.TrimSpace(asString(call.Arguments["content"]))
			if content == "" {
				t.Fatal("replay fixture memory.md write is missing content")
			}
			return content
		}
	}
	t.Fatal("replay fixture is missing a memory.md upsert")
	return ""
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
		return "gpt-5.4"
	}
	return model
}

// liveAIReasoningEffort pins the live capture lane to one default reasoning effort while allowing intentional overrides.
func liveAIReasoningEffort() string {
	effort := strings.TrimSpace(os.Getenv(integrationAIReasoningEffortEnv))
	if effort == "" {
		return "medium"
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
