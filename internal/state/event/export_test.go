package event

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestExportHumanReadable_SingleEvent(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		{
			CampaignID: "camp_abc123",
			Seq:        1,
			Hash:       "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
			Timestamp:  ts,
			Type:       TypeCampaignCreated,
			ActorType:  ActorTypeSystem,
			EntityType: "campaign",
			EntityID:   "camp_abc123",
			PayloadJSON: []byte(`{"name":"Dragon's Lair","system":"daggerheart"}`),
		},
	}

	var buf bytes.Buffer
	if err := ExportHumanReadable(events, &buf); err != nil {
		t.Fatalf("ExportHumanReadable failed: %v", err)
	}

	output := buf.String()

	// Verify key components are present
	checks := []string{
		"[2024-01-15T10:30:00Z] campaign.created",
		"hash: a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
		"campaign: camp_abc123",
		"seq: 1",
		"actor: system",
		"entity: campaign/camp_abc123",
		"payload:",
		`"name"`,
		`"Dragon's Lair"`,
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestExportHumanReadable_MultipleEvents(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		{
			CampaignID: "camp_abc123",
			Seq:        1,
			Hash:       "hash1",
			Timestamp:  ts,
			Type:       TypeCampaignCreated,
			ActorType:  ActorTypeSystem,
		},
		{
			CampaignID: "camp_abc123",
			Seq:        2,
			Hash:       "hash2",
			Timestamp:  ts.Add(time.Minute),
			Type:       TypeParticipantJoined,
			ActorType:  ActorTypeParticipant,
			ActorID:    "part_xyz",
			EntityType: "participant",
			EntityID:   "part_xyz",
			PayloadJSON: []byte(`{"display_name":"Alice","role":"player"}`),
		},
	}

	var buf bytes.Buffer
	if err := ExportHumanReadable(events, &buf); err != nil {
		t.Fatalf("ExportHumanReadable failed: %v", err)
	}

	output := buf.String()

	// Should have both events
	if !strings.Contains(output, "campaign.created") {
		t.Error("output missing campaign.created")
	}
	if !strings.Contains(output, "participant.joined") {
		t.Error("output missing participant.joined")
	}

	// Second event should have actor with ID
	if !strings.Contains(output, "actor: participant/part_xyz") {
		t.Error("output missing actor with ID")
	}
}

func TestExportHumanReadable_WithSessionAndCorrelation(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		{
			CampaignID:   "camp_abc123",
			Seq:          1,
			Hash:         "hash1",
			Timestamp:    ts,
			Type:         TypeCharacterStateChanged,
			SessionID:    "sess_123",
			RequestID:    "req_456",
			InvocationID: "inv_789",
			ActorType:    ActorTypeParticipant,
			ActorID:      "part_xyz",
			EntityType:   "character",
			EntityID:     "char_abc",
			PayloadJSON:  []byte(`{"hp_before":10,"hp_after":8}`),
		},
	}

	var buf bytes.Buffer
	if err := ExportHumanReadable(events, &buf); err != nil {
		t.Fatalf("ExportHumanReadable failed: %v", err)
	}

	output := buf.String()

	checks := []string{
		"session: sess_123",
		"request: req_456",
		"invocation: inv_789",
		"entity: character/char_abc",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestExportHumanReadable_EmptyEvents(t *testing.T) {
	var buf bytes.Buffer
	if err := ExportHumanReadable(nil, &buf); err != nil {
		t.Fatalf("ExportHumanReadable failed: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("expected empty output, got: %q", buf.String())
	}
}

func TestExportHumanReadable_InvalidJSON(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	events := []Event{
		{
			CampaignID:  "camp_abc123",
			Seq:         1,
			Timestamp:   ts,
			Type:        TypeNoteAdded,
			ActorType:   ActorTypeSystem,
			PayloadJSON: []byte(`not valid json`),
		},
	}

	var buf bytes.Buffer
	if err := ExportHumanReadable(events, &buf); err != nil {
		t.Fatalf("ExportHumanReadable failed: %v", err)
	}

	output := buf.String()

	// Should still contain the raw payload as fallback
	if !strings.Contains(output, "not valid json") {
		t.Errorf("output missing raw payload\nGot:\n%s", output)
	}
}
