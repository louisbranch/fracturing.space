package eventtransport

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestListTimelineEntries_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.ListTimelineEntries(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListTimelineEntries_MissingCampaignId(t *testing.T) {
	service := NewService(Deps{Event: gametest.NewFakeEventStore()})
	_, err := service.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListTimelineEntries_RequiresCampaignReadPolicy(t *testing.T) {
	participantStore := gametest.NewFakeParticipantStore()
	service := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: participantStore},
		Event:       gametest.NewFakeEventStore(),
		Participant: participantStore,
	})
	_, err := service.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListTimelineEntries_InvalidFilter(t *testing.T) {
	service := NewService(Deps{Event: gametest.NewFakeEventStore()})
	_, err := service.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
		Filter:     "invalid filter syntax ===",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListTimelineEntries_MissingParticipantStoreFailsFast(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	eventStore.Events["c1"] = []event.Event{
		{
			CampaignID: "c1",
			Seq:        1,
			Type:       event.Type("participant.joined"),
			EntityType: "participant",
			EntityID:   "p1",
			Timestamp:  time.Now().UTC(),
		},
	}

	service := NewService(Deps{
		Event: eventStore,
	})

	_, err := service.ListTimelineEntries(gametest.ContextWithAdminOverride("timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
	assertStatusMessage(t, err, "resolve timeline entry")
}

func TestListTimelineEntries_MissingCampaignStoreFailsFast(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	eventStore.Events["c1"] = []event.Event{
		{
			CampaignID: "c1",
			Seq:        1,
			Type:       event.Type("campaign.created"),
			EntityType: "campaign",
			EntityID:   "c1",
			Timestamp:  time.Now().UTC(),
		},
	}

	service := NewService(Deps{
		Event: eventStore,
	})

	_, err := service.ListTimelineEntries(gametest.ContextWithAdminOverride("timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
	assertStatusMessage(t, err, "resolve timeline entry")
}

func TestListTimelineEntries_MissingCharacterStoreFailsFast(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	eventStore.Events["c1"] = []event.Event{
		{
			CampaignID: "c1",
			Seq:        1,
			Type:       event.Type("character.created"),
			EntityType: "character",
			EntityID:   "ch1",
			Timestamp:  time.Now().UTC(),
		},
	}

	service := NewService(Deps{
		Event: eventStore,
	})

	_, err := service.ListTimelineEntries(gametest.ContextWithAdminOverride("timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
	assertStatusMessage(t, err, "resolve timeline entry")
}

func TestListTimelineEntries_MissingSessionStoreFailsFast(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	eventStore.Events["c1"] = []event.Event{
		{
			CampaignID: "c1",
			Seq:        1,
			Type:       event.Type("session.started"),
			EntityType: "session",
			EntityID:   "s1",
			Timestamp:  time.Now().UTC(),
		},
	}

	service := NewService(Deps{
		Event: eventStore,
	})

	_, err := service.ListTimelineEntries(gametest.ContextWithAdminOverride("timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
	assertStatusMessage(t, err, "resolve timeline entry")
}

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

	resp, err := svc.ListTimelineEntries(gametest.ContextWithAdminOverride("timeline-test"), &campaignv1.ListTimelineEntriesRequest{
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
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()

	now := time.Now().UTC()
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {
			ID:         "ch1",
			CampaignID: "c1",
			Name:       "Frodo",
			Kind:       character.KindPC,
		},
	}

	hp := 6
	hope := 2
	hopeMax := 6
	stress := 0
	armor := 0
	lifeState := "alive"
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: "ch1",
		HP:          &hp,
		Hope:        &hope,
		HopeMax:     &hopeMax,
		Stress:      &stress,
		Armor:       &armor,
		LifeState:   &lifeState,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	eventStore.Events["c1"] = []event.Event{
		{
			CampaignID:  "c1",
			Seq:         1,
			Type:        event.Type("sys.daggerheart.character_state_patched"),
			EntityType:  "character",
			EntityID:    "ch1",
			Timestamp:   now,
			PayloadJSON: payloadJSON,
		},
	}

	svc := NewService(Deps{
		Event:     eventStore,
		Character: characterStore,
	})

	resp, err := svc.ListTimelineEntries(gametest.ContextWithAdminOverride("timeline-test"), &campaignv1.ListTimelineEntriesRequest{
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
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()

	now := time.Now().UTC()
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {
			ID:         "ch1",
			CampaignID: "c1",
			Name:       "Frodo",
			Kind:       character.KindPC,
		},
	}

	hp := 6
	hope := 2
	hopeMax := 7
	stress := 1
	armor := 2
	lifeState := "dying"
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: "ch1",
		HP:          &hp,
		Hope:        &hope,
		HopeMax:     &hopeMax,
		Stress:      &stress,
		Armor:       &armor,
		LifeState:   &lifeState,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	eventStore.Events["c1"] = []event.Event{
		{
			CampaignID:  "c1",
			Seq:         1,
			Type:        event.Type("sys.daggerheart.character_state_patched"),
			EntityType:  "character",
			EntityID:    "ch1",
			Timestamp:   now,
			PayloadJSON: payloadJSON,
		},
	}

	svc := NewService(Deps{
		Event:     eventStore,
		Character: characterStore,
	})

	resp, err := svc.ListTimelineEntries(gametest.ContextWithAdminOverride("timeline-test"), &campaignv1.ListTimelineEntriesRequest{
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
	assertField("Hope Max", "= 7")
	assertField("Stress", "= 1")
	assertField("Armor", "= 2")
	assertField("Life State", "= dying")
}
