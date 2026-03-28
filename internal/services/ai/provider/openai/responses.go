package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

type openAIResponsesPayload struct {
	ID         string `json:"id"`
	OutputText string `json:"output_text"`
	Usage      struct {
		InputTokens        int32 `json:"input_tokens"`
		OutputTokens       int32 `json:"output_tokens"`
		TotalTokens        int32 `json:"total_tokens"`
		OutputTokenDetails struct {
			ReasoningTokens int32 `json:"reasoning_tokens"`
		} `json:"output_tokens_details"`
	} `json:"usage"`
	Output []struct {
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

func (a *InvokeAdapter) invokeResponsesAPI(ctx context.Context, input provider.InvokeInput) (provider.InvokeResult, error) {
	requestPayload := map[string]any{
		"model": input.Model,
		"input": input.Input,
	}
	if effort := strings.TrimSpace(input.ReasoningEffort); effort != "" {
		requestPayload["reasoning"] = map[string]any{
			"effort": effort,
		}
	}
	if instructions := strings.TrimSpace(input.Instructions); instructions != "" {
		requestPayload["instructions"] = instructions
	}
	payload, err := a.responsesRequest(ctx, requestPayload, input.AuthToken)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	outputText := strings.TrimSpace(payload.OutputText)
	if outputText == "" {
		for _, item := range payload.Output {
			for _, content := range item.Content {
				if strings.TrimSpace(content.Text) != "" {
					outputText = strings.TrimSpace(content.Text)
					break
				}
			}
			if outputText != "" {
				break
			}
		}
	}
	if outputText == "" {
		return provider.InvokeResult{}, fmt.Errorf("invoke response missing output text")
	}
	return provider.InvokeResult{
		OutputText: outputText,
		Usage:      openAIUsageFromPayload(payload),
	}, nil
}

func (a *InvokeAdapter) responsesRequest(ctx context.Context, body map[string]any, authToken string) (openAIResponsesPayload, error) {
	requestBody, err := json.Marshal(body)
	if err != nil {
		return openAIResponsesPayload{}, fmt.Errorf("marshal invoke request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(a.cfg.ResponsesURL), bytes.NewReader(requestBody))
	if err != nil {
		return openAIResponsesPayload{}, fmt.Errorf("build invoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(authToken))

	res, err := a.cfg.HTTPClient.Do(req)
	if err != nil {
		return openAIResponsesPayload{}, fmt.Errorf("invoke request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return openAIResponsesPayload{}, fmt.Errorf("read invoke error body: %w", err)
		}
		return openAIResponsesPayload{}, fmt.Errorf("invoke request status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload openAIResponsesPayload
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return openAIResponsesPayload{}, fmt.Errorf("decode invoke response: %w", err)
	}
	return payload, nil
}

func openAIUsageFromPayload(payload openAIResponsesPayload) provider.Usage {
	return provider.Usage{
		InputTokens:     payload.Usage.InputTokens,
		OutputTokens:    payload.Usage.OutputTokens,
		ReasoningTokens: payload.Usage.OutputTokenDetails.ReasoningTokens,
		TotalTokens:     payload.Usage.TotalTokens,
	}
}
