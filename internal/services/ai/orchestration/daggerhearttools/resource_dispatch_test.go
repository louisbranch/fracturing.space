package daggerhearttools

import (
	"context"
	"strings"
	"testing"
)

func TestReadResourceReturnsUnhandledForUnknownURI(t *testing.T) {
	value, handled, err := ReadResource(nil, context.Background(), "mystery://unsupported")
	if err != nil {
		t.Fatalf("ReadResource() error = %v", err)
	}
	if handled {
		t.Fatal("handled = true, want false")
	}
	if value != "" {
		t.Fatalf("value = %q, want empty", value)
	}
}

func TestParseSnapshotResourceURI(t *testing.T) {
	campaignID, err := parseSnapshotResourceURI("daggerheart://campaign/camp-1/snapshot")
	if err != nil {
		t.Fatalf("parseSnapshotResourceURI() error = %v", err)
	}
	if campaignID != "camp-1" {
		t.Fatalf("campaign_id = %q, want camp-1", campaignID)
	}

	_, err = parseSnapshotResourceURI("daggerheart://campaign//snapshot")
	if err == nil || !strings.Contains(err.Error(), "campaign ID is required") {
		t.Fatalf("parseSnapshotResourceURI() error = %v", err)
	}
}

func TestParseCombatBoardResourceURI(t *testing.T) {
	campaignID, sessionID, err := parseCombatBoardResourceURI("daggerheart://campaign/camp-1/sessions/sess-1/combat_board")
	if err != nil {
		t.Fatalf("parseCombatBoardResourceURI() error = %v", err)
	}
	if campaignID != "camp-1" || sessionID != "sess-1" {
		t.Fatalf("parseCombatBoardResourceURI() = (%q, %q)", campaignID, sessionID)
	}

	_, _, err = parseCombatBoardResourceURI("daggerheart://campaign/camp-1/sessions//combat_board")
	if err == nil || !strings.Contains(err.Error(), "campaign and session IDs are required") {
		t.Fatalf("parseCombatBoardResourceURI() error = %v", err)
	}
}

func TestParseCampaignCountdownsResourceURI(t *testing.T) {
	campaignID, err := parseCampaignCountdownsResourceURI("daggerheart://campaign/camp-1/campaign_countdowns")
	if err != nil {
		t.Fatalf("parseCampaignCountdownsResourceURI() error = %v", err)
	}
	if campaignID != "camp-1" {
		t.Fatalf("campaign_id = %q, want camp-1", campaignID)
	}

	_, err = parseCampaignCountdownsResourceURI("daggerheart://campaign//campaign_countdowns")
	if err == nil || !strings.Contains(err.Error(), "campaign ID is required") {
		t.Fatalf("parseCampaignCountdownsResourceURI() error = %v", err)
	}
}

func TestParseCharacterSheetResourceURI(t *testing.T) {
	campaignID, characterID, err := parseCharacterSheetResourceURI("campaign://camp-1/characters/char-1/sheet")
	if err != nil {
		t.Fatalf("parseCharacterSheetResourceURI() error = %v", err)
	}
	if campaignID != "camp-1" || characterID != "char-1" {
		t.Fatalf("parseCharacterSheetResourceURI() = (%q, %q)", campaignID, characterID)
	}

	_, _, err = parseCharacterSheetResourceURI("campaign://camp-1/characters//sheet")
	if err == nil || !strings.Contains(err.Error(), "campaign and character IDs are required") {
		t.Fatalf("parseCharacterSheetResourceURI() error = %v", err)
	}
}
