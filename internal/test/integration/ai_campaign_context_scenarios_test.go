//go:build integration

package integration

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	aiapp "github.com/louisbranch/fracturing.space/internal/services/ai/app"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/gametools"
	openaiprovider "github.com/louisbranch/fracturing.space/internal/services/ai/provider/openai"
	grpcauthctx "github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	aiGMBootstrapScenarioName     = "ai_gm_campaign_context_bootstrap"
	aiGMBootstrapFixtureFile      = "ai_gm_campaign_context_bootstrap_replay.json"
	aiGMBootstrapPrompt           = "Open the session, consult the Fear reference first, and update memory.md with session notes about the harbor debt."
	aiGMBootstrapStorySeed        = "Starter seed: The Black Lantern warns of a debt collected at dawn."
	aiGMBootstrapMemorySeed       = "Remember: the harbor master owes the party a favor."
	aiGMReviewAdvanceScenarioName = "ai_gm_campaign_context_review_advance"
	aiGMReviewAdvanceFixtureFile  = "ai_gm_campaign_context_review_advance_replay.json"
	aiGMReviewAdvancePrompt       = "Resolve the flooded archive review, open the next player-facing beat, and update memory.md with what changed in the archive."
	aiGMReviewAdvanceStorySeed    = "Starter seed: The archive flood is rising around a ledger vault."
	aiGMReviewAdvanceMemorySeed   = "Remember: Aria is trying to secure the ledger before the room collapses."
	aiGMOOCReplaceScenarioName    = "ai_gm_campaign_context_ooc_replace"
	aiGMOOCReplaceFixtureFile     = "ai_gm_campaign_context_ooc_replace_replay.json"
	aiGMOOCReplacePrompt          = "The OOC pause has closed and players are blocked waiting on you. Replace the interrupted beat with a new player-facing interaction and update memory.md with the new approach."
	aiGMOOCReplaceStorySeed       = "Starter seed: The vault ward can be bypassed from the roof vent after the group regroups."
	aiGMOOCReplaceMemorySeed      = "Remember: the seam is no longer the plan; the group is pivoting to the roof vent."
	aiGMSceneSwitchScenarioName   = "ai_gm_campaign_context_scene_switch"
	aiGMSceneSwitchFixtureFile    = "ai_gm_campaign_context_scene_switch_replay.json"
	aiGMSceneSwitchPrompt         = "Shift focus from the North Gate to the South Tunnel, make the tunnel the active scene, open the next player-facing beat there, and update memory.md with the split-party status."
	aiGMSceneSwitchStorySeed      = "Starter seed: A split party presses on from the gatehouse to the drainage tunnel."
	aiGMSceneSwitchMemorySeed     = "Remember: Aria holds the gate while the next beat should move to the tunnel."
)

type aiGMCampaignScenarioSpec struct {
	Name            string
	FixtureFile     string
	Prompt          string
	StorySeed       string
	MemorySeed      string
	ExtraCharacters []string
	PromptContains  []string
	RequiredToolSet []string
	ForbiddenTools  []string
	MaxToolErrors   *int
	ReferenceLimits *aiGMReferenceLimits
	Prepare         func(t *testing.T, setup *aiGMCampaignScenarioSetup)
	Assert          func(t *testing.T, result aiGMCampaignScenarioResult)
	AssertFixture   func(t *testing.T, fixture openAIReplayFixture)
}

type aiGMReferenceLimits struct {
	MaxSearches int
	MaxReads    int
}

type aiGMCampaignScenarioSetup struct {
	CampaignID         string
	SessionID          string
	CharacterID        string
	OwnerParticipantID string
	AIGMParticipantID  string
	UserCtx            context.Context
	OwnerCtx           context.Context
	AIGMCtx            context.Context

	CampaignClient    gamev1.CampaignServiceClient
	ParticipantClient gamev1.ParticipantServiceClient
	CharacterClient   gamev1.CharacterServiceClient
	SessionClient     gamev1.SessionServiceClient
	SceneClient       gamev1.SceneServiceClient
	InteractionClient gamev1.InteractionServiceClient
	ArtifactClient    aiv1.CampaignArtifactServiceClient
	SnapshotClient    gamev1.SnapshotServiceClient
	DaggerheartClient pb.DaggerheartServiceClient

	ReplayTokens      map[string]string
	ExtraCharacterIDs map[string]string
}

type aiGMCampaignScenarioOptions struct {
	ResponsesURL     string
	Model            string
	ReasoningEffort  string
	CredentialSecret string
	AgentLabel       string
	BeforeRun        func(aiGMCampaignScenarioSetup)
}

type aiGMCampaignScenarioResult struct {
	CampaignID         string
	SessionID          string
	CharacterID        string
	OwnerParticipantID string
	AIGMParticipantID  string
	OutputText         string
	MemoryContent      string
	SkillsReadOnly     bool
	InteractionState   *gamev1.InteractionState
	Scenes             []*gamev1.Scene
	ReplayTokens       map[string]string
}

var (
	aiGMBootstrapScenario = aiGMCampaignScenarioSpec{
		Name:        aiGMBootstrapScenarioName,
		FixtureFile: aiGMBootstrapFixtureFile,
		Prompt:      aiGMBootstrapPrompt,
		StorySeed:   aiGMBootstrapStorySeed,
		MemorySeed:  aiGMBootstrapMemorySeed,
		PromptContains: []string{
			"story.md:",
			aiGMBootstrapStorySeed,
			"memory.md:",
			aiGMBootstrapMemorySeed,
			"Bootstrap mode",
			"Each interaction is an ordered set of beats.",
			"end that interaction with a prompt beat before opening the first player phase",
		},
		RequiredToolSet: []string{
			"system_reference_search",
			"scene_create",
			"interaction_open_scene_player_phase",
		},
		Assert: func(t *testing.T, result aiGMCampaignScenarioResult) {
			t.Helper()
			if strings.TrimSpace(result.OutputText) == "" {
				t.Fatal("expected non-empty model output")
			}
			if strings.TrimSpace(result.MemoryContent) == "" || result.MemoryContent == aiGMBootstrapMemorySeed {
				t.Fatalf("memory.md = %q, expected updated memory content", result.MemoryContent)
			}
			if !result.SkillsReadOnly {
				t.Fatal("expected skills.md to remain read-only")
			}
			if len(result.Scenes) == 0 || strings.TrimSpace(activeSceneID(result.InteractionState)) == "" || !sceneOpenByID(result.Scenes, activeSceneID(result.InteractionState)) {
				t.Fatalf("bootstrap did not leave an active open scene: active_scene_id=%q scenes=%d", activeSceneID(result.InteractionState), len(result.Scenes))
			}
			if !playerPhaseOpen(result.InteractionState) {
				t.Fatal("expected bootstrap to start the first player phase")
			}
		},
	}
	aiGMReviewAdvanceScenario = aiGMCampaignScenarioSpec{
		Name:        aiGMReviewAdvanceScenarioName,
		FixtureFile: aiGMReviewAdvanceFixtureFile,
		Prompt:      aiGMReviewAdvancePrompt,
		StorySeed:   aiGMReviewAdvanceStorySeed,
		MemorySeed:  aiGMReviewAdvanceMemorySeed,
		PromptContains: []string{
			"story.md:",
			aiGMReviewAdvanceStorySeed,
			"memory.md:",
			aiGMReviewAdvanceMemorySeed,
			"Review-resolution mode",
			"end that interaction with a prompt beat and open the next player phase in the same call",
		},
		RequiredToolSet: []string{
			"interaction_resolve_scene_player_review",
		},
		Prepare: prepareReviewAdvanceScenario,
		Assert: func(t *testing.T, result aiGMCampaignScenarioResult) {
			t.Helper()
			if !playerPhaseOpen(result.InteractionState) {
				t.Fatal("expected review advance to reopen a player phase")
			}
			if got := currentPromptBeat(result.InteractionState); strings.TrimSpace(got) == "" {
				t.Fatal("expected current interaction to end with a prompt beat")
			}
		},
	}
	aiGMOOCReplaceScenario = aiGMCampaignScenarioSpec{
		Name:        aiGMOOCReplaceScenarioName,
		FixtureFile: aiGMOOCReplaceFixtureFile,
		Prompt:      aiGMOOCReplacePrompt,
		StorySeed:   aiGMOOCReplaceStorySeed,
		MemorySeed:  aiGMOOCReplaceMemorySeed,
		PromptContains: []string{
			"story.md:",
			aiGMOOCReplaceStorySeed,
			"memory.md:",
			aiGMOOCReplaceMemorySeed,
			"OOC-open mode",
			"use interaction_session_ooc_resolve to close the pause",
		},
		RequiredToolSet: []string{
			"interaction_session_ooc_resolve",
		},
		Prepare: prepareOOCReplaceScenario,
		Assert: func(t *testing.T, result aiGMCampaignScenarioResult) {
			t.Helper()
			if result.InteractionState.GetOoc().GetOpen() {
				t.Fatal("expected OOC to be closed after replacement resolution")
			}
			if result.InteractionState.GetOoc().GetResolutionPending() {
				t.Fatal("expected OOC resolution pending to be cleared")
			}
			if !playerPhaseOpen(result.InteractionState) {
				t.Fatal("expected replacement OOC resolution to open a player phase")
			}
		},
	}
	aiGMSceneSwitchScenario = aiGMCampaignScenarioSpec{
		Name:        aiGMSceneSwitchScenarioName,
		FixtureFile: aiGMSceneSwitchFixtureFile,
		Prompt:      aiGMSceneSwitchPrompt,
		StorySeed:   aiGMSceneSwitchStorySeed,
		MemorySeed:  aiGMSceneSwitchMemorySeed,
		PromptContains: []string{
			"story.md:",
			aiGMSceneSwitchStorySeed,
			"memory.md:",
			aiGMSceneSwitchMemorySeed,
			"Active scene mode",
			"call interaction_open_scene_player_phase with explicit acting character_ids",
		},
		RequiredToolSet: []string{
			"interaction_activate_scene",
			"interaction_open_scene_player_phase",
		},
		Prepare: prepareSceneSwitchScenario,
		Assert: func(t *testing.T, result aiGMCampaignScenarioResult) {
			t.Helper()
			want := strings.TrimSpace(result.ReplayTokens["target_scene_id"])
			if got := activeSceneID(result.InteractionState); got != want {
				t.Fatalf("active_scene_id = %q, want %q", got, want)
			}
			if !playerPhaseOpen(result.InteractionState) {
				t.Fatal("expected switched scene to leave a player phase open")
			}
		},
	}
)

func runAIGMCampaignContextScenario(t *testing.T, spec aiGMCampaignScenarioSpec, opts aiGMCampaignScenarioOptions) aiGMCampaignScenarioResult {
	t.Helper()
	t.Setenv(integrationSharedFixtureEnv, "false")
	aiAddr := pickUnusedAddress(t)
	t.Setenv("FRACTURING_SPACE_AI_ADDR", aiAddr)
	fixture := newSuiteFixture(t)
	userID := fixture.newUserID(t, uniqueTestUsername(t, spec.Name))

	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", fixture.grpcAddr)
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_RESPONSES_URL", strings.TrimSpace(opts.ResponsesURL))
	t.Setenv("FRACTURING_SPACE_AI_DAGGERHEART_REFERENCE_ROOT", daggerheartReferenceRoot)

	aiCtx, cancelAI := context.WithCancel(context.Background())
	aiServer, err := aiapp.New(aiCtx, aiAddr)
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

	gameConn := dialGRPCForIntegration(t, fixture.grpcAddr)
	defer gameConn.Close()
	aiConn := dialGRPCForIntegration(t, aiAddr)
	defer aiConn.Close()
	aiInternalConn := dialGRPCWithServiceID(t, aiAddr, serviceaddr.ServiceAI)
	defer aiInternalConn.Close()

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
		ThemePrompt: spec.StorySeed,
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
	ownerParticipantID := campaignResp.GetOwnerParticipant().GetId()
	ensureSessionStartReadiness(t, ctxWithUser, participantClient, characterClient, campaignID, ownerParticipantID)
	charactersResp, err := characterClient.ListCharacters(ctxWithUser, &gamev1.ListCharactersRequest{
		CampaignId: campaignID,
		PageSize:   20,
	})
	if err != nil {
		t.Fatalf("list characters: %v", err)
	}
	if len(charactersResp.GetCharacters()) == 0 || strings.TrimSpace(charactersResp.GetCharacters()[0].GetId()) == "" {
		t.Fatal("expected at least one campaign character")
	}
	characterID := strings.TrimSpace(charactersResp.GetCharacters()[0].GetId())
	if _, err := artifactClient.EnsureCampaignArtifacts(ctxWithUser, &aiv1.EnsureCampaignArtifactsRequest{
		CampaignId:        campaignID,
		StorySeedMarkdown: spec.StorySeed,
	}); err != nil {
		t.Fatalf("ensure campaign artifacts: %v", err)
	}
	if _, err := artifactClient.UpsertCampaignArtifact(ctxWithUser, &aiv1.UpsertCampaignArtifactRequest{
		CampaignId: campaignID,
		Path:       "memory.md",
		Content:    spec.MemorySeed,
	}); err != nil {
		t.Fatalf("seed memory artifact: %v", err)
	}
	extraCharacterIDs := make(map[string]string, len(spec.ExtraCharacters))
	for _, name := range spec.ExtraCharacters {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		id := createCharacter(t, ctxWithUser, characterClient, campaignID, name)
		ensureDaggerheartCreationReadiness(t, ctxWithUser, characterClient, campaignID, id)
		extraCharacterIDs[name] = id
	}

	startResp, err := sessionClient.StartSession(ctxWithUser, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Opening Night",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	sessionID := startResp.GetSession().GetId()

	setup := aiGMCampaignScenarioSetup{
		CampaignID:         campaignID,
		SessionID:          sessionID,
		CharacterID:        characterID,
		OwnerParticipantID: ownerParticipantID,
		AIGMParticipantID:  aiGMParticipantID,
		UserCtx:            ctxWithUser,
		OwnerCtx:           grpcauthctx.WithParticipantID(context.Background(), ownerParticipantID),
		AIGMCtx:            grpcauthctx.WithParticipantID(context.Background(), aiGMParticipantID),
		CampaignClient:     campaignClient,
		ParticipantClient:  participantClient,
		CharacterClient:    characterClient,
		SessionClient:      sessionClient,
		SceneClient:        sceneClient,
		InteractionClient:  interactionClient,
		ArtifactClient:     artifactClient,
		SnapshotClient:     gamev1.NewSnapshotServiceClient(gameConn),
		DaggerheartClient:  pb.NewDaggerheartServiceClient(gameConn),
		ExtraCharacterIDs:  extraCharacterIDs,
		ReplayTokens: map[string]string{
			"campaign_id":       campaignID,
			"session_id":        sessionID,
			"character_id":      characterID,
			"gm_participant_id": aiGMParticipantID,
		},
	}
	if spec.Prepare != nil {
		spec.Prepare(t, &setup)
	}
	if opts.BeforeRun != nil {
		opts.BeforeRun(setup)
	}

	provider := openaiprovider.NewInvokeAdapter(openaiprovider.InvokeConfig{
		ResponsesURL: strings.TrimSpace(opts.ResponsesURL),
	})
	dialer := gametools.NewDirectDialer(gametools.Clients{
		Interaction: gamev1.NewInteractionServiceClient(gameConn),
		Scene:       sceneClient,
		Campaign:    campaignClient,
		Participant: participantClient,
		Character:   characterClient,
		Session:     sessionClient,
		Snapshot:    gamev1.NewSnapshotServiceClient(gameConn),
		Daggerheart: pb.NewDaggerheartServiceClient(gameConn),
		Artifact:    &grpcArtifactAdapter{client: aiv1.NewCampaignArtifactServiceClient(aiInternalConn)},
		Reference:   &grpcReferenceAdapter{client: aiv1.NewSystemReferenceServiceClient(aiInternalConn)},
	})
	runner := orchestration.NewRunner(orchestration.RunnerConfig{
		Dialer:   dialer,
		MaxSteps: 16,
	})
	runResp, err := runner.Run(context.Background(), orchestration.Input{
		CampaignID:       campaignID,
		SessionID:        sessionID,
		ParticipantID:    aiGMParticipantID,
		Input:            spec.Prompt,
		Model:            strings.TrimSpace(opts.Model),
		ReasoningEffort:  strings.TrimSpace(opts.ReasoningEffort),
		CredentialSecret: strings.TrimSpace(opts.CredentialSecret),
		Provider:         provider,
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
	if active := activeSceneID(interactionResp.GetState()); active != "" && strings.TrimSpace(setup.ReplayTokens["scene_id"]) == "" {
		setup.ReplayTokens["scene_id"] = active
	}

	return aiGMCampaignScenarioResult{
		CampaignID:         campaignID,
		SessionID:          sessionID,
		CharacterID:        characterID,
		OwnerParticipantID: ownerParticipantID,
		AIGMParticipantID:  aiGMParticipantID,
		OutputText:         strings.TrimSpace(runResp.OutputText),
		MemoryContent:      strings.TrimSpace(memoryResp.GetArtifact().GetContent()),
		SkillsReadOnly:     skillsResp.GetArtifact().GetReadOnly(),
		InteractionState:   interactionResp.GetState(),
		Scenes:             scenesResp.GetScenes(),
		ReplayTokens:       mapsClone(setup.ReplayTokens),
	}
}

func prepareReviewAdvanceScenario(t *testing.T, setup *aiGMCampaignScenarioSetup) {
	t.Helper()
	setScenarioGMAuthority(t, setup, setup.AIGMParticipantID)
	sceneID := createScenarioScene(t, setup, "Flooded Archive", "Water rises around the ledger vault.", nil, setup.CharacterID)
	setup.ReplayTokens["scene_id"] = sceneID
	openScenarioPlayerPhase(t, setup, sceneID, "Rising Water", []string{setup.CharacterID},
		aiGMInteractionBeat{Type: gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT, Text: "Aria, what do you do before the ledger is swept away?"},
	)
	submitScenarioPlayerAction(t, setup, sceneID, "Aria tests the current with a hooked pole before moving deeper.", true, setup.CharacterID)
}

func prepareOOCReplaceScenario(t *testing.T, setup *aiGMCampaignScenarioSetup) {
	t.Helper()
	setScenarioGMAuthority(t, setup, setup.AIGMParticipantID)
	sceneID := createScenarioScene(t, setup, "Sealed Vault", "The vault ward surges whenever Aria nears the seam.", nil, setup.CharacterID)
	setup.ReplayTokens["scene_id"] = sceneID
	openScenarioPlayerPhase(t, setup, sceneID, "Ward Study", []string{setup.CharacterID},
		aiGMInteractionBeat{Type: gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT, Text: "Aria, what do you test first about the ward?"},
	)
	openScenarioOOC(t, setup, "Clarify the ward trigger.")
}

func prepareSceneSwitchScenario(t *testing.T, setup *aiGMCampaignScenarioSetup) {
	t.Helper()
	setScenarioGMAuthority(t, setup, setup.AIGMParticipantID)
	sourceSceneID := createScenarioScene(t, setup, "North Gate", "Aria watches the guard rotation above the gate.", nil, setup.CharacterID)
	activate := false
	targetSceneID := createScenarioScene(t, setup, "South Tunnel", "Aria crouches beside the drainage tunnel beneath the keep.", &activate, setup.CharacterID)
	setup.ReplayTokens["source_scene_id"] = sourceSceneID
	setup.ReplayTokens["target_scene_id"] = targetSceneID
}

type aiGMInteractionBeat struct {
	Type gamev1.GMInteractionBeatType
	Text string
}

func createScenarioScene(t *testing.T, setup *aiGMCampaignScenarioSetup, name, description string, activate *bool, characterIDs ...string) string {
	t.Helper()
	activateValue := true
	if activate != nil {
		activateValue = *activate
	}
	req := &gamev1.CreateSceneRequest{
		CampaignId:   setup.CampaignID,
		SessionId:    setup.SessionID,
		Name:         name,
		Description:  description,
		CharacterIds: append([]string(nil), characterIDs...),
		Activate:     &activateValue,
	}
	resp, err := setup.SceneClient.CreateScene(setup.AIGMCtx, req)
	if err != nil {
		t.Fatalf("create scene %q: %v", name, err)
	}
	return strings.TrimSpace(resp.GetSceneId())
}

func setScenarioGMAuthority(t *testing.T, setup *aiGMCampaignScenarioSetup, participantID string) {
	t.Helper()
	if _, err := setup.InteractionClient.SetSessionGMAuthority(setup.UserCtx, &gamev1.SetSessionGMAuthorityRequest{
		CampaignId:    setup.CampaignID,
		ParticipantId: participantID,
	}); err != nil {
		if strings.Contains(err.Error(), "gm authority participant is already set") {
			stateResp, stateErr := setup.InteractionClient.GetInteractionState(setup.UserCtx, &gamev1.GetInteractionStateRequest{
				CampaignId: setup.CampaignID,
			})
			if stateErr == nil && strings.TrimSpace(stateResp.GetState().GetGmAuthorityParticipantId()) == participantID {
				return
			}
		}
		t.Fatalf("set gm authority %q: %v", participantID, err)
	}
}

func openScenarioPlayerPhase(t *testing.T, setup *aiGMCampaignScenarioSetup, sceneID, title string, characterIDs []string, beats ...aiGMInteractionBeat) {
	t.Helper()
	if _, err := setup.InteractionClient.OpenScenePlayerPhase(setup.AIGMCtx, &gamev1.OpenScenePlayerPhaseRequest{
		CampaignId:   setup.CampaignID,
		SceneId:      sceneID,
		CharacterIds: append([]string(nil), characterIDs...),
		Interaction:  scenarioGMInteractionInput(title, characterIDs, beats...),
	}); err != nil {
		t.Fatalf("open scene player phase: %v", err)
	}
}

func submitScenarioPlayerAction(t *testing.T, setup *aiGMCampaignScenarioSetup, sceneID, summary string, yield bool, characterIDs ...string) {
	t.Helper()
	if _, err := setup.InteractionClient.SubmitScenePlayerAction(setup.OwnerCtx, &gamev1.SubmitScenePlayerActionRequest{
		CampaignId:   setup.CampaignID,
		SceneId:      sceneID,
		SummaryText:  summary,
		CharacterIds: append([]string(nil), characterIDs...),
	}); err != nil {
		t.Fatalf("submit scene player action: %v", err)
	}
	if !yield {
		return
	}
	if _, err := setup.InteractionClient.YieldScenePlayerPhase(setup.OwnerCtx, &gamev1.YieldScenePlayerPhaseRequest{
		CampaignId: setup.CampaignID,
		SceneId:    sceneID,
	}); err != nil {
		t.Fatalf("yield scene player phase: %v", err)
	}
}

func waitForGMReviewReady(t *testing.T, setup *aiGMCampaignScenarioSetup, sceneID string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := setup.InteractionClient.GetInteractionState(setup.UserCtx, &gamev1.GetInteractionStateRequest{
			CampaignId: setup.CampaignID,
		})
		if err != nil {
			t.Fatalf("get interaction state while waiting for GM review: %v", err)
		}
		state := resp.GetState()
		if activeSceneID(state) == strings.TrimSpace(sceneID) &&
			state.GetPlayerPhase().GetStatus() == gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW &&
			len(state.GetPlayerPhase().GetSlots()) > 0 &&
			state.GetPlayerPhase().GetSlots()[0].GetYielded() {
			time.Sleep(300 * time.Millisecond)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for GM review readiness on scene %q", sceneID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func extraCharacterID(t *testing.T, setup *aiGMCampaignScenarioSetup, name string) string {
	t.Helper()
	id := strings.TrimSpace(setup.ExtraCharacterIDs[name])
	if id == "" {
		t.Fatalf("missing extra character %q", name)
	}
	return id
}

func openScenarioOOC(t *testing.T, setup *aiGMCampaignScenarioSetup, reason string) {
	t.Helper()
	if _, err := setup.InteractionClient.OpenSessionOOC(setup.OwnerCtx, &gamev1.OpenSessionOOCRequest{
		CampaignId: setup.CampaignID,
		Reason:     reason,
	}); err != nil {
		t.Fatalf("open session ooc: %v", err)
	}
}

func scenarioGMInteractionInput(title string, characterIDs []string, beats ...aiGMInteractionBeat) *gamev1.GMInteractionInput {
	inputBeats := make([]*gamev1.GMInteractionInputBeat, 0, len(beats))
	for idx, beat := range beats {
		inputBeats = append(inputBeats, &gamev1.GMInteractionInputBeat{
			BeatId: beatID(idx),
			Type:   beat.Type,
			Text:   beat.Text,
		})
	}
	return &gamev1.GMInteractionInput{
		Title:        title,
		CharacterIds: append([]string(nil), characterIDs...),
		Beats:        inputBeats,
	}
}

func beatID(idx int) string {
	return "beat-" + strconv.Itoa(idx+1)
}

func mapsClone(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func activeSceneID(state *gamev1.InteractionState) string {
	if state == nil {
		return ""
	}
	return strings.TrimSpace(state.GetActiveScene().GetSceneId())
}

func requireActiveScene(t *testing.T, setup *aiGMCampaignScenarioSetup, wantSceneID string) {
	t.Helper()
	resp, err := setup.InteractionClient.GetInteractionState(setup.UserCtx, &gamev1.GetInteractionStateRequest{
		CampaignId: setup.CampaignID,
	})
	if err != nil {
		t.Fatalf("get interaction state: %v", err)
	}
	if got := activeSceneID(resp.GetState()); got != strings.TrimSpace(wantSceneID) {
		t.Fatalf("active_scene_id = %q, want %q", got, wantSceneID)
	}
}

func requireVisibleAdversaryOnSceneBoard(t *testing.T, setup *aiGMCampaignScenarioSetup, sceneID, adversaryID string) {
	t.Helper()
	resp, err := setup.DaggerheartClient.ListAdversaries(setup.UserCtx, &pb.DaggerheartListAdversariesRequest{
		CampaignId: setup.CampaignID,
		SessionId:  wrapperspb.String(setup.SessionID),
	})
	if err != nil {
		t.Fatalf("list adversaries: %v", err)
	}
	for _, adversary := range resp.GetAdversaries() {
		if strings.TrimSpace(adversary.GetId()) != strings.TrimSpace(adversaryID) {
			continue
		}
		if got := strings.TrimSpace(adversary.GetSceneId()); got != strings.TrimSpace(sceneID) {
			t.Fatalf("adversary %q scene_id = %q, want %q", adversaryID, got, sceneID)
		}
		return
	}
	t.Fatalf("expected adversary %q to be visible on scene board", adversaryID)
}

func requireNoVisibleAdversaryOnSceneBoard(t *testing.T, setup *aiGMCampaignScenarioSetup, sceneID string) {
	t.Helper()
	resp, err := setup.DaggerheartClient.ListAdversaries(setup.UserCtx, &pb.DaggerheartListAdversariesRequest{
		CampaignId: setup.CampaignID,
		SessionId:  wrapperspb.String(setup.SessionID),
	})
	if err != nil {
		t.Fatalf("list adversaries: %v", err)
	}
	for _, adversary := range resp.GetAdversaries() {
		if strings.TrimSpace(adversary.GetSceneId()) == strings.TrimSpace(sceneID) {
			t.Fatalf("expected no visible adversary on scene board, found %q", strings.TrimSpace(adversary.GetId()))
		}
	}
}

func sceneOpenByID(scenes []*gamev1.Scene, sceneID string) bool {
	for _, scene := range scenes {
		if strings.TrimSpace(scene.GetSceneId()) != strings.TrimSpace(sceneID) {
			continue
		}
		return scene.GetOpen()
	}
	return false
}

func playerPhaseOpen(state *gamev1.InteractionState) bool {
	return state != nil && state.GetPlayerPhase().GetStatus() == gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS
}

func currentPromptBeat(state *gamev1.InteractionState) string {
	if state == nil {
		return ""
	}
	for _, beat := range state.GetActiveScene().GetCurrentInteraction().GetBeats() {
		if beat.GetType() == gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT {
			return strings.TrimSpace(beat.GetText())
		}
	}
	return ""
}
