//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
)

type openAIReplayServer struct {
	fixture openAIReplayFixture

	mu           sync.Mutex
	step         int
	campaignID   string
	sessionID    string
	characterID  string
	sceneID      string
	firstErr     error
	callOutputs  map[string]string
	requestDebug []string
}

// TestAIGMCampaignContextReplayBootstrap keeps the committed fixture exercising the real orchestration tool loop.
func TestAIGMCampaignContextReplayBootstrap(t *testing.T) {
	replay := loadOpenAIReplayFixture(t, "ai_gm_campaign_context_bootstrap_replay.json")
	replayServer := &openAIReplayServer{fixture: replay}
	httpServer := httptest.NewServer(replayServer)
	t.Cleanup(httpServer.Close)

	var result aiGMBootstrapResult
	result = runAIGMCampaignContextBootstrapScenario(t, aiGMBootstrapScenarioOptions{
		ResponsesURL:     httpServer.URL,
		Model:            "gpt-4.1-mini",
		CredentialSecret: "test-openai-token",
		AgentLabel:       "replay-gm",
		BeforeRun: func(setup aiGMBootstrapSetup) {
			replayServer.campaignID = setup.CampaignID
			replayServer.sessionID = setup.SessionID
			replayServer.characterID = setup.CharacterID
		},
	})
	if got := strings.TrimSpace(result.OutputText); got != replayFixtureFinalOutputText(t, replay) {
		t.Fatalf("output_text = %q, want %q", got, replayFixtureFinalOutputText(t, replay))
	}
	if got := strings.TrimSpace(result.MemoryContent); got != replayFixtureMemoryContent(t, replay) {
		t.Fatalf("memory.md = %q, want %q", got, replayFixtureMemoryContent(t, replay))
	}
	if !result.SkillsReadOnly {
		t.Fatal("expected skills.md to be read-only")
	}
	if result.SceneCount != 1 {
		t.Fatalf("len(scenes) = %d, want 1", result.SceneCount)
	}
	if !result.SceneIsActive {
		t.Fatal("expected created scene to be active")
	}
	if !result.PlayerPhaseOpen {
		t.Fatal("expected bootstrap replay to leave the first player phase open")
	}
	if err := replayServer.Err(); err != nil {
		t.Fatalf("openai replay server: %v\nreplay outputs:\n%s", err, replayServer.DebugString())
	}
	if got, want := replayServer.StepCount(), len(replay.Steps); got != want {
		t.Fatalf("replay step count = %d, want %d", got, want)
	}
}

// ServeHTTP emulates the minimal Responses API surface needed by the deterministic replay lane.
func (s *openAIReplayServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && strings.HasSuffix(strings.TrimSpace(r.URL.Path), "/models") {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1-mini","owned_by":"openai","created":0}]}`))
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.setErr(fmt.Errorf("read request body: %w", err))
		http.Error(w, "read request body", http.StatusInternalServerError)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		s.setErr(fmt.Errorf("parse request body: %w", err))
		http.Error(w, "parse request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.step == 0 {
		if err := s.assertInitialRequest(payload); err != nil {
			s.firstErr = err
		}
	}
	s.captureRequestMetadata(payload)
	s.captureCallOutputs(payload)
	if sceneID := extractSceneID(payload); sceneID != "" {
		s.sceneID = sceneID
	}
	if s.step >= len(s.fixture.Steps) {
		s.firstErr = fmt.Errorf("unexpected extra replay request %d", s.step)
		http.Error(w, "unexpected extra replay request\n"+s.debugStringLocked(), http.StatusInternalServerError)
		return
	}
	step := s.fixture.Steps[s.step]
	s.step++

	responseBody, err := json.Marshal(buildOpenAIReplayResponse(step, map[string]string{
		"campaign_id":  s.campaignID,
		"session_id":   s.sessionID,
		"character_id": s.characterID,
		"scene_id":     s.sceneID,
	}))
	if err != nil {
		s.firstErr = fmt.Errorf("marshal replay response: %w", err)
		http.Error(w, "marshal replay response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(responseBody)
}

func (s *openAIReplayServer) captureRequestMetadata(payload map[string]any) {
	previousID := strings.TrimSpace(asString(payload["previous_response_id"]))
	inputItems, _ := payload["input"].([]any)
	functionOutputs := 0
	var followUpText string
	for _, raw := range inputItems {
		item, _ := raw.(map[string]any)
		switch strings.TrimSpace(asString(item["type"])) {
		case "function_call_output":
			functionOutputs++
		}
		if strings.TrimSpace(asString(item["role"])) != "user" {
			continue
		}
		contentItems, _ := item["content"].([]any)
		for _, rawContent := range contentItems {
			content, _ := rawContent.(map[string]any)
			if strings.TrimSpace(asString(content["type"])) != "input_text" {
				continue
			}
			followUpText = strings.TrimSpace(asString(content["text"]))
			break
		}
	}
	if previousID == "" {
		s.requestDebug = append(s.requestDebug, fmt.Sprintf("request step=%d previous=%s function_outputs=%d initial_prompt=%t", s.step, previousID, functionOutputs, strings.TrimSpace(followUpText) != ""))
		return
	}
	if len(followUpText) > 120 {
		followUpText = followUpText[:120] + "..."
	}
	s.requestDebug = append(s.requestDebug, fmt.Sprintf("request step=%d previous=%s function_outputs=%d follow_up=%q", s.step, previousID, functionOutputs, followUpText))
}

// assertInitialRequest locks the replay fixture to the expected prompt contract and tool allowlist.
func (s *openAIReplayServer) assertInitialRequest(payload map[string]any) error {
	prompt, toolNames := extractPromptAndToolNames(payload)
	for _, expected := range s.fixture.InitialPromptContains {
		if !strings.Contains(prompt, expected) {
			return fmt.Errorf("initial prompt missing %q", expected)
		}
	}
	for _, expected := range s.fixture.InitialToolNames {
		if !slices.Contains(toolNames, expected) {
			return fmt.Errorf("initial tool list missing %q", expected)
		}
	}
	return nil
}

// setErr keeps the first replay failure for clearer debugging.
func (s *openAIReplayServer) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.firstErr == nil {
		s.firstErr = err
	}
}

// Err returns the first replay-server assertion or transport error.
func (s *openAIReplayServer) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.firstErr
}

// StepCount reports how many recorded provider responses were consumed by the runner.
func (s *openAIReplayServer) StepCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.step
}

// DebugString exposes captured tool outputs when replay diverges from the fixture.
func (s *openAIReplayServer) DebugString() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.debugStringLocked()
}

func (s *openAIReplayServer) debugStringLocked() string {
	if len(s.callOutputs) == 0 && len(s.requestDebug) == 0 {
		return "(no replay call outputs captured)"
	}
	lines := make([]string, 0, len(s.requestDebug)+len(s.callOutputs))
	lines = append(lines, s.requestDebug...)
	if len(s.callOutputs) != 0 {
		callIDs := make([]string, 0, len(s.callOutputs))
		for callID := range s.callOutputs {
			callIDs = append(callIDs, callID)
		}
		slices.Sort(callIDs)
		for _, callID := range callIDs {
			lines = append(lines, fmt.Sprintf("%s => %s", callID, s.callOutputs[callID]))
		}
	}
	return strings.Join(lines, "\n")
}

// captureCallOutputs records function-call outputs so replay failures show the full tool loop context.
func (s *openAIReplayServer) captureCallOutputs(payload map[string]any) {
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
		if s.callOutputs == nil {
			s.callOutputs = map[string]string{}
		}
		s.callOutputs[callID] = output
		if len(output) > 200 {
			output = output[:200] + "..."
		}
		s.requestDebug = append(s.requestDebug, fmt.Sprintf("step=%d call_id=%s output=%s", s.step, callID, output))
	}
}

// extractPromptAndToolNames pulls the prompt and advertised tools from the initial Responses request.
func extractPromptAndToolNames(payload map[string]any) (string, []string) {
	var prompt string
	inputItems, _ := payload["input"].([]any)
	if len(inputItems) != 0 {
		firstInput, _ := inputItems[0].(map[string]any)
		contentItems, _ := firstInput["content"].([]any)
		if len(contentItems) != 0 {
			firstContent, _ := contentItems[0].(map[string]any)
			prompt, _ = firstContent["text"].(string)
		}
	}
	toolItems, _ := payload["tools"].([]any)
	names := make([]string, 0, len(toolItems))
	for _, raw := range toolItems {
		item, _ := raw.(map[string]any)
		name, _ := item["name"].(string)
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return prompt, names
}

func extractSceneID(payload map[string]any) string {
	inputItems, _ := payload["input"].([]any)
	for _, raw := range inputItems {
		item, _ := raw.(map[string]any)
		if strings.TrimSpace(asString(item["type"])) != "function_call_output" {
			continue
		}
		outputText, _ := item["output"].(string)
		var output map[string]any
		if err := json.Unmarshal([]byte(outputText), &output); err != nil {
			continue
		}
		if sceneID := strings.TrimSpace(asString(output["scene_id"])); sceneID != "" {
			return sceneID
		}
	}
	return ""
}
