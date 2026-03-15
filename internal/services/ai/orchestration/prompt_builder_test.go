package orchestration

import (
	"context"
	"strings"
	"testing"
)

func TestPromptBuilderUsesBootstrapModeWithoutActiveScene(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "")}
	sess.resources["campaign://camp-1/artifacts/memory.md"] = "Session memory."
	sess.resources["campaign://camp-1/sessions/sess-1/scenes"] = `{"scenes":[]}`

	prompt, err := newDefaultPromptBuilder().Build(context.Background(), sess, Input{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !strings.Contains(prompt, "Bootstrap mode: there is no active scene yet.") {
		t.Fatalf("prompt missing bootstrap instructions: %q", prompt)
	}
}
