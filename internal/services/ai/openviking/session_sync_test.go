package openviking

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

type sessionClientStub struct {
	sessionID  string
	messages   []AddMessageInput
	used       []UsedInput
	commits    int
	commitWait []bool
}

func (s *sessionClientStub) AddMessage(_ context.Context, sessionID string, input AddMessageInput) error {
	s.sessionID = sessionID
	s.messages = append(s.messages, input)
	return nil
}

func (s *sessionClientStub) Used(_ context.Context, _ string, input UsedInput) error {
	s.used = append(s.used, input)
	return nil
}

func (s *sessionClientStub) Commit(_ context.Context, _ string, wait bool) (CommitResult, error) {
	s.commits++
	s.commitWait = append(s.commitWait, wait)
	return CommitResult{Status: "committed"}, nil
}

func TestSessionSyncWritesTurnAndCommits(t *testing.T) {
	client := &sessionClientStub{}
	syncer, err := NewSessionSync(client, ModeLegacy)
	if err != nil {
		t.Fatalf("NewSessionSync() error = %v", err)
	}

	err = syncer.SyncTurn(context.Background(), TurnSyncInput{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		ParticipantID: "gm-1",
		UserText:      "Start the scene.",
		AssistantText: "The harbor wakes under steel-gray skies.",
		RetrievedContexts: []orchestration.RetrievedContext{{
			URI:         "viking://resources/fracturing-space/campaigns/camp-1/story.md",
			ContextType: "resource",
			Abstract:    "Storm buildup at the harbor.",
		}},
	})
	if err != nil {
		t.Fatalf("SyncTurn() error = %v", err)
	}
	if client.sessionID != StableSessionID("camp-1", "sess-1", "gm-1") {
		t.Fatalf("session ID = %q", client.sessionID)
	}
	if len(client.messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(client.messages))
	}
	if client.messages[0].Role != "user" || client.messages[0].Content != "Start the scene." {
		t.Fatalf("user message = %#v", client.messages[0])
	}
	if client.messages[1].Role != "assistant" || len(client.messages[1].Parts) != 2 {
		t.Fatalf("assistant message = %#v", client.messages[1])
	}
	if client.commits != 1 {
		t.Fatalf("commits = %d, want 1", client.commits)
	}
	if len(client.commitWait) != 1 || !client.commitWait[0] {
		t.Fatalf("commit waits = %#v, want [true]", client.commitWait)
	}
	if len(client.used) != 0 {
		t.Fatalf("used calls = %#v, want none in legacy mode", client.used)
	}
}

func TestSessionSyncRecordsUsedContextsInDocsAlignedMode(t *testing.T) {
	client := &sessionClientStub{}
	syncer, err := NewSessionSync(client, ModeDocsAlignedSupplement)
	if err != nil {
		t.Fatalf("NewSessionSync() error = %v", err)
	}

	err = syncer.SyncTurn(context.Background(), TurnSyncInput{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		ParticipantID: "gm-1",
		UserText:      "Start the scene.",
		AssistantText: "The harbor wakes under steel-gray skies.",
		RetrievedContexts: []orchestration.RetrievedContext{
			{URI: "viking://resources/fracturing-space/campaigns/camp-1/story.md", ContextType: "resource"},
			{URI: "viking://resources/fracturing-space/campaigns/camp-1/story.md", ContextType: "resource"},
			{URI: "viking://user/memories/events/floodgate", ContextType: "memory"},
		},
	})
	if err != nil {
		t.Fatalf("SyncTurn() error = %v", err)
	}
	if len(client.used) != 1 {
		t.Fatalf("used calls = %#v, want 1", client.used)
	}
	if got := client.used[0].Contexts; len(got) != 2 {
		t.Fatalf("used contexts = %#v, want 2 unique URIs", got)
	}
}
