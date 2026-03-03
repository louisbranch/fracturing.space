package campaign

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestFoldCampaignCreatedSetsCreated(t *testing.T) {
	state := State{}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("campaign.created"),
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.Created {
		t.Fatal("expected state to be marked created")
	}
	if updated.Locale != "en-US" {
		t.Fatalf("locale = %s, want %s", updated.Locale, "en-US")
	}
}

func TestFoldCampaignCreatedSetsFields(t *testing.T) {
	state := State{}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("campaign.created"),
		PayloadJSON: []byte(`{"name":"Sunfall","locale":"en-US","game_system":"daggerheart","gm_mode":"human","cover_asset_id":"camp-cover-03"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Sunfall" {
		t.Fatalf("name = %s, want %s", updated.Name, "Sunfall")
	}
	if updated.GameSystem != "daggerheart" {
		t.Fatalf("game system = %s, want %s", updated.GameSystem, "daggerheart")
	}
	if updated.Locale != "en-US" {
		t.Fatalf("locale = %s, want %s", updated.Locale, "en-US")
	}
	if updated.GmMode != "human" {
		t.Fatalf("gm mode = %s, want %s", updated.GmMode, "human")
	}
	if updated.CoverAssetID != "camp-cover-03" {
		t.Fatalf("cover asset id = %s, want %s", updated.CoverAssetID, "camp-cover-03")
	}
}

func TestFoldCampaignCreated_ReturnsErrorOnCorruptPayload(t *testing.T) {
	state := State{}
	_, err := Fold(state, event.Event{
		Type:        EventTypeCreated,
		PayloadJSON: []byte(`{corrupt`),
	})
	if err == nil {
		t.Fatal("expected error for corrupt payload")
	}
}

func TestFoldCampaignUpdatedSetsFields(t *testing.T) {
	state := State{Created: true, Status: StatusDraft, Name: "Old", Locale: "en-US", ThemePrompt: "Old theme", CoverAssetID: "camp-cover-01"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("campaign.updated"),
		PayloadJSON: []byte(`{"fields":{"name":"Sunfall","status":"active","theme_prompt":"New theme","locale":"pt-BR","cover_asset_id":"camp-cover-04"}}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Sunfall" {
		t.Fatalf("name = %s, want %s", updated.Name, "Sunfall")
	}
	if updated.Status != StatusActive {
		t.Fatalf("status = %s, want %s", updated.Status, StatusActive)
	}
	if updated.ThemePrompt != "New theme" {
		t.Fatalf("theme prompt = %s, want %s", updated.ThemePrompt, "New theme")
	}
	if updated.Locale != "pt-BR" {
		t.Fatalf("locale = %s, want %s", updated.Locale, "pt-BR")
	}
	if updated.CoverAssetID != "camp-cover-04" {
		t.Fatalf("cover asset id = %s, want %s", updated.CoverAssetID, "camp-cover-04")
	}
}

func TestFoldCampaignUpdatedSetsCoverSetID(t *testing.T) {
	state := State{Created: true, CoverSetID: "old-cover-set"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("campaign.updated"),
		PayloadJSON: []byte(`{"fields":{"cover_set_id":"  campaign-covers-v1  "}}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.CoverSetID != "campaign-covers-v1" {
		t.Fatalf("cover set id = %s, want %s", updated.CoverSetID, "campaign-covers-v1")
	}
}

func TestFoldCampaignUpdated_ReturnsErrorOnCorruptPayload(t *testing.T) {
	_, err := Fold(State{}, event.Event{
		Type:        EventTypeUpdated,
		PayloadJSON: []byte(`{`),
	})
	if err == nil {
		t.Fatal("expected error for corrupt update payload")
	}
}

func TestFoldCampaignAIBoundSetsAgentID(t *testing.T) {
	updated, err := Fold(State{}, event.Event{
		Type:        EventTypeAIBound,
		PayloadJSON: []byte(`{"ai_agent_id":"  agent-1  "}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.AIAgentID != "agent-1" {
		t.Fatalf("ai agent id = %q, want %q", updated.AIAgentID, "agent-1")
	}
}

func TestFoldCampaignAIBound_ReturnsErrorOnCorruptPayload(t *testing.T) {
	_, err := Fold(State{}, event.Event{
		Type:        EventTypeAIBound,
		PayloadJSON: []byte(`{`),
	})
	if err == nil {
		t.Fatal("expected error for corrupt ai_bound payload")
	}
}

func TestFoldCampaignAIUnboundClearsAgentID(t *testing.T) {
	updated, err := Fold(State{AIAgentID: "agent-1"}, event.Event{
		Type:        EventTypeAIUnbound,
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.AIAgentID != "" {
		t.Fatalf("ai agent id = %q, want empty", updated.AIAgentID)
	}
}

func TestFoldCampaignAIAuthRotatedSetsEpoch(t *testing.T) {
	updated, err := Fold(State{AIAuthEpoch: 1}, event.Event{
		Type:        EventTypeAIAuthRotated,
		PayloadJSON: []byte(`{"epoch_after":5,"reason":"rotate"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.AIAuthEpoch != 5 {
		t.Fatalf("ai auth epoch = %d, want %d", updated.AIAuthEpoch, 5)
	}
}

func TestFoldCampaignAIAuthRotated_ReturnsErrorOnCorruptPayload(t *testing.T) {
	_, err := Fold(State{}, event.Event{
		Type:        EventTypeAIAuthRotated,
		PayloadJSON: []byte(`{`),
	})
	if err == nil {
		t.Fatal("expected error for corrupt ai_auth_rotated payload")
	}
}

func TestFoldCampaignForked_NoStateMutation(t *testing.T) {
	initial := State{
		Created:     true,
		Name:        "Sunfall",
		AIAgentID:   "agent-1",
		AIAuthEpoch: 3,
	}
	updated, err := Fold(initial, event.Event{
		Type:        EventTypeForked,
		PayloadJSON: []byte(`{"parent_campaign_id":"camp-0","origin_campaign_id":"camp-root"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated != initial {
		t.Fatalf("fork event mutated state: got %#v, want %#v", updated, initial)
	}
}
