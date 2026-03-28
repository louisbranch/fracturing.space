package forktransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestShouldCopyForkEvent(t *testing.T) {
	tests := []struct {
		name             string
		eventType        event.Type
		copyParticipants bool
		payload          []byte
		wantCopy         bool
		wantErr          bool
	}{
		{
			name:      "campaign_created_always_skipped",
			eventType: event.Type("campaign.created"),
			wantCopy:  false,
		},
		{
			name:      "campaign_forked_always_skipped",
			eventType: event.Type("campaign.forked"),
			wantCopy:  false,
		},
		{
			name:             "participant_joined_skip_when_no_copy",
			eventType:        event.Type("participant.joined"),
			copyParticipants: false,
			wantCopy:         false,
		},
		{
			name:             "participant_joined_copy_when_enabled",
			eventType:        event.Type("participant.joined"),
			copyParticipants: true,
			wantCopy:         true,
		},
		{
			name:             "participant_updated_skip_when_no_copy",
			eventType:        event.Type("participant.updated"),
			copyParticipants: false,
			wantCopy:         false,
		},
		{
			name:             "participant_left_skip_when_no_copy",
			eventType:        event.Type("participant.left"),
			copyParticipants: false,
			wantCopy:         false,
		},
		{
			name:             "character_updated_copy_when_participants_enabled",
			eventType:        event.Type("character.updated"),
			copyParticipants: true,
			payload:          []byte(`{"fields":{"owner_participant_id":"p1"}}`),
			wantCopy:         true,
		},
		{
			name:             "character_updated_no_participant_field",
			eventType:        event.Type("character.updated"),
			copyParticipants: false,
			payload:          []byte(`{"fields":{"name":"Hero"}}`),
			wantCopy:         true,
		},
		{
			name:             "character_updated_only_owner_participant_id_field",
			eventType:        event.Type("character.updated"),
			copyParticipants: false,
			payload:          []byte(`{"fields":{"owner_participant_id":"p1"}}`),
			wantCopy:         false,
		},
		{
			name:             "character_updated_owner_participant_id_plus_others",
			eventType:        event.Type("character.updated"),
			copyParticipants: false,
			payload:          []byte(`{"fields":{"owner_participant_id":"p1","name":"Hero"}}`),
			wantCopy:         true,
		},
		{
			name:             "character_updated_empty_owner_participant_id",
			eventType:        event.Type("character.updated"),
			copyParticipants: false,
			payload:          []byte(`{"fields":{"owner_participant_id":""}}`),
			wantCopy:         true,
		},
		{
			name:      "session_started_always_copied",
			eventType: event.Type("session.started"),
			wantCopy:  true,
		},
		{
			name:      "unknown_event_always_copied",
			eventType: event.Type("custom.event"),
			wantCopy:  true,
		},
		{
			name:             "character_updated_invalid_json",
			eventType:        event.Type("character.updated"),
			copyParticipants: false,
			payload:          []byte(`not json`),
			wantErr:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evt := event.Event{Type: tc.eventType, PayloadJSON: tc.payload}
			got, err := shouldCopyForkEvent(evt, tc.copyParticipants)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantCopy {
				t.Errorf("shouldCopyForkEvent = %v, want %v", got, tc.wantCopy)
			}
		})
	}
}

func TestForkEventForCampaign(t *testing.T) {
	evt := event.Event{
		CampaignID: "old-camp",
		Seq:        42,
		Hash:       "abc",
		EntityType: "campaign",
		EntityID:   "old-camp",
		Type:       event.Type("campaign.updated"),
	}
	forked := forkEventForCampaign(evt, "new-camp", true)

	if forked.CampaignID != "new-camp" {
		t.Fatalf("CampaignID = %q, want %q", forked.CampaignID, "new-camp")
	}
	if forked.Seq != 0 {
		t.Fatalf("Seq = %d, want 0", forked.Seq)
	}
	if forked.Hash != "" {
		t.Fatalf("Hash = %q, want empty", forked.Hash)
	}
	if forked.EntityID != "new-camp" {
		t.Fatalf("EntityID = %q, want %q (campaign entity should be updated)", forked.EntityID, "new-camp")
	}

	evt2 := event.Event{
		CampaignID: "old-camp",
		Seq:        10,
		Hash:       "def",
		EntityType: "character",
		EntityID:   "char-1",
	}
	forked2 := forkEventForCampaign(evt2, "new-camp", true)
	if forked2.EntityID != "char-1" {
		t.Fatalf("EntityID = %q, want %q (non-campaign entity should stay)", forked2.EntityID, "char-1")
	}
}
