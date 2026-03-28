package gametools

import (
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var testTS = timestamppb.New(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))

func TestCampaignProtoToEntry(t *testing.T) {
	t.Parallel()
	c := &statev1.Campaign{
		Id:               "camp-1",
		Name:             "Test Campaign",
		Status:           statev1.CampaignStatus_ACTIVE,
		GmMode:           statev1.GmMode_AI,
		Intent:           statev1.CampaignIntent_STANDARD,
		AccessPolicy:     statev1.CampaignAccessPolicy_PRIVATE,
		ParticipantCount: 3,
		CharacterCount:   2,
		ThemePrompt:      "Dark fantasy",
		CreatedAt:        testTS,
		UpdatedAt:        testTS,
	}
	entry := campaignProtoToEntry(c)

	if entry.ID != "camp-1" {
		t.Errorf("ID = %q, want camp-1", entry.ID)
	}
	if entry.Name != "Test Campaign" {
		t.Errorf("Name = %q, want Test Campaign", entry.Name)
	}
	if entry.Status != "ACTIVE" {
		t.Errorf("Status = %q, want ACTIVE", entry.Status)
	}
	if entry.GmMode != "AI" {
		t.Errorf("GmMode = %q, want AI", entry.GmMode)
	}
	if entry.ParticipantCount != 3 {
		t.Errorf("ParticipantCount = %d, want 3", entry.ParticipantCount)
	}
	if entry.CharacterCount != 2 {
		t.Errorf("CharacterCount = %d, want 2", entry.CharacterCount)
	}
	if entry.ThemePrompt != "Dark fantasy" {
		t.Errorf("ThemePrompt = %q, want Dark fantasy", entry.ThemePrompt)
	}
	if entry.CreatedAt == "" {
		t.Error("CreatedAt is empty")
	}
}

func TestCampaignProtoToEntryNilTimestamps(t *testing.T) {
	t.Parallel()
	entry := campaignProtoToEntry(&statev1.Campaign{Id: "camp-nil"})
	if entry.CreatedAt != "" {
		t.Errorf("CreatedAt = %q, want empty for nil timestamp", entry.CreatedAt)
	}
	if entry.CompletedAt != "" {
		t.Errorf("CompletedAt = %q, want empty for nil timestamp", entry.CompletedAt)
	}
}

func TestParticipantProtosToEntries(t *testing.T) {
	t.Parallel()
	participants := []*statev1.Participant{
		{Id: "p-1", CampaignId: "camp-1", Name: "Alice", Role: statev1.ParticipantRole_GM, Controller: statev1.Controller_CONTROLLER_HUMAN, CreatedAt: testTS},
		{Id: "p-2", CampaignId: "camp-1", Name: "Bob", Role: statev1.ParticipantRole_PLAYER, Controller: statev1.Controller_CONTROLLER_AI, CreatedAt: testTS},
	}
	entries := participantProtosToEntries(participants)

	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
	if entries[0].Name != "Alice" || entries[0].Role != "GM" {
		t.Errorf("entry[0] = (%q, %q), want (Alice, GM)", entries[0].Name, entries[0].Role)
	}
	if entries[1].Controller != "AI" {
		t.Errorf("entry[1].Controller = %q, want AI", entries[1].Controller)
	}
}

func TestParticipantProtosToEntriesEmpty(t *testing.T) {
	t.Parallel()
	entries := participantProtosToEntries(nil)
	if len(entries) != 0 {
		t.Fatalf("len = %d, want 0 for nil input", len(entries))
	}
}

func TestCharacterProtosToEntries(t *testing.T) {
	t.Parallel()
	characters := []*statev1.Character{
		{Id: "c-1", CampaignId: "camp-1", Name: "Hero", Kind: statev1.CharacterKind_PC, Aliases: []string{"The Brave"}, CreatedAt: testTS},
		{Id: "c-2", CampaignId: "camp-1", Name: "Guide", Kind: statev1.CharacterKind_NPC, CreatedAt: testTS},
	}
	entries := characterProtosToEntries(characters)

	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
	if entries[0].Kind != "PC" {
		t.Errorf("entry[0].Kind = %q, want PC", entries[0].Kind)
	}
	if len(entries[0].Aliases) != 1 || entries[0].Aliases[0] != "The Brave" {
		t.Errorf("entry[0].Aliases = %v, want [The Brave]", entries[0].Aliases)
	}
	// Verify NPC with no aliases gets empty slice, not nil.
	if entries[1].Aliases == nil {
		t.Error("entry[1].Aliases is nil, want empty slice")
	}
	if len(entries[1].Aliases) != 0 {
		t.Errorf("entry[1].Aliases = %v, want empty", entries[1].Aliases)
	}
}

func TestCharacterProtosToEntriesAliasesCopied(t *testing.T) {
	t.Parallel()
	original := []string{"A", "B"}
	characters := []*statev1.Character{
		{Id: "c-1", Aliases: original},
	}
	entries := characterProtosToEntries(characters)

	// Mutating the entry's aliases must not affect the original slice.
	entries[0].Aliases[0] = "MUTATED"
	if original[0] == "MUTATED" {
		t.Fatal("alias slice was not copied — original was mutated")
	}
}
