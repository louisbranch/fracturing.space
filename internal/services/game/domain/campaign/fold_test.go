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
}

func TestFoldCampaignCreatedSetsFields(t *testing.T) {
	state := State{}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("campaign.created"),
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"daggerheart","gm_mode":"human","cover_asset_id":"camp-cover-03"}`),
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
	state := State{Created: true, Status: StatusDraft, Name: "Old", ThemePrompt: "Old theme", CoverAssetID: "camp-cover-01"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("campaign.updated"),
		PayloadJSON: []byte(`{"fields":{"name":"Sunfall","status":"active","theme_prompt":"New theme","cover_asset_id":"camp-cover-04"}}`),
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
	if updated.CoverAssetID != "camp-cover-04" {
		t.Fatalf("cover asset id = %s, want %s", updated.CoverAssetID, "camp-cover-04")
	}
}
