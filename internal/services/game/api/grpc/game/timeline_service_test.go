package game

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"google.golang.org/grpc/codes"
)

func TestListTimelineEntries_NilRequest(t *testing.T) {
	svc := NewEventService(Stores{})
	_, err := svc.ListTimelineEntries(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListTimelineEntries_MissingCampaignId(t *testing.T) {
	service := NewEventService(Stores{Event: newFakeEventStore()})
	_, err := service.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListTimelineEntries_InvalidFilter(t *testing.T) {
	service := NewEventService(Stores{Event: newFakeEventStore()})
	_, err := service.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
		Filter:     "invalid filter syntax ===",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListTimelineEntries_ProjectionDisplayByDomain(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()

	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Name:   "Riverfall",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status: campaign.CampaignStatusDraft,
	}
	participantStore.participants["c1"] = map[string]participant.Participant{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			DisplayName:    "Ada",
			Role:           participant.ParticipantRoleGM,
			Controller:     participant.ControllerAI,
			CampaignAccess: participant.CampaignAccessOwner,
		},
	}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {
			ID:         "ch1",
			CampaignID: "c1",
			Name:       "Frodo",
			Kind:       character.CharacterKindPC,
		},
	}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {
			ID:         "s1",
			CampaignID: "c1",
			Name:       "Session 1",
			Status:     session.SessionStatusActive,
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

	eventStore.events["c1"] = []event.Event{
		{
			CampaignID:  "c1",
			Seq:         1,
			Type:        event.TypeCampaignCreated,
			EntityType:  "campaign",
			EntityID:    "c1",
			Timestamp:   now,
			PayloadJSON: seedPayload(event.CampaignCreatedPayload{Name: "Riverfall"}),
		},
		{
			CampaignID:  "c1",
			Seq:         2,
			Type:        event.TypeParticipantJoined,
			EntityType:  "participant",
			EntityID:    "p1",
			Timestamp:   now,
			PayloadJSON: seedPayload(event.ParticipantJoinedPayload{ParticipantID: "p1"}),
		},
		{
			CampaignID:  "c1",
			Seq:         3,
			Type:        event.TypeCharacterCreated,
			EntityType:  "character",
			EntityID:    "ch1",
			Timestamp:   now,
			PayloadJSON: seedPayload(event.CharacterCreatedPayload{CharacterID: "ch1"}),
		},
		{
			CampaignID:  "c1",
			Seq:         4,
			Type:        event.TypeSessionStarted,
			EntityType:  "session",
			EntityID:    "s1",
			Timestamp:   now,
			PayloadJSON: seedPayload(event.SessionStartedPayload{SessionID: "s1"}),
		},
	}

	svc := NewEventService(Stores{
		Event:       eventStore,
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Session:     sessionStore,
	})

	resp, err := svc.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{
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

func TestListTimelineEntries_CharacterStateChanges(t *testing.T) {
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	now := time.Now().UTC()
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {
			ID:         "ch1",
			CampaignID: "c1",
			Name:       "Frodo",
			Kind:       character.CharacterKindPC,
		},
	}

	hpAfter := 6
	hopeAfter := 2
	hopeMaxAfter := 6
	stressAfter := 0
	armorAfter := 0
	lifeStateAfter := "alive"
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:    "ch1",
		HpAfter:        &hpAfter,
		HopeAfter:      &hopeAfter,
		HopeMaxAfter:   &hopeMaxAfter,
		StressAfter:    &stressAfter,
		ArmorAfter:     &armorAfter,
		LifeStateAfter: &lifeStateAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	eventStore.events["c1"] = []event.Event{
		{
			CampaignID:  "c1",
			Seq:         1,
			Type:        daggerheart.EventTypeCharacterStatePatched,
			EntityType:  "character",
			EntityID:    "ch1",
			Timestamp:   now,
			PayloadJSON: payloadJSON,
		},
	}

	svc := NewEventService(Stores{
		Event:     eventStore,
		Character: characterStore,
	})

	resp, err := svc.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
		OrderBy:    "seq",
	})
	if err != nil {
		t.Fatalf("list timeline entries: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}

	fields := resp.Entries[0].GetProjection().GetFields()
	if len(fields) == 0 {
		t.Fatal("expected change fields")
	}
	fieldMap := make(map[string]string, len(fields))
	for _, field := range fields {
		fieldMap[field.GetLabel()] = field.GetValue()
	}
	assertField := func(label, value string) {
		t.Helper()
		if got := fieldMap[label]; got != value {
			t.Fatalf("field %q = %q, want %q", label, got, value)
		}
	}
	assertField("HP", "= 6")
	assertField("Hope", "= 2")
	assertField("Hope Max", "= 6")
	assertField("Stress", "= 0")
	assertField("Armor", "= 0")
	assertField("Life State", "= alive")
}

func TestListTimelineEntries_CharacterStateChanges_WithBefore(t *testing.T) {
	characterStore := newFakeCharacterStore()
	eventStore := newFakeEventStore()

	now := time.Now().UTC()
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {
			ID:         "ch1",
			CampaignID: "c1",
			Name:       "Frodo",
			Kind:       character.CharacterKindPC,
		},
	}

	hpBefore := 3
	hpAfter := 6
	hopeBefore := 1
	hopeAfter := 2
	hopeMaxBefore := 6
	hopeMaxAfter := 7
	stressBefore := 1
	stressAfter := 1
	armorBefore := 0
	armorAfter := 2
	lifeStateBefore := "alive"
	lifeStateAfter := "dying"
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:     "ch1",
		HpBefore:        &hpBefore,
		HpAfter:         &hpAfter,
		HopeBefore:      &hopeBefore,
		HopeAfter:       &hopeAfter,
		HopeMaxBefore:   &hopeMaxBefore,
		HopeMaxAfter:    &hopeMaxAfter,
		StressBefore:    &stressBefore,
		StressAfter:     &stressAfter,
		ArmorBefore:     &armorBefore,
		ArmorAfter:      &armorAfter,
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  &lifeStateAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	eventStore.events["c1"] = []event.Event{
		{
			CampaignID:  "c1",
			Seq:         1,
			Type:        daggerheart.EventTypeCharacterStatePatched,
			EntityType:  "character",
			EntityID:    "ch1",
			Timestamp:   now,
			PayloadJSON: payloadJSON,
		},
	}

	svc := NewEventService(Stores{
		Event:     eventStore,
		Character: characterStore,
	})

	resp, err := svc.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
		OrderBy:    "seq",
	})
	if err != nil {
		t.Fatalf("list timeline entries: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}

	fields := resp.Entries[0].GetProjection().GetFields()
	if len(fields) == 0 {
		t.Fatal("expected change fields")
	}
	fieldMap := make(map[string]string, len(fields))
	for _, field := range fields {
		fieldMap[field.GetLabel()] = field.GetValue()
	}
	assertField := func(label, value string) {
		t.Helper()
		if got := fieldMap[label]; got != value {
			t.Fatalf("field %q = %q, want %q", label, got, value)
		}
	}
	assertField("HP", "3 -> 6")
	assertField("Hope", "1 -> 2")
	assertField("Hope Max", "6 -> 7")
	assertField("Armor", "0 -> 2")
	assertField("Life State", "alive -> dying")
	if _, ok := fieldMap["Stress"]; ok {
		t.Fatalf("expected stress change to be omitted")
	}
}
