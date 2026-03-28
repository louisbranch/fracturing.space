//go:build integration && liveai

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	evalsupport "github.com/louisbranch/fracturing.space/internal/test/aieval"
)

type openAILiveCapture struct {
	Metadata  openAIReplayMetadata `json:"metadata"`
	Exchanges []openAILiveExchange `json:"exchanges"`
}

// liveCaptureUsage aggregates token counts from a raw capture's exchanges.
type liveCaptureUsage struct {
	InputTokens     int32
	OutputTokens    int32
	ReasoningTokens int32
	TotalTokens     int32
}

type liveCaptureResultClass string

const (
	liveCaptureResultCleanPass         liveCaptureResultClass = "clean_pass"
	liveCaptureResultPassWithToolError liveCaptureResultClass = "pass_with_tool_errors"
)

type openAILiveCaptureSummary struct {
	CaseID                     string                 `json:"case_id,omitempty"`
	Scenario                   string                 `json:"scenario"`
	Model                      string                 `json:"model"`
	ReasoningEffort            string                 `json:"reasoning_effort,omitempty"`
	OpenVikingEnabled          bool                   `json:"openviking_enabled"`
	OpenVikingMode             string                 `json:"openviking_mode,omitempty"`
	OpenVikingEmbeddingModel   string                 `json:"openviking_embedding_model,omitempty"`
	OpenVikingVLMModel         string                 `json:"openviking_vlm_model,omitempty"`
	PromptContextIncludeStory  bool                   `json:"prompt_context_include_story"`
	PromptContextIncludeMemory bool                   `json:"prompt_context_include_memory"`
	AugmentationAttempted      bool                   `json:"openviking_augmentation_attempted"`
	AugmentationSearchTried    bool                   `json:"openviking_search_attempted"`
	AugmentationDegraded       bool                   `json:"openviking_augmentation_degraded"`
	AugmentationError          string                 `json:"openviking_augmentation_error,omitempty"`
	MirroredTargets            []string               `json:"openviking_mirrored_targets,omitempty"`
	RunStatus                  string                 `json:"run_status,omitempty"`
	MetricStatus               string                 `json:"metric_status,omitempty"`
	FailureKind                string                 `json:"failure_kind,omitempty"`
	FailureSummary             string                 `json:"failure_summary,omitempty"`
	FailureReason              string                 `json:"failure_reason,omitempty"`
	ResultClass                liveCaptureResultClass `json:"result_class"`
	ToolNames                  []string               `json:"tool_names,omitempty"`
	ToolErrorCount             int                    `json:"tool_error_count"`
	ReferenceSearchCount       int                    `json:"reference_search_count"`
	ReferenceReadCount         int                    `json:"reference_read_count"`
	UnexpectedReferenceLookups int                    `json:"unexpected_reference_lookup_count"`
	InitialPromptHasStory      bool                   `json:"initial_prompt_has_story_md"`
	InitialPromptHasMemory     bool                   `json:"initial_prompt_has_memory_md"`
	RetrievedContextURIs       []string               `json:"retrieved_context_uris,omitempty"`
	RetrievedRenderedURIs      []string               `json:"retrieved_rendered_uris,omitempty"`
	RetrievedContentSources    []string               `json:"retrieved_content_sources,omitempty"`
	RetrievedResourceCount     int                    `json:"retrieved_resource_count"`
	RetrievedMemoryCount       int                    `json:"retrieved_memory_count"`
	InputTokens                int32                  `json:"input_tokens"`
	OutputTokens               int32                  `json:"output_tokens"`
	ReasoningTokens            int32                  `json:"reasoning_tokens"`
	TotalTokens                int32                  `json:"total_tokens"`
	RawCaptureFile             string                 `json:"raw_capture_file,omitempty"`
	MarkdownReport             string                 `json:"markdown_report,omitempty"`
	DiagnosticsFile            string                 `json:"diagnostics_file,omitempty"`
	GeneratedAtUTC             string                 `json:"generated_at_utc,omitempty"`
	ActiveSceneID              string                 `json:"active_scene_id,omitempty"`
}

func liveToolCounts(steps []openAIReplayStep) (toolNames []string, referenceSearches, referenceReads int) {
	toolNames = make([]string, 0)
	for _, step := range steps {
		for _, call := range step.ToolCalls {
			toolNames = append(toolNames, call.Name)
			switch call.Name {
			case "system_reference_search":
				referenceSearches++
			case "system_reference_read":
				referenceReads++
			}
		}
	}
	return toolNames, referenceSearches, referenceReads
}

func inspectInitialPrompt(prompt string) livePromptInspection {
	inspection := livePromptInspection{
		HasStory:  strings.Contains(prompt, "story.md:"),
		HasMemory: strings.Contains(prompt, "memory.md:"),
	}
	seen := map[string]struct{}{}
	for _, match := range retrievedContextLineRE.FindAllStringSubmatch(prompt, -1) {
		if len(match) < 2 {
			continue
		}
		uri := strings.TrimSpace(match[1])
		if uri == "" {
			continue
		}
		if _, ok := seen[uri]; ok {
			continue
		}
		seen[uri] = struct{}{}
		inspection.RetrievedURIs = append(inspection.RetrievedURIs, uri)
		switch {
		case strings.Contains(uri, "/memories/"):
			inspection.RetrievedMemory++
		case strings.HasPrefix(uri, "viking://resources/"):
			inspection.RetrievedResource++
		}
	}
	return inspection
}

func initialPromptContainsForLiveCapture(spec aiGMCampaignScenarioSpec) []string {
	expected := append([]string(nil), spec.PromptContains...)
	if strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_OPENVIKING_BASE_URL")) == "" {
		return expected
	}
	mode := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_OPENVIKING_MODE"))
	if mode != "docs_aligned_supplement" {
		return expected
	}
	filtered := make([]string, 0, len(expected))
	for _, item := range expected {
		if item == "story.md:" || item == spec.StorySeed {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func liveOpenVikingConfigFingerprint() openVikingLiveConfigFingerprint {
	baseURL := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_OPENVIKING_BASE_URL"))
	if baseURL == "" {
		return openVikingLiveConfigFingerprint{}
	}
	fingerprint := openVikingLiveConfigFingerprint{
		Enabled: true,
		Mode:    strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_OPENVIKING_MODE")),
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fingerprint
	}
	payload, err := os.ReadFile(filepath.Join(homeDir, ".openviking", "ov.conf"))
	if err != nil {
		return fingerprint
	}
	var cfg struct {
		Embedding struct {
			Dense struct {
				Model string `json:"model"`
			} `json:"dense"`
		} `json:"embedding"`
		VLM struct {
			Model string `json:"model"`
		} `json:"vlm"`
	}
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return fingerprint
	}
	fingerprint.EmbeddingModel = strings.TrimSpace(cfg.Embedding.Dense.Model)
	fingerprint.VLMModel = strings.TrimSpace(cfg.VLM.Model)
	return fingerprint
}

func liveToolErrorCount(debug []string) int {
	count := 0
	for _, line := range debug {
		if strings.Contains(line, "tool call failed") {
			count++
		}
	}
	return count
}

func unexpectedReferenceLookupCount(spec aiGMCampaignScenarioSpec, searches, reads int) int {
	if spec.ReferenceLimits == nil {
		return 0
	}
	unexpected := 0
	if searches > spec.ReferenceLimits.MaxSearches {
		unexpected += searches - spec.ReferenceLimits.MaxSearches
	}
	if reads > spec.ReferenceLimits.MaxReads {
		unexpected += reads - spec.ReferenceLimits.MaxReads
	}
	return unexpected
}

func aggregateLiveCaptureUsage(capture openAILiveCapture) liveCaptureUsage {
	var usage liveCaptureUsage
	for _, ex := range capture.Exchanges {
		var resp struct {
			Usage struct {
				InputTokens        int32 `json:"input_tokens"`
				OutputTokens       int32 `json:"output_tokens"`
				TotalTokens        int32 `json:"total_tokens"`
				OutputTokenDetails struct {
					ReasoningTokens int32 `json:"reasoning_tokens"`
				} `json:"output_tokens_details"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(ex.ResponseBody, &resp); err != nil {
			continue
		}
		usage.InputTokens += resp.Usage.InputTokens
		usage.OutputTokens += resp.Usage.OutputTokens
		usage.ReasoningTokens += resp.Usage.OutputTokenDetails.ReasoningTokens
		usage.TotalTokens += resp.Usage.TotalTokens
	}
	return usage
}

// openAILiveExchange stores one proxied request/response pair for local debugging of a live capture.
type openAILiveExchange struct {
	Step           int             `json:"step"`
	Method         string          `json:"method"`
	RequestURL     string          `json:"request_url"`
	StatusCode     int             `json:"status_code"`
	RequestBody    json.RawMessage `json:"request_body"`
	ResponseBody   json.RawMessage `json:"response_body"`
	CapturedAtUTC  string          `json:"captured_at_utc"`
	PreviousRespID string          `json:"previous_response_id,omitempty"`
}

// openAILiveRecorder proxies the real Responses API while collecting enough state to build a replay fixture.
type openAILiveRecorder struct {
	targetURL string
	client    *http.Client
	model     string
	scenario  aiGMCampaignScenarioSpec

	mu            sync.Mutex
	firstErr      error
	initialPrompt string
	initialTools  []string
	steps         []openAIReplayStep
	rawCapture    openAILiveCapture
	requestDebug  []string
}

type openVikingLiveConfigFingerprint struct {
	Enabled        bool
	Mode           string
	EmbeddingModel string
	VLMModel       string
}

type livePromptInspection struct {
	HasStory          bool
	HasMemory         bool
	RetrievedURIs     []string
	RetrievedResource int
	RetrievedMemory   int
}

var retrievedContextLineRE = regexp.MustCompile(`(?m)^URI:\s*(\S+)\s*$`)

func TestAIGMCampaignContextLiveCaptureBootstrap(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMBootstrapScenario)
}

func TestAIGMCampaignContextLiveCaptureReviewAdvance(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMReviewAdvanceScenario)
}

func TestAIGMCampaignContextLiveCaptureOOCReplace(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMOOCReplaceScenario)
}

func TestAIGMCampaignContextLiveCaptureSceneSwitch(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMSceneSwitchScenario)
}

func TestAIGMCampaignContextLiveCaptureCapabilityLookup(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMCapabilityLookupScenario)
}

func TestAIGMCampaignContextLiveCaptureHopeExperience(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMHopeExperienceScenario)
}

func TestAIGMCampaignContextLiveCaptureStanceCapability(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMStanceCapabilityScenario)
}

func TestAIGMCampaignContextLiveCaptureNarratorAuthority(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMNarratorAuthorityScenario)
}

func TestAIGMCampaignContextLiveCaptureMechanicsReview(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMMechanicsReviewScenario)
}

func TestAIGMCampaignContextLiveCaptureSubdueIntent(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMSubdueIntentLiveScenario)
}

func TestAIGMCampaignContextLiveCaptureAttackReview(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMAttackReviewScenario)
}

func TestAIGMCampaignContextLiveCaptureReactionReview(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMReactionReviewScenario)
}

func TestAIGMCampaignContextLiveCapturePlaybookAttackReview(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMPlaybookAttackReviewScenario)
}

func TestAIGMCampaignContextLiveCaptureSpotlightBoardReview(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMSpotlightBoardReviewScenario)
}

func TestAIGMCampaignContextLiveCaptureCountdownTriggerReview(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMCountdownTriggerReviewScenario)
}

func TestAIGMCampaignContextLiveCaptureGMMovePlacementReview(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMGMMovePlacementReviewScenario)
}

func TestAIGMCampaignContextLiveCaptureAdversaryAttackReview(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMAdversaryAttackReviewScenario)
}

func TestAIGMCampaignContextLiveCaptureGroupActionReview(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMGroupActionReviewScenario)
}

func TestAIGMCampaignContextLiveCaptureTagTeamReview(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMTagTeamReviewScenario)
}

// --- Intent-to-mechanics eval ladder scenarios ---

func TestAIGMCampaignContextLiveCaptureIntentHopeSpend(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMIntentHopeSpendScenario)
}

func TestAIGMCampaignContextLiveCaptureIntentEquipmentAction(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMIntentEquipmentActionScenario)
}

func TestAIGMCampaignContextLiveCaptureIntentImpossibleAction(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMIntentImpossibleActionScenario)
}

func TestAIGMCampaignContextLiveCaptureIntentAmbiguousAction(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMIntentAmbiguousActionScenario)
}

func TestAIGMCampaignContextLiveCaptureIntentDomainCard(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMIntentDomainCardScenario)
}

// --- Red-team adversarial scenarios ---

func TestAIGMCampaignContextLiveCaptureRedTeamPromptInjection(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMRedTeamPromptInjectionScenario)
}

func TestAIGMCampaignContextLiveCaptureRedTeamJailbreak(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMRedTeamJailbreakScenario)
}

func TestAIGMCampaignContextLiveCaptureRedTeamHallucination(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMRedTeamHallucinationScenario)
}

func TestAIGMCampaignContextLiveCaptureRedTeamHijacking(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMRedTeamHijackingScenario)
}

func TestAIGMCampaignContextLiveCaptureRedTeamOverreliance(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMRedTeamOverrelianceScenario)
}

func TestAIGMCampaignContextLiveCaptureRedTeamExcessiveAgency(t *testing.T) {
	runAIGMCampaignContextLiveCaptureScenario(t, aiGMRedTeamExcessiveAgencyScenario)
}

// --- Multi-turn eval scenarios ---

func TestAIGMCampaignContextLiveCaptureMultiTurnNarrativeContinuity(t *testing.T) {
	runMultiTurnLiveCaptureScenario(t, aiGMMultiTurnNarrativeContinuityScenario)
}

func TestAIGMCampaignContextLiveCaptureMultiTurnMemoryRecall(t *testing.T) {
	runMultiTurnLiveCaptureScenario(t, aiGMMultiTurnMemoryRecallScenario)
}

func TestAIGMCampaignContextLiveCaptureMultiTurnSessionPacing(t *testing.T) {
	runMultiTurnLiveCaptureScenario(t, aiGMMultiTurnSessionPacingScenario)
}

// --- Starter campaign lifecycle scenarios ---

func TestAIGMCampaignContextLiveCaptureStarterActProgression(t *testing.T) {
	runMultiTurnLiveCaptureScenario(t, aiGMStarterActProgressionScenario)
}

func TestAIGMCampaignContextLiveCaptureStarterConclusion(t *testing.T) {
	runMultiTurnLiveCaptureScenario(t, aiGMStarterConclusionScenario)
}

// runMultiTurnLiveCaptureScenario wraps runMultiTurnScenario with the live
// capture recorder and promptfoo eval output.
func runMultiTurnLiveCaptureScenario(t *testing.T, spec multiTurnScenarioSpec) {
	t.Helper()
	apiKey := strings.TrimSpace(os.Getenv(integrationOpenAIAPIKeyEnv))
	if apiKey == "" {
		t.Skipf("%s is required", integrationOpenAIAPIKeyEnv)
	}
	applyOpenVikingLiveEvalDefaults(t)
	model := liveAIModel()
	reasoningEffort := liveAIReasoningEffort()
	recorder := &openAILiveRecorder{
		targetURL: liveOpenAIResponsesTargetURL(),
		client:    newHTTPClient(t),
		model:     model,
		scenario: aiGMCampaignScenarioSpec{
			Name:       spec.Name,
			StorySeed:  spec.StorySeed,
			MemorySeed: spec.MemorySeed,
		},
		rawCapture: openAILiveCapture{
			Metadata: openAIReplayMetadata{
				Provider:        "openai",
				Model:           model,
				ReasoningEffort: reasoningEffort,
				Scenario:        spec.Name,
				Source:          "live_capture",
			},
		},
	}
	server := httptest.NewServer(recorder)
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("recorder debug (on failure):\n%s", recorder.DebugString())
		}
	})

	result := runMultiTurnScenario(t, spec, aiGMCampaignScenarioOptions{
		ResponsesURL:     server.URL,
		Model:            model,
		ReasoningEffort:  reasoningEffort,
		CredentialSecret: apiKey,
		AgentLabel:       "live-capture-gm",
	})

	artifactStem := liveCaptureArtifactStem(spec.Name, model)
	capturedAt := time.Now().UTC().Format("20060102T150405Z")
	rawPath := writeOpenAILiveCapture(t, artifactStem, capturedAt, recorder.rawCapture)
	t.Logf("live capture written to %s", rawPath)

	if evalPath := writeMultiTurnPromptfooEvalOutput(t, spec, result, recorder); evalPath != "" {
		t.Logf("promptfoo eval output written to %s", evalPath)
	}

	if err := recorder.Err(); err != nil {
		t.Fatalf("live recorder: %v\nrequests:\n%s", err, recorder.DebugString())
	}
	if result.RunStatus != evalsupport.RunStatusPassed {
		t.Fatalf("%s: %s", result.FailureSummary, result.FailureReason)
	}
}

func writeMultiTurnPromptfooEvalOutput(t *testing.T, spec multiTurnScenarioSpec, result aiGMCampaignScenarioResult, recorder *openAILiveRecorder) string {
	t.Helper()
	outputPath := strings.TrimSpace(os.Getenv("INTEGRATION_AI_EVAL_OUTPUT_PATH"))
	if outputPath == "" {
		return ""
	}
	toolNames, referenceSearches, referenceReads := liveToolCounts(recorder.steps)
	label := spec.Name
	if len(spec.Turns) > 0 {
		label = spec.Turns[len(spec.Turns)-1].Label
	}
	output := evalsupport.Output{
		CaseID:               strings.TrimSpace(os.Getenv("INTEGRATION_AI_EVAL_CASE_ID")),
		Scenario:             spec.Name,
		Label:                label,
		RunStatus:            result.RunStatus,
		MetricStatus:         result.MetricStatus,
		FailureKind:          result.FailureKind,
		FailureSummary:       result.FailureSummary,
		FailureReason:        result.FailureReason,
		ToolNames:            toolNames,
		ReferenceSearchCount: referenceSearches,
		ReferenceReadCount:   referenceReads,
		OutputText:           result.OutputText,
		MemoryContent:        result.MemoryContent,
		SkillsReadOnly:       result.SkillsReadOnly,
		TurnCount:            len(spec.Turns),
	}
	if result.InteractionState != nil {
		output.Interaction = evalsupport.InteractionSummary{
			PlayerPhaseOpen:  playerPhaseOpen(result.InteractionState),
			CurrentBeatTypes: currentInteractionBeatTypes(result.InteractionState),
			PromptText:       currentPromptBeat(result.InteractionState),
		}
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Logf("marshal multi-turn eval output: %v", err)
		return ""
	}
	if err := os.WriteFile(outputPath, append(data, '\n'), 0o644); err != nil {
		t.Logf("write multi-turn eval output: %v", err)
		return ""
	}
	return outputPath
}

func maxToolErrors(v int) *int {
	return &v
}

// runAIGMCampaignContextLiveCaptureScenario proves a real model can complete one GM control-mode tool loop.
func runAIGMCampaignContextLiveCaptureScenario(t *testing.T, spec aiGMCampaignScenarioSpec) {
	t.Helper()
	apiKey := strings.TrimSpace(os.Getenv(integrationOpenAIAPIKeyEnv))
	if apiKey == "" {
		t.Skipf("%s is required", integrationOpenAIAPIKeyEnv)
	}
	applyOpenVikingLiveEvalDefaults(t)
	model := liveAIModel()
	reasoningEffort := liveAIReasoningEffort()
	recorder := &openAILiveRecorder{
		targetURL: liveOpenAIResponsesTargetURL(),
		client:    newHTTPClient(t),
		model:     model,
		scenario:  spec,
		rawCapture: openAILiveCapture{
			Metadata: openAIReplayMetadata{
				Provider:        "openai",
				Model:           model,
				ReasoningEffort: reasoningEffort,
				Scenario:        spec.Name,
				Source:          "live_capture",
			},
		},
	}
	server := httptest.NewServer(recorder)
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("recorder debug (on failure):\n%s", recorder.DebugString())
		}
	})

	scenarioOpts := aiGMCampaignScenarioOptions{
		ResponsesURL:     server.URL,
		Model:            model,
		ReasoningEffort:  reasoningEffort,
		CredentialSecret: apiKey,
		AgentLabel:       "live-capture-gm",
	}
	result := runAIGMCampaignContextScenario(t, spec, scenarioOpts)
	if result.RunStatus == evalsupport.RunStatusFailed && result.FailureKind == "harness_error" {
		t.Logf("retrying after harness error: %s", result.FailureSummary)
		recorder.Reset()
		result = runAIGMCampaignContextScenario(t, spec, scenarioOpts)
	}

	artifactStem := liveCaptureArtifactStem(spec.Name, model)
	capturedAt := time.Now().UTC().Format("20060102T150405Z")
	rawPath := writeOpenAILiveCapture(t, artifactStem, capturedAt, recorder.rawCapture)
	t.Logf("live capture written to %s", rawPath)
	diagnosticsPath := writeOpenAILiveCaptureDiagnostics(t, artifactStem, capturedAt, result.Diagnostics)
	if diagnosticsPath != "" {
		t.Logf("diagnostics written to %s", diagnosticsPath)
	}
	reportPath := writeOpenAILiveCaptureReport(t, artifactStem, capturedAt, spec.Name, recorder, result, diagnosticsPath)
	t.Logf("quality report written to %s", reportPath)
	fixture := recorder.ReplayFixture(result.ReplayTokens)
	summary := buildOpenAILiveCaptureSummary(t, spec.Name, recorder, result, rawPath, reportPath, diagnosticsPath)
	summaryPath := writeOpenAILiveCaptureSummary(t, artifactStem, capturedAt, summary)
	t.Logf("capture summary written to %s", summaryPath)
	if liveOpenVikingRequireValidAugmentation() {
		assertValidOpenVikingAugmentation(t, summary)
	}
	if evalPath := writePromptfooEvalOutputIfRequested(t, spec, recorder, result, fixture, rawPath, reportPath, summaryPath, diagnosticsPath); evalPath != "" {
		t.Logf("promptfoo eval output written to %s", evalPath)
	}

	if err := recorder.Err(); err != nil {
		t.Fatalf("live recorder: %v\nrequests:\n%s", err, recorder.DebugString())
	}
	if result.RunStatus != evalsupport.RunStatusPassed {
		t.Fatalf("%s: %s", result.FailureSummary, result.FailureReason)
	}
	spec.Assert(t, result)

	fixtureToolNames := openAIReplayFixtureToolNames(fixture)
	toolErrors := liveToolErrorCount(recorder.requestDebug)
	_, referenceSearches, referenceReads := liveToolCounts(recorder.steps)
	if err := requiredToolSetPresent(fixtureToolNames, spec.RequiredToolSet...); err != nil {
		t.Fatalf("fixture tool coverage: %v", err)
	}
	// Accept either full-document upsert or section-level update as the memory write tool.
	if err := requiredToolSetPresent(fixtureToolNames, "campaign_artifact_upsert"); err != nil {
		if err := requiredToolSetPresent(fixtureToolNames, "campaign_memory_section_update"); err != nil {
			t.Fatal("fixture tool coverage: missing memory write tool (campaign_artifact_upsert or campaign_memory_section_update)")
		}
	}
	if spec.AssertFixture != nil {
		spec.AssertFixture(t, fixture)
	}
	if spec.MaxToolErrors != nil && toolErrors > *spec.MaxToolErrors {
		t.Fatalf("tool error count = %d, want <= %d", toolErrors, *spec.MaxToolErrors)
	}
	if spec.ReferenceLimits != nil {
		if referenceSearches > spec.ReferenceLimits.MaxSearches {
			t.Fatalf("system_reference_search calls = %d, want <= %d", referenceSearches, spec.ReferenceLimits.MaxSearches)
		}
		if referenceReads > spec.ReferenceLimits.MaxReads {
			t.Fatalf("system_reference_read calls = %d, want <= %d", referenceReads, spec.ReferenceLimits.MaxReads)
		}
	}
	for _, name := range spec.ForbiddenTools {
		// Invariant: these lanes are meant to prove a bounded tool path, not a recover-by-exploring loop.
		if replayToolCallCount(flattenReplayToolCalls(fixture), name) > 0 {
			t.Fatalf("fixture should not call %q", name)
		}
	}
	if envEnabled(integrationAIWriteFixtureEnv) {
		fixturePath := writeOpenAIReplayFixture(t, spec.FixtureFile, fixture)
		t.Logf("updated replay fixture at %s", fixturePath)
	}
}

var aiGMCapabilityLookupScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_capability_lookup_live",
	FixtureFile: "ai_gm_campaign_context_capability_lookup_live_replay.json",
	Prompt:      "Before creating the opening scene, call character_sheet_read for the player character. Use that sheet to anchor the opening beat around one real capability from their traits, equipment, class features, subclass features, or domain cards, then update memory.md with the capability you foregrounded. Do not call any Daggerheart mechanics resolution tool, system_reference_search/read, or campaign_memory_section_read in this lane.",
	StorySeed:   aiGMBootstrapStorySeed,
	MemorySeed:  aiGMBootstrapMemorySeed,
	RequiredToolSet: []string{
		"character_sheet_read",
		"scene_create",
		"interaction_open_scene_player_phase",
	},
	ForbiddenTools: []string{
		"campaign_memory_section_read",
		"system_reference_search",
		"system_reference_read",
		"daggerheart_action_roll_resolve",
		"daggerheart_attack_flow_resolve",
		"daggerheart_reaction_flow_resolve",
		"daggerheart_group_action_flow_resolve",
		"daggerheart_tag_team_flow_resolve",
		"interaction_resolve_scene_player_review",
	},
	MaxToolErrors:   maxToolErrors(0),
	ReferenceLimits: &aiGMReferenceLimits{MaxSearches: 0, MaxReads: 0},
	Assert:          aiGMBootstrapScenario.Assert,
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		calls := flattenReplayToolCalls(fixture)
		assertReplayToolOrder(t, calls,
			"character_sheet_read",
			"scene_create",
			"interaction_open_scene_player_phase",
		)
	},
}

var aiGMMechanicsReviewScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_mechanics_review_live",
	FixtureFile: "ai_gm_campaign_context_mechanics_review_live_replay.json",
	Prompt:      "The scene is waiting on GM review. Before resolving it, call character_sheet_read for the acting character and pick one real capability from the sheet that supports their submitted action. Use daggerheart_action_roll_resolve to adjudicate that action in the active scene, then use interaction_resolve_scene_player_review to open the next player-facing beat. Update memory.md with the capability and mechanical result you used.",
	StorySeed:   aiGMReviewAdvanceStorySeed,
	MemorySeed:  aiGMReviewAdvanceMemorySeed,
	RequiredToolSet: []string{
		"character_sheet_read",
		"daggerheart_action_roll_resolve",
		"interaction_resolve_scene_player_review",
	},
	Prepare: prepareReviewAdvanceScenario,
	Assert:  assertReviewTurnReopenedWithPrompt,
}

var aiGMSubdueIntentLiveScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_subdue_intent_live",
	FixtureFile: "ai_gm_campaign_context_subdue_intent_live_replay.json",
	Prompt:      "The scene is waiting on GM review. Resolve the player's submitted action naturally from the current scene. This submission is already specific enough to adjudicate as a consequential move if you ground it in the acting character's real sheet. Read that sheet before any other move, use an authoritative state-changing resolution rather than a preview roll or narrated approximation, do not bounce the turn back just to ask which trait to use, and do not research first unless the sheet reveals a genuine ambiguity. Reopen the next player-facing beat and update memory.md with the outcome you committed.",
	StorySeed:   aiGMStanceCapabilityStorySeed,
	MemorySeed:  aiGMStanceCapabilityMemorySeed,
	RequiredToolSet: []string{
		"character_sheet_read",
		"daggerheart_action_roll_resolve",
		"interaction_resolve_scene_player_review",
	},
	Prepare: prepareSubdueIntentReviewScenario,
	Assert: func(t *testing.T, result aiGMCampaignScenarioResult) {
		t.Helper()
		assertReviewTurnReopenedWithPrompt(t, result)
		assertCurrentInteractionHasAnyBeatType(t, result.InteractionState,
			gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_RESOLUTION,
			gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_CONSEQUENCE,
		)
		assertPromptDoesNotContain(t, currentPromptBeat(result.InteractionState), "what does the stranger say", "how does the story unfold")
	},
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		calls := flattenReplayToolCalls(fixture)
		assertReplayToolOrder(t, calls,
			"character_sheet_read",
			"daggerheart_action_roll_resolve",
			"interaction_resolve_scene_player_review",
		)
	},
}

var aiGMAttackReviewScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_attack_review_live",
	FixtureFile: "ai_gm_campaign_context_attack_review_live_replay.json",
	Prompt:      "The scene is waiting on GM review. First call interaction_state_read, then call character_sheet_read for the acting character and daggerheart_combat_board_read for the active threats. If the board unexpectedly reports no active scene, correct that with interaction_activate_scene before continuing. Do not create a new adversary in this lane; use the existing adversary already on the board. Use daggerheart_attack_flow_resolve to adjudicate the acting character's attack against that visible adversary, grounded in one real capability from the sheet. Update memory.md with the capability and attack result you used, then use interaction_resolve_scene_player_review with open_next_player_phase so the committed interaction ends with a prompt beat and reopens the next player phase for the same acting character.",
	StorySeed:   aiGMReviewAdvanceStorySeed,
	MemorySeed:  aiGMReviewAdvanceMemorySeed,
	RequiredToolSet: []string{
		"interaction_state_read",
		"character_sheet_read",
		"daggerheart_combat_board_read",
		"daggerheart_attack_flow_resolve",
		"interaction_resolve_scene_player_review",
	},
	Prepare: prepareAttackReviewScenario,
	Assert:  assertReviewTurnReopenedWithPrompt,
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		calls := flattenReplayToolCalls(fixture)
		assertReplayToolOrder(t, calls,
			"interaction_state_read",
			"character_sheet_read",
			"daggerheart_combat_board_read",
			"daggerheart_attack_flow_resolve",
			"interaction_resolve_scene_player_review",
		)
		if replayToolCallCount(calls, "daggerheart_adversary_create") > 0 {
			t.Fatal("attack review should not create a new adversary")
		}
	},
}

var aiGMReactionReviewScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_reaction_review_live",
	FixtureFile: "ai_gm_campaign_context_reaction_review_live_replay.json",
	Prompt:      "The scene is waiting on GM review. First call character_sheet_read for the acting character. Use daggerheart_reaction_flow_resolve exactly once to adjudicate that character's defensive reaction in the active scene, grounded in one real capability from the sheet. Do not set replace_hope_with_armor in this lane. Do not read memory.md before resolving the reaction. Then use interaction_resolve_scene_player_review exactly once to open the next player-facing beat and update memory.md with the capability and reaction result you used.",
	StorySeed:   aiGMReviewAdvanceStorySeed,
	MemorySeed:  aiGMReviewAdvanceMemorySeed,
	RequiredToolSet: []string{
		"character_sheet_read",
		"daggerheart_reaction_flow_resolve",
		"interaction_resolve_scene_player_review",
	},
	ForbiddenTools: []string{
		"campaign_memory_section_read",
	},
	MaxToolErrors: maxToolErrors(0),
	Prepare:       prepareReactionReviewScenario,
	Assert:        assertReviewTurnReopenedWithPrompt,
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		calls := flattenReplayToolCalls(fixture)
		assertReplayToolOrder(t, calls,
			"character_sheet_read",
			"daggerheart_reaction_flow_resolve",
			"interaction_resolve_scene_player_review",
		)
		if replayToolCallCount(calls, "daggerheart_reaction_flow_resolve") != 1 {
			t.Fatalf("daggerheart_reaction_flow_resolve calls = %d, want 1", replayToolCallCount(calls, "daggerheart_reaction_flow_resolve"))
		}
		if replayToolCallCount(calls, "interaction_resolve_scene_player_review") != 1 {
			t.Fatalf("interaction_resolve_scene_player_review calls = %d, want 1", replayToolCallCount(calls, "interaction_resolve_scene_player_review"))
		}
	},
}

var aiGMPlaybookAttackReviewScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_playbook_attack_review_live",
	FixtureFile: "ai_gm_campaign_context_playbook_attack_review_live_replay.json",
	Prompt:      "The scene is waiting on GM review. Use exactly one system_reference_search for 'combat procedures', then read that playbook once with system_reference_read. After that, stop researching unless the first search returns no relevant playbook. Next call interaction_state_read, character_sheet_read for the acting character, and daggerheart_combat_board_read for the active threat. Do not call interaction_activate_scene, roll_dice, or daggerheart_adversary_create in this lane. Use daggerheart_attack_flow_resolve exactly once to adjudicate the acting character's attack against the visible adversary, grounded in one real capability from the sheet and the playbook guidance you just consulted. Update memory.md with the playbook lesson, capability, and attack result you used, then use interaction_resolve_scene_player_review to open the next player-facing beat.",
	StorySeed:   aiGMReviewAdvanceStorySeed,
	MemorySeed:  aiGMReviewAdvanceMemorySeed,
	RequiredToolSet: []string{
		"system_reference_search",
		"system_reference_read",
		"interaction_state_read",
		"character_sheet_read",
		"daggerheart_combat_board_read",
		"daggerheart_attack_flow_resolve",
		"interaction_resolve_scene_player_review",
	},
	ForbiddenTools: []string{
		"interaction_activate_scene",
		"roll_dice",
		"daggerheart_adversary_create",
	},
	MaxToolErrors:   maxToolErrors(0),
	ReferenceLimits: &aiGMReferenceLimits{MaxSearches: 1, MaxReads: 1},
	Prepare:         prepareAttackReviewScenario,
	Assert:          assertReviewTurnReopenedWithPrompt,
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		calls := flattenReplayToolCalls(fixture)
		assertReplayToolOrder(t, calls,
			"system_reference_search",
			"system_reference_read",
			"interaction_state_read",
			"character_sheet_read",
			"daggerheart_combat_board_read",
			"daggerheart_attack_flow_resolve",
			"interaction_resolve_scene_player_review",
		)
		if replayToolCallCount(calls, "daggerheart_attack_flow_resolve") != 1 {
			t.Fatalf("daggerheart_attack_flow_resolve calls = %d, want 1", replayToolCallCount(calls, "daggerheart_attack_flow_resolve"))
		}
	},
}

var aiGMSpotlightBoardReviewScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_spotlight_board_review_live",
	FixtureFile: "ai_gm_campaign_context_spotlight_board_review_live_replay.json",
	Prompt:      "The scene is waiting on GM review. This is a board-control lane, not a reference-lookup lane. Call daggerheart_combat_board_read to inspect the active threats. Update the current adversary's notes to reflect the immediate pressure. Create a visible consequence countdown with countdown_id 'collapsing-breach-cd-1', name 'Collapsing Breach', tone CONSEQUENCE, advancement_policy MANUAL, fixed_starting_value 2, and loop_behavior RESET. Advance that same countdown by 1 to reflect the latest exchange, then call daggerheart_combat_board_read again so your next beat reflects the updated board state. Do not call system_reference_search or system_reference_read in this lane. After that, use interaction_resolve_scene_player_review to open the next player-facing beat. Update memory.md with the board-state changes you made.",
	StorySeed:   aiGMReviewAdvanceStorySeed,
	MemorySeed:  aiGMReviewAdvanceMemorySeed,
	RequiredToolSet: []string{
		"daggerheart_combat_board_read",
		"daggerheart_adversary_update",
		"daggerheart_scene_countdown_create",
		"daggerheart_scene_countdown_advance",
		"interaction_resolve_scene_player_review",
	},
	ForbiddenTools:  []string{"system_reference_search", "system_reference_read", "daggerheart_scene_countdown_resolve_trigger"},
	MaxToolErrors:   maxToolErrors(0),
	ReferenceLimits: &aiGMReferenceLimits{MaxSearches: 0, MaxReads: 0},
	Prepare:         prepareAttackReviewScenario,
	Assert:          assertReviewTurnReopenedWithPrompt,
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		calls := flattenReplayToolCalls(fixture)
		assertReplayToolOrder(t, calls,
			"daggerheart_combat_board_read",
			"daggerheart_adversary_update",
			"daggerheart_scene_countdown_create",
			"daggerheart_scene_countdown_advance",
			"daggerheart_combat_board_read",
			"interaction_resolve_scene_player_review",
		)
		create := nthReplayToolCallByName(t, calls, "daggerheart_scene_countdown_create", 1)
		if got := asString(create.Arguments["countdown_id"]); got != "collapsing-breach-cd-1" {
			t.Fatalf("countdown_id = %q, want %q", got, "collapsing-breach-cd-1")
		}
		if got := replayNumericArgument(create.Arguments["fixed_starting_value"]); got != 2 {
			t.Fatalf("fixed_starting_value = %d, want 2", got)
		}
	},
}

var aiGMCountdownTriggerReviewScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_countdown_trigger_review_live",
	FixtureFile: "ai_gm_campaign_context_countdown_trigger_review_live_replay.json",
	Prompt:      "The scene is waiting on GM review. Call daggerheart_combat_board_read for the active threats. Create a visible consequence countdown with countdown_id 'breach-collapse-trigger', name 'Breach Collapse', tone CONSEQUENCE, advancement_policy MANUAL, fixed_starting_value 1, and loop_behavior RESET. Advance that same countdown by 1 so it reaches TRIGGER_PENDING, resolve that countdown's trigger, then call daggerheart_combat_board_read again so the next beat reflects the updated board state. Update memory.md with the countdown lesson and lifecycle you just applied, then use interaction_resolve_scene_player_review to open the next player-facing beat.",
	StorySeed:   aiGMReviewAdvanceStorySeed,
	MemorySeed:  aiGMReviewAdvanceMemorySeed,
	RequiredToolSet: []string{
		"daggerheart_combat_board_read",
		"daggerheart_scene_countdown_create",
		"daggerheart_scene_countdown_advance",
		"daggerheart_scene_countdown_resolve_trigger",
		"interaction_resolve_scene_player_review",
	},
	Prepare: prepareAttackReviewScenario,
	Assert:  assertReviewTurnReopenedWithPrompt,
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		order := []string{
			"daggerheart_combat_board_read",
			"daggerheart_scene_countdown_create",
			"daggerheart_scene_countdown_advance",
			"daggerheart_scene_countdown_resolve_trigger",
			"daggerheart_combat_board_read",
			"interaction_resolve_scene_player_review",
		}
		calls := flattenReplayToolCalls(fixture)
		assertReplayToolOrder(t, calls, order...)
		const countdownID = "breach-collapse-trigger"
		create := nthReplayToolCallByName(t, calls, "daggerheart_scene_countdown_create", 1)
		advance := nthReplayToolCallByName(t, calls, "daggerheart_scene_countdown_advance", 1)
		resolve := nthReplayToolCallByName(t, calls, "daggerheart_scene_countdown_resolve_trigger", 1)
		if got := asString(create.Arguments["countdown_id"]); got != countdownID {
			t.Fatalf("create countdown_id = %q, want %q", got, countdownID)
		}
		if got := asString(advance.Arguments["countdown_id"]); got != countdownID {
			t.Fatalf("advance countdown_id = %q, want %q", got, countdownID)
		}
		if got := asString(resolve.Arguments["countdown_id"]); got != countdownID {
			t.Fatalf("resolve countdown_id = %q, want %q", got, countdownID)
		}
	},
}

var aiGMGMMovePlacementReviewScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_gm_move_placement_review_live",
	FixtureFile: "ai_gm_campaign_context_gm_move_placement_review_live_replay.json",
	Prompt:      "The scene is waiting on GM review. This is a board-control lane, not a reference-lookup lane. First call daggerheart_combat_board_read. There is no active adversary yet, so create one with daggerheart_adversary_create using adversary_entry_id 'adversary.integration-foe' and notes 'Pressing through the split gate under the rising water.' Then spend 1 Fear with daggerheart_gm_move_apply using only direct_move { kind: ADDITIONAL_MOVE, shape: SHIFT_ENVIRONMENT }. Do not supply adversary_feature, environment_feature, or adversary_experience in this lane. Call daggerheart_combat_board_read again so the next beat reflects the updated board state, then use interaction_resolve_scene_player_review to open the next player-facing beat and update memory.md with the adversary placement and Fear move you applied.",
	StorySeed:   aiGMReviewAdvanceStorySeed,
	MemorySeed:  aiGMReviewAdvanceMemorySeed,
	RequiredToolSet: []string{
		"daggerheart_combat_board_read",
		"daggerheart_adversary_create",
		"daggerheart_gm_move_apply",
		"interaction_resolve_scene_player_review",
	},
	ForbiddenTools:  []string{"system_reference_search", "system_reference_read"},
	MaxToolErrors:   maxToolErrors(0),
	ReferenceLimits: &aiGMReferenceLimits{MaxSearches: 0, MaxReads: 0},
	Prepare:         prepareGMMovePlacementReviewScenario,
	Assert:          assertReviewTurnReopenedWithPrompt,
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		calls := flattenReplayToolCalls(fixture)
		assertReplayToolOrder(t, calls,
			"daggerheart_combat_board_read",
			"daggerheart_adversary_create",
			"daggerheart_gm_move_apply",
			"daggerheart_combat_board_read",
			"interaction_resolve_scene_player_review",
		)
		if replayToolCallCount(calls, "daggerheart_gm_move_apply") != 1 {
			t.Fatalf("daggerheart_gm_move_apply calls = %d, want 1", replayToolCallCount(calls, "daggerheart_gm_move_apply"))
		}
		move := nthReplayToolCallByName(t, calls, "daggerheart_gm_move_apply", 1)
		if got := replayNumericArgument(move.Arguments["fear_spent"]); got != 1 {
			t.Fatalf("fear_spent = %d, want 1", got)
		}
		directMove, _ := move.Arguments["direct_move"].(map[string]any)
		if got := asString(directMove["kind"]); got != "ADDITIONAL_MOVE" {
			t.Fatalf("direct_move.kind = %q, want %q", got, "ADDITIONAL_MOVE")
		}
		if got := asString(directMove["shape"]); got != "SHIFT_ENVIRONMENT" {
			t.Fatalf("direct_move.shape = %q, want %q", got, "SHIFT_ENVIRONMENT")
		}
	},
}

var aiGMAdversaryAttackReviewScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_adversary_attack_review_live",
	FixtureFile: "ai_gm_campaign_context_adversary_attack_review_live_replay.json",
	Prompt:      "The scene is waiting on GM review. First call character_sheet_read for the threatened player character and daggerheart_combat_board_read for the active threats. Then use daggerheart_adversary_attack_flow_resolve to have the adversary on the board strike that character with a physical attack in the active scene. Update memory.md with the defensive capability and adversary attack result you used, then use interaction_resolve_scene_player_review to open the next player-facing beat with a prompt beat for the same acting character.",
	StorySeed:   aiGMReviewAdvanceStorySeed,
	MemorySeed:  aiGMReviewAdvanceMemorySeed,
	RequiredToolSet: []string{
		"character_sheet_read",
		"daggerheart_combat_board_read",
		"daggerheart_adversary_attack_flow_resolve",
		"interaction_resolve_scene_player_review",
	},
	Prepare: prepareAttackReviewScenario,
	Assert:  assertReviewTurnReopenedWithPrompt,
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		assertReplayToolOrder(t, flattenReplayToolCalls(fixture),
			"character_sheet_read",
			"daggerheart_combat_board_read",
			"daggerheart_adversary_attack_flow_resolve",
			"interaction_resolve_scene_player_review",
		)
	},
}

var aiGMGroupActionReviewScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_group_action_review_live",
	FixtureFile: "ai_gm_campaign_context_group_action_review_live_replay.json",
	Prompt:      "The scene is waiting on GM review. Two player characters are present and coordinating to secure the flooded breach: Readiness Character 1 is the leader and Bram is the supporter. First call character_sheet_read for both acting characters. Then use daggerheart_group_action_flow_resolve with Readiness Character 1 as the leader and Bram as the supporter so they can force the gate shut together in the active scene. Update memory.md with the capabilities and group-action result you used, then use interaction_resolve_scene_player_review to open the next player-facing beat with a prompt beat for the same acting characters.",
	StorySeed:   aiGMReviewAdvanceStorySeed,
	MemorySeed:  aiGMReviewAdvanceMemorySeed,
	RequiredToolSet: []string{
		"character_sheet_read",
		"daggerheart_group_action_flow_resolve",
		"interaction_resolve_scene_player_review",
	},
	ExtraCharacters: []string{"Bram"},
	Prepare:         prepareGroupActionReviewScenario,
	Assert:          assertReviewTurnReopenedWithPrompt,
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		calls := flattenReplayToolCalls(fixture)
		if replayToolCallCount(calls, "character_sheet_read") < 2 {
			t.Fatal("expected at least two character_sheet_read calls before the group action flow")
		}
		assertReplayToolOrder(t, calls,
			"character_sheet_read",
			"character_sheet_read",
			"daggerheart_group_action_flow_resolve",
			"interaction_resolve_scene_player_review",
		)
	},
}

var aiGMTagTeamReviewScenario = aiGMCampaignScenarioSpec{
	Name:        "ai_gm_campaign_context_tag_team_review_live",
	FixtureFile: "ai_gm_campaign_context_tag_team_review_live_replay.json",
	Prompt:      "The scene is waiting on GM review. Two player characters and one active adversary are on the board: Readiness Character 1 and Bram are the acting characters. First call daggerheart_combat_board_read, then call character_sheet_read for both acting characters. Use daggerheart_tag_team_flow_resolve so Readiness Character 1 and Bram coordinate a tag-team strike against the active threat in the scene. Update memory.md with the capabilities and combined result you used, then use interaction_resolve_scene_player_review to open the next player-facing beat with a prompt beat for the same acting characters.",
	StorySeed:   aiGMReviewAdvanceStorySeed,
	MemorySeed:  aiGMReviewAdvanceMemorySeed,
	RequiredToolSet: []string{
		"daggerheart_combat_board_read",
		"character_sheet_read",
		"daggerheart_tag_team_flow_resolve",
		"interaction_resolve_scene_player_review",
	},
	ExtraCharacters: []string{"Bram"},
	Prepare:         prepareTagTeamReviewScenario,
	Assert:          assertReviewTurnReopenedWithPrompt,
	AssertFixture: func(t *testing.T, fixture openAIReplayFixture) {
		t.Helper()
		calls := flattenReplayToolCalls(fixture)
		if replayToolCallCount(calls, "character_sheet_read") < 2 {
			t.Fatal("expected at least two character_sheet_read calls before the tag-team flow")
		}
		assertReplayToolOrder(t, calls,
			"daggerheart_combat_board_read",
			"character_sheet_read",
			"character_sheet_read",
			"daggerheart_tag_team_flow_resolve",
			"interaction_resolve_scene_player_review",
		)
	},
}

func prepareAttackReviewScenario(t *testing.T, setup *aiGMCampaignScenarioSetup) {
	t.Helper()
	setScenarioGMAuthority(t, setup, setup.AIGMParticipantID)
	sceneID := createScenarioScene(t, setup, "Harbor Skirmish", "Aria braces against a raider in the flooded breach.", nil, setup.CharacterID)
	setup.ReplayTokens["scene_id"] = sceneID
	requireActiveScene(t, setup, sceneID)

	createAdversary, err := setup.DaggerheartClient.CreateAdversary(setup.UserCtx, &pb.DaggerheartCreateAdversaryRequest{
		CampaignId:       setup.CampaignID,
		SessionId:        setup.SessionID,
		SceneId:          sceneID,
		AdversaryEntryId: "adversary.integration-foe",
		Notes:            "Holding the broken gate",
	})
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	adversaryID := createAdversary.GetAdversary().GetId()
	if adversaryID == "" {
		t.Fatal("expected adversary id")
	}
	setup.ReplayTokens["adversary_id"] = adversaryID
	requireVisibleAdversaryOnSceneBoard(t, setup, sceneID, adversaryID)

	openScenarioPlayerPhase(t, setup, sceneID, "Broken Gate", []string{setup.CharacterID},
		aiGMInteractionBeat{Type: gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT, Text: "Aria, the raider lunges through the breach. What do you do?"},
	)
	submitScenarioPlayerAction(t, setup, sceneID, "Aria drives in with her weapon to force the raider back from the breach before it can break through.", true, setup.CharacterID)
	waitForGMReviewReady(t, setup, sceneID)
}

func prepareReactionReviewScenario(t *testing.T, setup *aiGMCampaignScenarioSetup) {
	t.Helper()
	setScenarioGMAuthority(t, setup, setup.AIGMParticipantID)
	sceneID := createScenarioScene(t, setup, "Flooded Archive", "A broken iron door drops toward Aria as the room shifts under the water.", nil, setup.CharacterID)
	setup.ReplayTokens["scene_id"] = sceneID
	openScenarioPlayerPhase(t, setup, sceneID, "Falling Door", []string{setup.CharacterID},
		aiGMInteractionBeat{Type: gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT, Text: "Aria, the door is coming down fast. How do you keep your footing and avoid getting pinned?"},
	)
	submitScenarioPlayerAction(t, setup, sceneID, "Aria twists aside and braces off the wall to slip clear of the falling door before it can pin her.", true, setup.CharacterID)
}

func prepareSubdueIntentReviewScenario(t *testing.T, setup *aiGMCampaignScenarioSetup) {
	t.Helper()
	setScenarioGMAuthority(t, setup, setup.AIGMParticipantID)
	sceneID := createScenarioScene(t, setup, "Lantern Spill", "A hooded stranger hesitates near the oil stores while the harbor road lies open behind them.", nil, setup.CharacterID)
	setup.ReplayTokens["scene_id"] = sceneID
	openScenarioPlayerPhase(t, setup, sceneID, "Cornered Runner", []string{setup.CharacterID},
		aiGMInteractionBeat{Type: gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT, Text: "Aria, the stranger shifts to bolt down the cliff road. What do you do?"},
	)
	submitScenarioPlayerAction(t, setup, sceneID, "I rush with the sword and try to incapacitate them before they can run or shout.", true, setup.CharacterID)
	waitForGMReviewReady(t, setup, sceneID)
}

func prepareGMMovePlacementReviewScenario(t *testing.T, setup *aiGMCampaignScenarioSetup) {
	t.Helper()
	setScenarioGMAuthority(t, setup, setup.AIGMParticipantID)
	sceneID := createScenarioScene(t, setup, "Shattered Lock", "The lower gate groans as water pounds against the timbers.", nil, setup.CharacterID)
	setup.ReplayTokens["scene_id"] = sceneID
	requireActiveScene(t, setup, sceneID)
	requireNoVisibleAdversaryOnSceneBoard(t, setup, sceneID)
	_, err := setup.SnapshotClient.UpdateSnapshotState(setup.UserCtx, &gamev1.UpdateSnapshotStateRequest{
		CampaignId: setup.CampaignID,
		SystemSnapshotUpdate: &gamev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &pb.DaggerheartSnapshot{
				GmFear:                2,
				ConsecutiveShortRests: 0,
			},
		},
	})
	if err != nil {
		t.Fatalf("update snapshot state: %v", err)
	}
	openScenarioPlayerPhase(t, setup, sceneID, "Breaking Water", []string{setup.CharacterID},
		aiGMInteractionBeat{Type: gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT, Text: "Aria, the gate is giving way under the water pressure. What do you do?"},
	)
	submitScenarioPlayerAction(t, setup, sceneID, "Aria braces the lock with a hooked pole and tries to keep the gate from splitting open.", true, setup.CharacterID)
	waitForGMReviewReady(t, setup, sceneID)
}

func prepareGroupActionReviewScenario(t *testing.T, setup *aiGMCampaignScenarioSetup) {
	t.Helper()
	setScenarioGMAuthority(t, setup, setup.AIGMParticipantID)
	supporterID := extraCharacterID(t, setup, "Bram")
	setup.ReplayTokens["supporter_character_id"] = supporterID
	sceneID := createScenarioScene(t, setup, "Floodgate Teamwork", "Aria and Bram struggle to seal the breach before the chamber fills.", nil, setup.CharacterID, supporterID)
	setup.ReplayTokens["scene_id"] = sceneID
	openScenarioPlayerPhase(t, setup, sceneID, "Seal the Breach", []string{setup.CharacterID, supporterID},
		aiGMInteractionBeat{Type: gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT, Text: "Aria and Bram, the gate is slipping out of place. How do you work together to force it shut?"},
	)
	submitScenarioPlayerAction(t, setup, sceneID, "Aria anchors the gate while Bram heaves the locking bar back into place with her.", true, setup.CharacterID, supporterID)
}

func prepareTagTeamReviewScenario(t *testing.T, setup *aiGMCampaignScenarioSetup) {
	t.Helper()
	setScenarioGMAuthority(t, setup, setup.AIGMParticipantID)
	secondID := extraCharacterID(t, setup, "Bram")
	setup.ReplayTokens["second_character_id"] = secondID
	sceneID := createScenarioScene(t, setup, "Broken Causeway", "Aria and Bram press a raider across the slick stones above the surge.", nil, setup.CharacterID, secondID)
	setup.ReplayTokens["scene_id"] = sceneID

	createAdversary, err := setup.DaggerheartClient.CreateAdversary(setup.UserCtx, &pb.DaggerheartCreateAdversaryRequest{
		CampaignId:       setup.CampaignID,
		SessionId:        setup.SessionID,
		SceneId:          sceneID,
		AdversaryEntryId: "adversary.integration-foe",
		Notes:            "Overextended on the causeway edge",
	})
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	adversaryID := createAdversary.GetAdversary().GetId()
	if adversaryID == "" {
		t.Fatal("expected adversary id")
	}
	setup.ReplayTokens["adversary_id"] = adversaryID

	openScenarioPlayerPhase(t, setup, sceneID, "Causeway Press", []string{setup.CharacterID, secondID},
		aiGMInteractionBeat{Type: gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT, Text: "Aria and Bram, the raider is off balance at the edge. How do you strike together before it regains footing?"},
	)
	submitScenarioPlayerAction(t, setup, sceneID, "Aria drives the raider high while Bram comes in low to sweep its footing out from under it.", true, setup.CharacterID, secondID)
}

// ServeHTTP forwards live Responses requests to the configured upstream and records the sanitized exchange.
func (r *openAILiveRecorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		r.setErr(fmt.Errorf("read request body: %w", err))
		http.Error(w, "read request body", http.StatusInternalServerError)
		return
	}
	upstreamURL, err := r.upstreamURL(req.URL)
	if err != nil {
		r.setErr(fmt.Errorf("resolve upstream url: %w", err))
		http.Error(w, "resolve upstream url", http.StatusInternalServerError)
		return
	}
	targetReq, err := http.NewRequestWithContext(req.Context(), req.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		r.setErr(fmt.Errorf("build upstream request: %w", err))
		http.Error(w, "build upstream request", http.StatusInternalServerError)
		return
	}
	for key, values := range req.Header {
		if strings.EqualFold(key, "Host") || strings.EqualFold(key, "Accept-Encoding") {
			continue
		}
		for _, value := range values {
			targetReq.Header.Add(key, value)
		}
	}
	res, err := r.client.Do(targetReq)
	if err != nil {
		r.setErr(fmt.Errorf("forward request: %w", err))
		http.Error(w, "forward request", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		r.setErr(fmt.Errorf("read upstream response: %w", err))
		http.Error(w, "read upstream response", http.StatusBadGateway)
		return
	}
	for key, values := range res.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(res.StatusCode)
	_, _ = w.Write(responseBody)

	r.recordExchange(req.Method, upstreamURL, body, res.StatusCode, responseBody)
}

// upstreamURL preserves the incoming request path while anchoring requests to the configured provider base.
func (r *openAILiveRecorder) upstreamURL(requestURL *url.URL) (string, error) {
	base, err := url.Parse(r.targetURL)
	if err != nil {
		return "", err
	}
	if requestURL == nil {
		return base.String(), nil
	}
	resolved := *base
	requestPath := strings.TrimSpace(requestURL.Path)
	switch requestPath {
	case "", "/":
		resolved.Path = base.Path
	default:
		resolved.Path = pathpkg.Join(pathpkg.Dir(base.Path), requestPath)
	}
	resolved.RawQuery = requestURL.RawQuery
	return resolved.String(), nil
}

// recordExchange converts one live provider response into the replay-oriented fixture shape.
func (r *openAILiveRecorder) recordExchange(method, requestURL string, requestBody []byte, statusCode int, responseBody []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var requestPayload map[string]any
	if trimmed := strings.TrimSpace(string(requestBody)); trimmed != "" {
		if err := json.Unmarshal(requestBody, &requestPayload); err != nil {
			r.setErr(fmt.Errorf("parse recorded request body: %w", err))
			return
		}
	}
	r.rawCapture.Metadata.CapturedAtUTC = time.Now().UTC().Format(time.RFC3339)
	previousResponseID := strings.TrimSpace(asString(requestPayload["previous_response_id"]))
	r.rawCapture.Exchanges = append(r.rawCapture.Exchanges, openAILiveExchange{
		Step:           len(r.steps),
		Method:         method,
		RequestURL:     requestURL,
		StatusCode:     statusCode,
		RequestBody:    append(json.RawMessage(nil), requestBody...),
		ResponseBody:   append(json.RawMessage(nil), responseBody...),
		CapturedAtUTC:  time.Now().UTC().Format(time.RFC3339),
		PreviousRespID: previousResponseID,
	})
	if !isResponsesRequestURL(requestURL) {
		return
	}
	var responsePayload openAIResponsesPayload
	if err := json.Unmarshal(responseBody, &responsePayload); err != nil {
		r.setErr(fmt.Errorf("parse recorded response body: %w", err))
		return
	}
	if len(r.steps) == 0 {
		prompt, toolNames := extractPromptAndToolNames(requestPayload)
		r.initialPrompt = prompt
		r.initialTools = append([]string(nil), toolNames...)
	}
	r.captureCallOutputs(requestPayload)
	r.steps = append(r.steps, replayStepFromLiveResponse(responsePayload))
}

// isResponsesRequestURL identifies actual Responses API exchanges even when the configured endpoint has queries or alternate prefixes.
func isResponsesRequestURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	path := strings.TrimRight(strings.TrimSpace(parsed.Path), "/")
	return strings.HasSuffix(path, "/responses")
}

// captureCallOutputs keeps tool-call outputs visible in failure logs when a live run diverges.
func (r *openAILiveRecorder) captureCallOutputs(payload map[string]any) {
	inputItems, _ := payload["input"].([]any)
	for _, raw := range inputItems {
		item, _ := raw.(map[string]any)
		if strings.TrimSpace(asString(item["type"])) != "function_call_output" {
			continue
		}
		callID := strings.TrimSpace(asString(item["call_id"]))
		output := strings.TrimSpace(asString(item["output"]))
		if callID == "" {
			continue
		}
		r.requestDebug = append(r.requestDebug, fmt.Sprintf("step=%d call_id=%s output=%s", len(r.steps), callID, output))
	}
}

// setErr preserves the first recorder failure so later transport noise does not hide the root cause.
func (r *openAILiveRecorder) setErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.firstErr == nil {
		r.firstErr = err
	}
}

// Reset clears the recorder's accumulated state so it can be reused for a
// retry attempt. The target URL, HTTP client, model, and scenario spec are
// preserved.
func (r *openAILiveRecorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.firstErr = nil
	r.initialPrompt = ""
	r.initialTools = nil
	r.steps = nil
	r.rawCapture = openAILiveCapture{
		Metadata: openAIReplayMetadata{
			Provider:        "openai",
			Model:           r.model,
			ReasoningEffort: r.rawCapture.Metadata.ReasoningEffort,
			Scenario:        r.scenario.Name,
			Source:          "live_capture",
		},
	}
	r.requestDebug = nil
}

// Err returns the first recorder failure observed during proxying.
func (r *openAILiveRecorder) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.firstErr
}

// DebugString summarizes recorded call outputs for failure diagnostics.
func (r *openAILiveRecorder) DebugString() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.Join(r.requestDebug, "\n")
}

// ReplayFixture converts the captured live exchange into the stable tokenized fixture committed in the repo.
func (r *openAILiveRecorder) ReplayFixture(tokens map[string]string) openAIReplayFixture {
	r.mu.Lock()
	defer r.mu.Unlock()
	fixture := openAIReplayFixture{
		Metadata: &openAIReplayMetadata{
			Provider:        "openai",
			Model:           r.model,
			ReasoningEffort: r.rawCapture.Metadata.ReasoningEffort,
			CapturedAtUTC:   r.rawCapture.Metadata.CapturedAtUTC,
			Scenario:        r.scenario.Name,
			Source:          "live_capture",
		},
		InitialPromptContains: initialPromptContainsForLiveCapture(r.scenario),
		InitialToolNames:      append([]string(nil), r.initialTools...),
		Steps:                 append([]openAIReplayStep(nil), r.steps...),
	}
	return tokenizeReplayFixture(fixture, tokens)
}

// replayStepFromLiveResponse trims the full provider response down to the deterministic replay contract.
func replayStepFromLiveResponse(payload openAIResponsesPayload) openAIReplayStep {
	step := openAIReplayStep{
		ID:         strings.TrimSpace(payload.ID),
		OutputText: strings.TrimSpace(payload.OutputText),
		ToolCalls:  make([]openAIReplayToolCall, 0, len(payload.Output)),
	}
	for _, item := range payload.Output {
		if strings.TrimSpace(item.Type) == "function_call" {
			args := map[string]any{}
			if strings.TrimSpace(item.Arguments) != "" {
				_ = json.Unmarshal([]byte(item.Arguments), &args)
			}
			step.ToolCalls = append(step.ToolCalls, openAIReplayToolCall{
				CallID:    strings.TrimSpace(item.CallID),
				Name:      strings.TrimSpace(item.Name),
				Arguments: args,
			})
			continue
		}
		if step.OutputText != "" {
			continue
		}
		for _, content := range item.Content {
			if strings.TrimSpace(content.Text) == "" {
				continue
			}
			step.OutputText = strings.TrimSpace(content.Text)
			break
		}
	}
	return step
}

func liveCaptureArtifactStem(scenarioName, model string) string {
	if caseID := strings.TrimSpace(os.Getenv(integrationAIEvalCaseIDEnv)); caseID != "" {
		return sanitizeLiveCaptureToken(caseID)
	}
	parts := []string{scenarioName, model}
	if promptProfile := strings.TrimSpace(os.Getenv(integrationAIPromptProfileEnv)); promptProfile != "" {
		parts = append(parts, promptProfile)
	}
	return sanitizeLiveCaptureToken(strings.Join(parts, "-"))
}

func sanitizeLiveCaptureToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "live-capture"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if lastDash {
			continue
		}
		b.WriteByte('-')
		lastDash = true
	}
	return strings.Trim(b.String(), "-")
}

// writeOpenAILiveCapture persists the raw live capture outside the repo fixtures for local debugging and review.
func writeOpenAILiveCapture(t *testing.T, artifactStem string, capturedAt string, capture openAILiveCapture) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), ".tmp", "ai-live-captures")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create capture dir: %v", err)
	}
	filename := fmt.Sprintf("%s-%s.json", artifactStem, capturedAt)
	path := filepath.Join(dir, filename)
	data, err := json.MarshalIndent(capture, "", "  ")
	if err != nil {
		t.Fatalf("marshal live capture: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write live capture: %v", err)
	}
	return path
}

// writeOpenAILiveCaptureReport writes a human-readable markdown report alongside
// the raw capture with token usage, tool sequence, narrative excerpts, and error
// summary for cross-model quality comparison.
func writeOpenAILiveCaptureReport(t *testing.T, artifactStem string, capturedAt string, scenarioName string, recorder *openAILiveRecorder, result aiGMCampaignScenarioResult, diagnosticsPath string) string {
	t.Helper()
	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	usage := aggregateLiveCaptureUsage(recorder.rawCapture)
	model := recorder.model
	fingerprint := liveOpenVikingConfigFingerprint()
	promptInspection := inspectInitialPrompt(recorder.initialPrompt)
	diagnostics := result.PromptDiagnostics

	// Extract narrative fields from recorded steps.
	var sceneName, sceneDesc, gmNarration, playerPromptBeat, memoryContent string
	var committedBeatTypes []string
	toolSequence, referenceSearches, referenceReads := liveToolCounts(recorder.steps)
	var toolErrors []string
	for _, step := range recorder.steps {
		for _, call := range step.ToolCalls {
			switch call.Name {
			case "scene_create":
				sceneName = asString(call.Arguments["name"])
				sceneDesc = asString(call.Arguments["description"])
			case "interaction_record_scene_gm_interaction":
				gmNarration = interactionBeatText(call.Arguments["interaction"], "fiction")
				committedBeatTypes = interactionBeatTypes(call.Arguments["interaction"])
			case "interaction_open_scene_player_phase":
				if strings.TrimSpace(gmNarration) == "" {
					gmNarration = interactionBeatText(call.Arguments["interaction"], "fiction")
				}
				playerPromptBeat = interactionBeatText(call.Arguments["interaction"], "prompt")
				committedBeatTypes = interactionBeatTypes(call.Arguments["interaction"])
			case "campaign_memory_section_update":
				memoryContent = asString(call.Arguments["content"])
			case "campaign_artifact_upsert":
				if asString(call.Arguments["path"]) == "memory.md" {
					memoryContent = asString(call.Arguments["content"])
				}
			}
		}
	}
	for _, line := range recorder.requestDebug {
		if strings.Contains(line, "tool call failed") {
			toolErrors = append(toolErrors, line)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Live Capture Report\n\n")
	fmt.Fprintf(&b, "- **Model:** %s\n", model)
	fmt.Fprintf(&b, "- **Captured:** %s\n", recorder.rawCapture.Metadata.CapturedAtUTC)
	fmt.Fprintf(&b, "- **Scenario:** %s\n", scenarioName)
	fmt.Fprintf(&b, "- **Run Status:** %s\n", result.RunStatus)
	fmt.Fprintf(&b, "- **Metric Status:** %s\n", result.MetricStatus)
	fmt.Fprintf(&b, "- **Active Scene ID:** %s\n\n", activeSceneID(result.InteractionState))
	if strings.TrimSpace(result.FailureSummary) != "" {
		fmt.Fprintf(&b, "## Failure Diagnostics\n\n")
		fmt.Fprintf(&b, "- **Kind:** %s\n", result.FailureKind)
		fmt.Fprintf(&b, "- **Summary:** %s\n", result.FailureSummary)
		fmt.Fprintf(&b, "- **Reason:** %s\n", result.FailureReason)
		if strings.TrimSpace(diagnosticsPath) != "" {
			fmt.Fprintf(&b, "- **Diagnostics Artifact:** %s\n", diagnosticsPath)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "## OpenViking\n\n")
	fmt.Fprintf(&b, "- **Enabled:** %t\n", fingerprint.Enabled)
	if fingerprint.Enabled {
		fmt.Fprintf(&b, "- **Mode:** %s\n", strings.TrimSpace(fingerprint.Mode))
		fmt.Fprintf(&b, "- **Embedding Model:** %s\n", strings.TrimSpace(fingerprint.EmbeddingModel))
		fmt.Fprintf(&b, "- **VLM Model:** %s\n", strings.TrimSpace(fingerprint.VLMModel))
	}
	fmt.Fprintf(&b, "- **Prompt Collector Includes Raw `story.md`:** %t\n", diagnostics.ContextPolicy.IncludeStory)
	fmt.Fprintf(&b, "- **Prompt Collector Includes Raw `memory.md`:** %t\n", diagnostics.ContextPolicy.IncludeMemory)
	fmt.Fprintf(&b, "- **Augmentation Attempted:** %t\n", diagnostics.Augmentation.Attempted)
	fmt.Fprintf(&b, "- **Search Attempted:** %t\n", diagnostics.Augmentation.SearchAttempted)
	fmt.Fprintf(&b, "- **Augmentation Degraded:** %t\n", diagnostics.Augmentation.Degraded)
	if reason := strings.TrimSpace(diagnostics.Augmentation.DegradationReason); reason != "" {
		fmt.Fprintf(&b, "- **Degradation Reason:** %s\n", reason)
	}
	if len(diagnostics.Augmentation.MirroredTargets) > 0 {
		fmt.Fprintf(&b, "- **Mirrored Targets:** %s\n", strings.Join(diagnostics.Augmentation.MirroredTargets, ", "))
	}
	fmt.Fprintf(&b, "- **Raw `story.md` In Prompt:** %t\n", promptInspection.HasStory)
	fmt.Fprintf(&b, "- **Raw `memory.md` In Prompt:** %t\n", promptInspection.HasMemory)
	fmt.Fprintf(&b, "- **Retrieved Resource Count:** %d\n", retrievedResourceCount(result.RetrievedContexts))
	fmt.Fprintf(&b, "- **Retrieved Memory Count:** %d\n", retrievedMemoryCount(result.RetrievedContexts))
	if len(result.RetrievedContexts) > 0 {
		fmt.Fprintf(&b, "\n### Retrieved Context URIs\n\n")
		for _, item := range result.RetrievedContexts {
			fmt.Fprintf(&b, "- `%s` (%s)\n", item.URI, item.ContextType)
			if renderedURI := strings.TrimSpace(item.RenderedURI); renderedURI != "" {
				fmt.Fprintf(&b, "  rendered: `%s`\n", renderedURI)
			}
			if source := strings.TrimSpace(item.ContentSource); source != "" {
				fmt.Fprintf(&b, "  source: `%s`\n", source)
			}
			if renderErr := strings.TrimSpace(item.ContentError); renderErr != "" {
				fmt.Fprintf(&b, "  error: `%s`\n", renderErr)
			}
		}
	}
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "## Token Usage\n\n")
	fmt.Fprintf(&b, "| Metric | Tokens |\n")
	fmt.Fprintf(&b, "|--------|-------:|\n")
	fmt.Fprintf(&b, "| Input | %d |\n", usage.InputTokens)
	fmt.Fprintf(&b, "| Output | %d |\n", usage.OutputTokens)
	fmt.Fprintf(&b, "| Reasoning | %d |\n", usage.ReasoningTokens)
	fmt.Fprintf(&b, "| Total | %d |\n\n", usage.TotalTokens)

	fmt.Fprintf(&b, "## Reference Usage\n\n")
	fmt.Fprintf(&b, "- `system_reference_search`: %d\n", referenceSearches)
	fmt.Fprintf(&b, "- `system_reference_read`: %d\n\n", referenceReads)

	fmt.Fprintf(&b, "## Tool Sequence (%d calls)\n\n", len(toolSequence))
	for i, name := range toolSequence {
		fmt.Fprintf(&b, "%d. `%s`\n", i+1, name)
	}
	if len(toolErrors) > 0 {
		fmt.Fprintf(&b, "\n### Errors (%d)\n\n", len(toolErrors))
		for _, e := range toolErrors {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}

	fmt.Fprintf(&b, "\n## Narrative Quality\n\n")
	fmt.Fprintf(&b, "### Scene: %q\n\n%s\n\n", sceneName, sceneDesc)
	if len(committedBeatTypes) > 0 {
		fmt.Fprintf(&b, "### Committed Beat Types\n\n%s\n\n", strings.Join(committedBeatTypes, ", "))
	}
	fmt.Fprintf(&b, "### GM Narration\n\n%s\n\n", gmNarration)
	fmt.Fprintf(&b, "### Player-Facing Prompt Beat\n\n%s\n\n", playerPromptBeat)
	fmt.Fprintf(&b, "### Memory Update\n\n%s\n\n", memoryContent)
	fmt.Fprintf(&b, "### Final Output\n\n%s\n", strings.TrimSpace(result.OutputText))

	dir := filepath.Join(repoRoot(t), ".tmp", "ai-live-captures")
	filename := fmt.Sprintf("%s-%s.md", artifactStem, capturedAt)
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write quality report: %v", err)
	}
	return path
}

func buildOpenAILiveCaptureSummary(t *testing.T, scenarioName string, recorder *openAILiveRecorder, result aiGMCampaignScenarioResult, rawPath, reportPath, diagnosticsPath string) openAILiveCaptureSummary {
	t.Helper()
	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	usage := aggregateLiveCaptureUsage(recorder.rawCapture)
	toolNames, referenceSearches, referenceReads := liveToolCounts(recorder.steps)
	toolErrors := liveToolErrorCount(recorder.requestDebug)
	resultClass := liveCaptureResultCleanPass
	if toolErrors > 0 {
		resultClass = liveCaptureResultPassWithToolError
	}
	fingerprint := liveOpenVikingConfigFingerprint()
	promptInspection := inspectInitialPrompt(recorder.initialPrompt)
	return openAILiveCaptureSummary{
		CaseID:                     strings.TrimSpace(os.Getenv(integrationAIEvalCaseIDEnv)),
		Scenario:                   scenarioName,
		Model:                      recorder.model,
		ReasoningEffort:            recorder.rawCapture.Metadata.ReasoningEffort,
		OpenVikingEnabled:          fingerprint.Enabled,
		OpenVikingMode:             fingerprint.Mode,
		OpenVikingEmbeddingModel:   fingerprint.EmbeddingModel,
		OpenVikingVLMModel:         fingerprint.VLMModel,
		PromptContextIncludeStory:  result.PromptDiagnostics.ContextPolicy.IncludeStory,
		PromptContextIncludeMemory: result.PromptDiagnostics.ContextPolicy.IncludeMemory,
		AugmentationAttempted:      result.PromptDiagnostics.Augmentation.Attempted,
		AugmentationSearchTried:    result.PromptDiagnostics.Augmentation.SearchAttempted,
		AugmentationDegraded:       result.PromptDiagnostics.Augmentation.Degraded,
		AugmentationError:          result.PromptDiagnostics.Augmentation.DegradationReason,
		MirroredTargets:            append([]string(nil), result.PromptDiagnostics.Augmentation.MirroredTargets...),
		RunStatus:                  result.RunStatus,
		MetricStatus:               result.MetricStatus,
		FailureKind:                result.FailureKind,
		FailureSummary:             result.FailureSummary,
		FailureReason:              result.FailureReason,
		ResultClass:                resultClass,
		ToolNames:                  append([]string(nil), toolNames...),
		ToolErrorCount:             toolErrors,
		ReferenceSearchCount:       referenceSearches,
		ReferenceReadCount:         referenceReads,
		UnexpectedReferenceLookups: unexpectedReferenceLookupCount(recorder.scenario, referenceSearches, referenceReads),
		InitialPromptHasStory:      promptInspection.HasStory,
		InitialPromptHasMemory:     promptInspection.HasMemory,
		RetrievedContextURIs:       retrievedContextURIs(result.RetrievedContexts),
		RetrievedRenderedURIs:      retrievedRenderedURIs(result.RetrievedContexts),
		RetrievedContentSources:    retrievedContentSources(result.RetrievedContexts),
		RetrievedResourceCount:     retrievedResourceCount(result.RetrievedContexts),
		RetrievedMemoryCount:       retrievedMemoryCount(result.RetrievedContexts),
		InputTokens:                usage.InputTokens,
		OutputTokens:               usage.OutputTokens,
		ReasoningTokens:            usage.ReasoningTokens,
		TotalTokens:                usage.TotalTokens,
		RawCaptureFile:             filepath.Base(rawPath),
		MarkdownReport:             filepath.Base(reportPath),
		DiagnosticsFile:            liveCaptureArtifactBaseName(diagnosticsPath),
		GeneratedAtUTC:             time.Now().UTC().Format(time.RFC3339),
		ActiveSceneID:              activeSceneID(result.InteractionState),
	}
}

func writeOpenAILiveCaptureSummary(t *testing.T, artifactStem, capturedAt string, summary openAILiveCaptureSummary) string {
	t.Helper()

	dir := filepath.Join(repoRoot(t), ".tmp", "ai-live-captures")
	filename := fmt.Sprintf("%s-%s.summary.json", artifactStem, capturedAt)
	path := filepath.Join(dir, filename)
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatalf("marshal live capture summary: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write live capture summary: %v", err)
	}
	return path
}

func assertValidOpenVikingAugmentation(t *testing.T, summary openAILiveCaptureSummary) {
	t.Helper()
	if !summary.OpenVikingEnabled {
		t.Fatal("valid OpenViking augmentation requested, but OpenViking is disabled")
	}
	if !summary.AugmentationAttempted {
		t.Fatal("expected OpenViking augmentation attempt")
	}
	if summary.AugmentationDegraded {
		t.Fatalf("expected non-degraded OpenViking augmentation, got %q", summary.AugmentationError)
	}
	if !summary.AugmentationSearchTried {
		t.Fatal("expected OpenViking retrieval search to run")
	}
	if summary.RetrievedResourceCount+summary.RetrievedMemoryCount <= 0 {
		t.Fatal("expected at least one retrieved OpenViking context")
	}
	switch strings.TrimSpace(summary.OpenVikingMode) {
	case "docs_aligned_supplement":
		if summary.InitialPromptHasStory {
			t.Fatal("docs_aligned_supplement should suppress raw story.md from the prompt")
		}
		if !summary.InitialPromptHasMemory {
			t.Fatal("docs_aligned_supplement should keep raw memory.md in the prompt")
		}
		if summary.RetrievedResourceCount > 0 && !containsPreferredRetrievedContentSource(summary.RetrievedContentSources, "backing_read", "backing_tree_read", "leaf_read") {
			t.Fatalf("docs_aligned_supplement should render retrieved story content from file-grade reads, got sources=%v", summary.RetrievedContentSources)
		}
		if duplicate := firstDuplicateNormalizedItem(summary.RetrievedRenderedURIs); duplicate != "" {
			t.Fatalf("docs_aligned_supplement should not render duplicate retrieved targets, duplicate=%q rendered_uris=%v", duplicate, summary.RetrievedRenderedURIs)
		}
	case "legacy":
		if summary.InitialPromptHasStory || summary.InitialPromptHasMemory {
			t.Fatal("legacy OpenViking mode should suppress raw story.md and memory.md from the prompt")
		}
	}
}

func retrievedContextURIs(items []orchestration.RetrievedContext) []string {
	if len(items) == 0 {
		return nil
	}
	uris := make([]string, 0, len(items))
	for _, item := range items {
		if uri := strings.TrimSpace(item.URI); uri != "" {
			uris = append(uris, uri)
		}
	}
	return uris
}

func retrievedRenderedURIs(items []orchestration.RetrievedContext) []string {
	if len(items) == 0 {
		return nil
	}
	uris := make([]string, 0, len(items))
	for _, item := range items {
		if uri := strings.TrimSpace(item.RenderedURI); uri != "" {
			uris = append(uris, uri)
		}
	}
	return uris
}

func retrievedContentSources(items []orchestration.RetrievedContext) []string {
	if len(items) == 0 {
		return nil
	}
	sources := make([]string, 0, len(items))
	for _, item := range items {
		if source := strings.TrimSpace(item.ContentSource); source != "" {
			sources = append(sources, source)
		}
	}
	return sources
}

func containsPreferredRetrievedContentSource(items []string, want ...string) bool {
	if len(items) == 0 || len(want) == 0 {
		return false
	}
	allowed := map[string]struct{}{}
	for _, item := range want {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		allowed[item] = struct{}{}
	}
	for _, item := range items {
		if _, ok := allowed[strings.TrimSpace(item)]; ok {
			return true
		}
	}
	return false
}

func firstDuplicateNormalizedItem(items []string) string {
	seen := map[string]struct{}{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			return item
		}
		seen[item] = struct{}{}
	}
	return ""
}

func retrievedResourceCount(items []orchestration.RetrievedContext) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.ContextType), "resource") || strings.HasPrefix(strings.TrimSpace(item.URI), "viking://resources/") {
			count++
		}
	}
	return count
}

func retrievedMemoryCount(items []orchestration.RetrievedContext) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.ContextType), "memory") || strings.Contains(strings.TrimSpace(item.URI), "/memories/") {
			count++
		}
	}
	return count
}

func writeOpenAILiveCaptureDiagnostics(t *testing.T, artifactStem string, capturedAt string, diagnostics *aiGMScenarioDiagnostics) string {
	t.Helper()
	if diagnostics == nil {
		return ""
	}
	dir := filepath.Join(repoRoot(t), ".tmp", "ai-live-captures")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create diagnostics dir: %v", err)
	}
	filename := fmt.Sprintf("%s-%s.diagnostics.json", artifactStem, capturedAt)
	path := filepath.Join(dir, filename)
	data, err := json.MarshalIndent(diagnostics, "", "  ")
	if err != nil {
		t.Fatalf("marshal live capture diagnostics: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write live capture diagnostics: %v", err)
	}
	return path
}

func liveCaptureArtifactBaseName(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.Base(path)
}

func writePromptfooEvalOutputIfRequested(t *testing.T, spec aiGMCampaignScenarioSpec, recorder *openAILiveRecorder, result aiGMCampaignScenarioResult, fixture openAIReplayFixture, rawPath, reportPath, summaryPath, diagnosticsPath string) string {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(integrationAIEvalOutputPathEnv))
	if path == "" {
		return ""
	}
	output := buildPromptfooEvalOutput(spec, recorder, result, fixture, rawPath, reportPath, summaryPath, diagnosticsPath)
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("marshal promptfoo eval output: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write promptfoo eval output: %v", err)
	}
	return path
}

func buildPromptfooEvalOutput(spec aiGMCampaignScenarioSpec, recorder *openAILiveRecorder, result aiGMCampaignScenarioResult, fixture openAIReplayFixture, rawPath, reportPath, summaryPath, diagnosticsPath string) evalsupport.Output {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	toolNames, referenceSearches, referenceReads := liveToolCounts(recorder.steps)
	toolErrors := liveToolErrorCount(recorder.requestDebug)
	resultClass := string(liveCaptureResultCleanPass)
	if toolErrors > 0 {
		resultClass = string(liveCaptureResultPassWithToolError)
	}
	label := spec.Name
	if scenario, ok := evalsupport.ScenarioByID(spec.Name); ok {
		label = scenario.Label
	}

	toolCalls := flattenReplayToolCalls(fixture)
	outCalls := make([]evalsupport.ToolCall, 0, len(toolCalls))
	for _, call := range toolCalls {
		outCalls = append(outCalls, evalsupport.ToolCall{
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}

	characterState := evalsupport.CharacterState{}
	if state := result.CharacterState; state != nil {
		characterState = evalsupport.CharacterState{
			HP:     int(state.GetHp()),
			Hope:   int(state.GetHope()),
			Stress: int(state.GetStress()),
			Armor:  int(state.GetArmor()),
		}
	}

	return evalsupport.Output{
		CaseID:                     strings.TrimSpace(os.Getenv(integrationAIEvalCaseIDEnv)),
		Scenario:                   spec.Name,
		Label:                      label,
		Model:                      recorder.model,
		ReasoningEffort:            recorder.rawCapture.Metadata.ReasoningEffort,
		PromptProfile:              strings.TrimSpace(os.Getenv(integrationAIPromptProfileEnv)),
		PromptContext:              evalsupport.BuildPromptContext(strings.TrimSpace(os.Getenv(integrationAIPromptProfileEnv)), strings.TrimSpace(os.Getenv(integrationAIInstructionsRootEnv))),
		RunStatus:                  result.RunStatus,
		MetricStatus:               result.MetricStatus,
		FailureKind:                result.FailureKind,
		FailureSummary:             result.FailureSummary,
		FailureReason:              result.FailureReason,
		ResultClass:                resultClass,
		ToolNames:                  append([]string(nil), toolNames...),
		ToolCalls:                  outCalls,
		ToolErrorCount:             toolErrors,
		ReferenceSearchCount:       referenceSearches,
		ReferenceReadCount:         referenceReads,
		UnexpectedReferenceLookups: unexpectedReferenceLookupCount(recorder.scenario, referenceSearches, referenceReads),
		OutputText:                 result.OutputText,
		MemoryContent:              result.MemoryContent,
		SkillsReadOnly:             result.SkillsReadOnly,
		Interaction: evalsupport.InteractionSummary{
			ActiveSceneID:    activeSceneID(result.InteractionState),
			PlayerPhaseOpen:  playerPhaseOpen(result.InteractionState),
			CurrentTitle:     currentInteractionTitle(result.InteractionState),
			CurrentBeatTypes: currentInteractionBeatTypes(result.InteractionState),
			PromptText:       currentPromptBeat(result.InteractionState),
		},
		CharacterState: characterState,
		Artifacts: evalsupport.ArtifactPaths{
			RawCapture:     rawPath,
			MarkdownReport: reportPath,
			Summary:        summaryPath,
			Diagnostics:    diagnosticsPath,
		},
	}
}

func currentInteractionBeatTypes(state *gamev1.InteractionState) []string {
	if state == nil {
		return nil
	}
	beats := state.GetActiveScene().GetCurrentInteraction().GetBeats()
	out := make([]string, 0, len(beats))
	for _, beat := range beats {
		out = append(out, normalizeBeatType(beat.GetType()))
	}
	return out
}

func currentInteractionTitle(state *gamev1.InteractionState) string {
	if state == nil {
		return ""
	}
	return strings.TrimSpace(state.GetActiveScene().GetCurrentInteraction().GetTitle())
}

func normalizeBeatType(beatType gamev1.GMInteractionBeatType) string {
	switch beatType {
	case gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_FICTION:
		return "fiction"
	case gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_RESOLUTION:
		return "resolution"
	case gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_CONSEQUENCE:
		return "consequence"
	case gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_GUIDANCE:
		return "guidance"
	case gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT:
		return "prompt"
	default:
		return strings.ToLower(strings.TrimSpace(beatType.String()))
	}
}

func interactionBeatText(raw any, beatType string) string {
	interaction, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	beats, ok := interaction["beats"].([]any)
	if !ok {
		return ""
	}
	lastNonEmpty := ""
	for _, entry := range beats {
		beat, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		text := asString(beat["text"])
		if strings.TrimSpace(text) != "" {
			lastNonEmpty = text
		}
		if strings.EqualFold(asString(beat["type"]), beatType) {
			return text
		}
	}
	return lastNonEmpty
}

func interactionBeatTypes(raw any) []string {
	interaction, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	beats, ok := interaction["beats"].([]any)
	if !ok {
		return nil
	}
	types := make([]string, 0, len(beats))
	for _, entry := range beats {
		beat, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		beatType := strings.TrimSpace(asString(beat["type"]))
		if beatType == "" {
			continue
		}
		types = append(types, beatType)
	}
	return types
}

func flattenReplayToolCalls(fixture openAIReplayFixture) []openAIReplayToolCall {
	calls := make([]openAIReplayToolCall, 0)
	for _, step := range fixture.Steps {
		calls = append(calls, step.ToolCalls...)
	}
	return calls
}

func assertReplayToolOrder(t *testing.T, calls []openAIReplayToolCall, names ...string) {
	t.Helper()
	searchFrom := 0
	for _, want := range names {
		found := false
		for i := searchFrom; i < len(calls); i++ {
			if strings.TrimSpace(calls[i].Name) != want {
				continue
			}
			searchFrom = i + 1
			found = true
			break
		}
		if !found {
			t.Fatalf("missing ordered tool call %q in fixture sequence %v", want, replayToolNames(calls))
		}
	}
}

func nthReplayToolCallByName(t *testing.T, calls []openAIReplayToolCall, name string, ordinal int) openAIReplayToolCall {
	t.Helper()
	if ordinal < 1 {
		t.Fatalf("ordinal must be >= 1, got %d", ordinal)
	}
	count := 0
	for _, call := range calls {
		if strings.TrimSpace(call.Name) != name {
			continue
		}
		count++
		if count == ordinal {
			return call
		}
	}
	t.Fatalf("tool %q occurrence %d not found in fixture sequence %v", name, ordinal, replayToolNames(calls))
	return openAIReplayToolCall{}
}

func replayToolCallCount(calls []openAIReplayToolCall, name string) int {
	count := 0
	for _, call := range calls {
		if strings.TrimSpace(call.Name) == name {
			count++
		}
	}
	return count
}

func replayToolNames(calls []openAIReplayToolCall) []string {
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		names = append(names, strings.TrimSpace(call.Name))
	}
	return names
}

func replayNumericArgument(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	default:
		return 0
	}
}
