package eventtransport

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestListTimelineEntries_ProjectionDisplayByDomain(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	sessionStore := gametest.NewFakeSessionStore()
	eventStore := gametest.NewFakeEventStore()

	now := time.Now().UTC()
	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Riverfall",
		System: bridge.SystemIDDaggerheart,
		Status: campaign.StatusDraft,
	}
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "Ada",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerAI,
			CampaignAccess: participant.CampaignAccessOwner,
		},
	}
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {
			ID:         "ch1",
			CampaignID: "c1",
			Name:       "Frodo",
			Kind:       character.KindPC,
		},
	}
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {
			ID:         "s1",
			CampaignID: "c1",
			Name:       "Session 1",
			Status:     session.StatusActive,
			StartedAt:  now,
		},
	}

	seedPayload := func(payload any) []byte {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		return encoded
	}

	eventStore.Events["c1"] = []event.Event{
		{
			CampaignID:  "c1",
			Seq:         1,
			Type:        event.Type("campaign.created"),
			EntityType:  "campaign",
			EntityID:    "c1",
			Timestamp:   now,
			PayloadJSON: seedPayload(campaign.CreatePayload{Name: "Riverfall"}),
		},
		{
			CampaignID:  "c1",
			Seq:         2,
			Type:        event.Type("participant.joined"),
			EntityType:  "participant",
			EntityID:    "p1",
			Timestamp:   now,
			PayloadJSON: seedPayload(participant.JoinPayload{ParticipantID: "p1"}),
		},
		{
			CampaignID:  "c1",
			Seq:         3,
			Type:        event.Type("character.created"),
			EntityType:  "character",
			EntityID:    "ch1",
			Timestamp:   now,
			PayloadJSON: seedPayload(character.CreatePayload{CharacterID: "ch1"}),
		},
		{
			CampaignID:  "c1",
			Seq:         4,
			Type:        event.Type("session.started"),
			EntityType:  "session",
			EntityID:    "s1",
			Timestamp:   now,
			PayloadJSON: seedPayload(session.StartPayload{SessionID: "s1", SessionName: "Session 1"}),
		},
	}

	svc := NewService(Deps{
		Event:       eventStore,
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Session:     sessionStore,
	})

	resp, err := svc.ListTimelineEntries(requestctx.WithAdminOverride(context.Background(), "timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
		OrderBy:    "seq",
	})
	if err != nil {
		t.Fatalf("list timeline entries: %v", err)
	}
	if len(resp.Entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(resp.Entries))
	}

	assertEntry := func(entry *campaignv1.TimelineEntry, iconID commonv1.IconId, title string) {
		if entry.GetIconId() != iconID {
			t.Fatalf("expected icon %s, got %s", iconID.String(), entry.GetIconId().String())
		}
		projection := entry.GetProjection()
		if projection == nil {
			t.Fatalf("expected projection display for %s", iconID.String())
		}
		if projection.GetTitle() != title {
			t.Fatalf("expected title %q, got %q", title, projection.GetTitle())
		}
		if strings.TrimSpace(entry.GetEventPayloadJson()) == "" {
			t.Fatalf("expected event payload for %s", iconID.String())
		}
	}

	assertEntry(resp.Entries[0], commonv1.IconId_ICON_ID_CAMPAIGN, "Riverfall")
	assertEntry(resp.Entries[1], commonv1.IconId_ICON_ID_PARTICIPANT, "Ada")
	assertEntry(resp.Entries[2], commonv1.IconId_ICON_ID_CHARACTER, "Frodo")
	assertEntry(resp.Entries[3], commonv1.IconId_ICON_ID_SESSION, "Session 1")
}
