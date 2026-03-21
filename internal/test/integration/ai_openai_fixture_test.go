package integration

import (
	"reflect"
	"testing"
)

// TestBuildOpenAIReplayResponseAppliesTokens protects placeholder substitution in served replay tool calls.
func TestBuildOpenAIReplayResponseAppliesTokens(t *testing.T) {
	response := buildOpenAIReplayResponse(openAIReplayStep{
		ID: "resp-1",
		ToolCalls: []openAIReplayToolCall{{
			CallID: "call-1",
			Name:   "scene_create",
			Arguments: map[string]any{
				"character_ids": []any{"{{character_id}}"},
				"scene_id":      "{{scene_id}}",
			},
		}},
	}, map[string]string{
		"character_id": "char-1",
		"scene_id":     "scene-1",
	})

	output, ok := response["output"].([]map[string]any)
	if !ok || len(output) != 1 {
		t.Fatalf("output = %#v", response["output"])
	}
	if got := output[0]["arguments"]; got != `{"character_ids":["char-1"],"scene_id":"scene-1"}` {
		t.Fatalf("arguments = %v", got)
	}
}

// TestTokenizeReplayFixtureReplacesDynamicIDs protects the live-capture to committed-fixture normalization step.
func TestTokenizeReplayFixtureReplacesDynamicIDs(t *testing.T) {
	fixture := openAIReplayFixture{
		Steps: []openAIReplayStep{{
			ID:         "resp-1",
			OutputText: "Scene scene-1 is ready for char-1.",
			ToolCalls: []openAIReplayToolCall{{
				CallID: "call-1",
				Name:   "interaction_activate_scene",
				Arguments: map[string]any{
					"scene_id": "scene-1",
					"actors":   []any{"char-1"},
				},
			}},
		}},
	}

	got := tokenizeReplayFixture(fixture, map[string]string{
		"character_id": "char-1",
		"scene_id":     "scene-1",
	})
	want := openAIReplayFixture{
		Steps: []openAIReplayStep{{
			ID:         "resp-1",
			OutputText: "Scene {{scene_id}} is ready for {{character_id}}.",
			ToolCalls: []openAIReplayToolCall{{
				CallID: "call-1",
				Name:   "interaction_activate_scene",
				Arguments: map[string]any{
					"scene_id": "{{scene_id}}",
					"actors":   []any{"{{character_id}}"},
				},
			}},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokenizeReplayFixture() = %#v, want %#v", got, want)
	}
}
