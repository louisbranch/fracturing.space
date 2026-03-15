package integration

import (
	"encoding/json"
	"strings"
)

// openAIReplayMetadata records the live-capture provenance without changing replay behavior.
type openAIReplayMetadata struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	CapturedAtUTC   string `json:"captured_at_utc"`
	Scenario        string `json:"scenario"`
	Source          string `json:"source"`
}

// openAIReplayFixture keeps the committed fixture human-reviewable and independent of raw provider JSON.
type openAIReplayFixture struct {
	Metadata              *openAIReplayMetadata `json:"metadata,omitempty"`
	InitialPromptContains []string              `json:"initial_prompt_contains"`
	InitialToolNames      []string              `json:"initial_tool_names"`
	Steps                 []openAIReplayStep    `json:"steps"`
}

// openAIReplayStep models one provider response in the tool loop.
type openAIReplayStep struct {
	ID         string                 `json:"id"`
	OutputText string                 `json:"output_text,omitempty"`
	ToolCalls  []openAIReplayToolCall `json:"tool_calls,omitempty"`
}

// openAIReplayToolCall captures the minimal tool-call contract needed for deterministic replay.
type openAIReplayToolCall struct {
	CallID    string         `json:"call_id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// openAIResponsesPayload decodes only the response fields needed to convert live captures into replay steps.
type openAIResponsesPayload struct {
	ID         string `json:"id"`
	OutputText string `json:"output_text"`
	Output     []struct {
		Type      string `json:"type"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
		CallID    string `json:"call_id"`
		Content   []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

// buildOpenAIReplayResponse reconstructs the subset of the Responses API payload used by the replay server.
func buildOpenAIReplayResponse(step openAIReplayStep, tokens map[string]string) map[string]any {
	output := make([]map[string]any, 0, len(step.ToolCalls))
	for _, call := range step.ToolCalls {
		argsJSON, _ := json.Marshal(replaceReplayTokens(call.Arguments, tokens))
		output = append(output, map[string]any{
			"type":      "function_call",
			"name":      call.Name,
			"call_id":   call.CallID,
			"arguments": string(argsJSON),
		})
	}
	if len(output) == 0 && strings.TrimSpace(step.OutputText) != "" {
		output = append(output, map[string]any{
			"type": "message",
			"content": []map[string]any{{
				"type": "output_text",
				"text": step.OutputText,
			}},
		})
	}
	return map[string]any{
		"id":          step.ID,
		"output_text": step.OutputText,
		"output":      output,
	}
}

// replaceReplayTokens resolves placeholder values when a replay step is served back to the adapter.
func replaceReplayTokens(value any, tokens map[string]string) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = replaceReplayTokens(item, tokens)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, replaceReplayTokens(item, tokens))
		}
		return out
	case string:
		result := typed
		for key, replacement := range tokens {
			result = strings.ReplaceAll(result, "{{"+key+"}}", replacement)
		}
		return result
	default:
		return value
	}
}

// tokenizeReplayFixture strips run-specific IDs from a live capture before it becomes a committed fixture.
func tokenizeReplayFixture(fixture openAIReplayFixture, tokens map[string]string) openAIReplayFixture {
	tokenized := fixture
	tokenized.Steps = make([]openAIReplayStep, 0, len(fixture.Steps))
	for _, step := range fixture.Steps {
		tokenized.Steps = append(tokenized.Steps, tokenizeReplayStep(step, tokens))
	}
	return tokenized
}

// tokenizeReplayStep applies fixture tokenization to one recorded provider response.
func tokenizeReplayStep(step openAIReplayStep, tokens map[string]string) openAIReplayStep {
	tokenized := step
	tokenized.OutputText = tokenizeReplayString(step.OutputText, tokens)
	tokenized.ToolCalls = make([]openAIReplayToolCall, 0, len(step.ToolCalls))
	for _, call := range step.ToolCalls {
		tokenized.ToolCalls = append(tokenized.ToolCalls, openAIReplayToolCall{
			CallID:    call.CallID,
			Name:      call.Name,
			Arguments: tokenizeReplayValue(call.Arguments, tokens).(map[string]any),
		})
	}
	return tokenized
}

// tokenizeReplayValue walks nested arguments so dynamic IDs are replaced consistently.
func tokenizeReplayValue(value any, tokens map[string]string) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = tokenizeReplayValue(item, tokens)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, tokenizeReplayValue(item, tokens))
		}
		return out
	case string:
		return tokenizeReplayString(typed, tokens)
	default:
		return value
	}
}

// tokenizeReplayString turns concrete IDs back into stable fixture tokens.
func tokenizeReplayString(value string, tokens map[string]string) string {
	result := value
	for key, actual := range tokens {
		if strings.TrimSpace(actual) == "" {
			continue
		}
		result = strings.ReplaceAll(result, actual, "{{"+key+"}}")
	}
	return result
}

// asString normalizes loose JSON values into the trimmed strings used by the recorder assertions.
func asString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
