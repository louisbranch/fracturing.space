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
	"strings"
	"sync"
	"testing"
	"time"
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

	mu           sync.Mutex
	firstErr     error
	initialTools []string
	steps        []openAIReplayStep
	rawCapture   openAILiveCapture
	requestDebug []string
	lastSceneID  string
}

// TestAIGMCampaignContextLiveCaptureBootstrap proves a real model can complete the GM bootstrap tool loop.
func TestAIGMCampaignContextLiveCaptureBootstrap(t *testing.T) {
	apiKey := strings.TrimSpace(os.Getenv(integrationOpenAIAPIKeyEnv))
	if apiKey == "" {
		t.Skipf("%s is required", integrationOpenAIAPIKeyEnv)
	}
	model := liveAIModel()
	reasoningEffort := liveAIReasoningEffort()
	recorder := &openAILiveRecorder{
		targetURL: liveOpenAIResponsesTargetURL(),
		client:    newHTTPClient(t),
		model:     model,
		rawCapture: openAILiveCapture{
			Metadata: openAIReplayMetadata{
				Provider:        "openai",
				Model:           model,
				ReasoningEffort: reasoningEffort,
				Scenario:        aiGMBootstrapScenarioName,
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

	result := runAIGMCampaignContextBootstrapScenario(t, aiGMBootstrapScenarioOptions{
		ResponsesURL:     server.URL,
		Model:            model,
		ReasoningEffort:  reasoningEffort,
		CredentialSecret: apiKey,
		AgentLabel:       "live-capture-gm",
	})

	rawPath := writeOpenAILiveCapture(t, recorder.rawCapture)
	t.Logf("live capture written to %s", rawPath)
	reportPath := writeOpenAILiveCaptureReport(t, recorder, result)
	t.Logf("quality report written to %s", reportPath)

	if err := recorder.Err(); err != nil {
		t.Fatalf("live recorder: %v\nrequests:\n%s", err, recorder.DebugString())
	}
	if strings.TrimSpace(result.OutputText) == "" {
		t.Fatal("expected non-empty model output")
	}
	if strings.TrimSpace(result.MemoryContent) == "" || result.MemoryContent == aiGMBootstrapMemorySeed {
		t.Fatalf("memory.md = %q, expected updated memory content", result.MemoryContent)
	}
	if !result.SkillsReadOnly {
		t.Fatal("expected skills.md to remain read-only")
	}
	if result.SceneCount == 0 || strings.TrimSpace(result.ActiveSceneID) == "" || !result.SceneIsActive {
		t.Fatalf("scene bootstrap failed: count=%d active_scene_id=%q active=%v", result.SceneCount, result.ActiveSceneID, result.SceneIsActive)
	}
	if !result.PlayerPhaseOpen {
		t.Fatal("expected bootstrap to start the first player phase")
	}

	fixture := recorder.ReplayFixture(result.CampaignID, result.SessionID, result.CharacterID, result.ActiveSceneID)
	fixtureToolNames := openAIReplayFixtureToolNames(fixture)
	if err := requiredToolSetPresent(fixtureToolNames,
		"system_reference_search",
		"scene_create",
		"interaction_active_scene_set",
		"interaction_scene_gm_interaction_commit",
		"interaction_scene_player_phase_start",
	); err != nil {
		t.Fatalf("fixture tool coverage: %v", err)
	}
	// Accept either full-document upsert or section-level update as the memory write tool.
	if err := requiredToolSetPresent(fixtureToolNames, "campaign_artifact_upsert"); err != nil {
		if err := requiredToolSetPresent(fixtureToolNames, "campaign_memory_section_update"); err != nil {
			t.Fatal("fixture tool coverage: missing memory write tool (campaign_artifact_upsert or campaign_memory_section_update)")
		}
	}
	if envEnabled(integrationAIWriteFixtureEnv) {
		fixturePath := writeOpenAIReplayFixture(t, aiGMBootstrapFixtureFile, fixture)
		t.Logf("updated replay fixture at %s", fixturePath)
	}
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
		_, toolNames := extractPromptAndToolNames(requestPayload)
		r.initialTools = append([]string(nil), toolNames...)
	}
	r.captureCallOutputs(requestPayload)
	if sceneID := extractSceneID(requestPayload); sceneID != "" {
		r.lastSceneID = sceneID
	}
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
func (r *openAILiveRecorder) ReplayFixture(campaignID, sessionID, characterID, sceneID string) openAIReplayFixture {
	r.mu.Lock()
	defer r.mu.Unlock()
	fixture := openAIReplayFixture{
		Metadata: &openAIReplayMetadata{
			Provider:        "openai",
			Model:           r.model,
			ReasoningEffort: r.rawCapture.Metadata.ReasoningEffort,
			CapturedAtUTC:   r.rawCapture.Metadata.CapturedAtUTC,
			Scenario:        aiGMBootstrapScenarioName,
			Source:          "live_capture",
		},
		InitialPromptContains: bootstrapPromptContains(),
		InitialToolNames:      append([]string(nil), r.initialTools...),
		Steps:                 append([]openAIReplayStep(nil), r.steps...),
	}
	return tokenizeReplayFixture(fixture, map[string]string{
		"campaign_id":  campaignID,
		"session_id":   sessionID,
		"character_id": characterID,
		"scene_id":     sceneID,
	})
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

// writeOpenAILiveCapture persists the raw live capture outside the repo fixtures for local debugging and review.
func writeOpenAILiveCapture(t *testing.T, capture openAILiveCapture) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), ".tmp", "ai-live-captures")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create capture dir: %v", err)
	}
	filename := fmt.Sprintf("%s-%s.json", aiGMBootstrapScenarioName, time.Now().UTC().Format("20060102T150405Z"))
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
func writeOpenAILiveCaptureReport(t *testing.T, recorder *openAILiveRecorder, result aiGMBootstrapResult) string {
	t.Helper()
	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	usage := aggregateLiveCaptureUsage(recorder.rawCapture)
	model := recorder.model

	// Extract narrative fields from recorded steps.
	var sceneName, sceneDesc, gmNarration, playerFrame, memoryContent string
	var toolSequence []string
	var toolErrors []string
	for _, step := range recorder.steps {
		for _, call := range step.ToolCalls {
			toolSequence = append(toolSequence, call.Name)
			switch call.Name {
			case "scene_create":
				sceneName = asString(call.Arguments["name"])
				sceneDesc = asString(call.Arguments["description"])
			case "interaction_scene_gm_interaction_commit":
				gmNarration = interactionBeatText(call.Arguments["interaction"], "fiction")
			case "interaction_scene_player_phase_start":
				playerFrame = interactionBeatText(call.Arguments["interaction"], "prompt")
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
	fmt.Fprintf(&b, "- **Scenario:** %s\n\n", aiGMBootstrapScenarioName)

	fmt.Fprintf(&b, "## Token Usage\n\n")
	fmt.Fprintf(&b, "| Metric | Tokens |\n")
	fmt.Fprintf(&b, "|--------|-------:|\n")
	fmt.Fprintf(&b, "| Input | %d |\n", usage.InputTokens)
	fmt.Fprintf(&b, "| Output | %d |\n", usage.OutputTokens)
	fmt.Fprintf(&b, "| Reasoning | %d |\n", usage.ReasoningTokens)
	fmt.Fprintf(&b, "| Total | %d |\n\n", usage.TotalTokens)

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
	fmt.Fprintf(&b, "### GM Narration\n\n%s\n\n", gmNarration)
	fmt.Fprintf(&b, "### Player Phase Frame\n\n%s\n\n", playerFrame)
	fmt.Fprintf(&b, "### Memory Update\n\n%s\n", memoryContent)

	dir := filepath.Join(repoRoot(t), ".tmp", "ai-live-captures")
	filename := fmt.Sprintf("%s-%s-%s.md",
		aiGMBootstrapScenarioName,
		model,
		time.Now().UTC().Format("20060102T150405Z"),
	)
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write quality report: %v", err)
	}
	return path
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
