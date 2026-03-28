package charactertransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSetDefaultControl_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.SetDefaultControl(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultControl_MissingCampaignId(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String(""),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultControl_MissingCharacterId(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(ts.build())
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		ParticipantId: wrapperspb.String(""),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultControl_CampaignNotFound(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:    "nonexistent",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String(""),
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestSetDefaultControl_CharacterNotFound(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(ts.build())
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "nonexistent",
		ParticipantId: wrapperspb.String(""),
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestSetDefaultControl_RequiresDomainEngine(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", OwnerParticipantID: "manager-1"},
	}

	svc := NewService(ts.build())
	_, err := svc.SetDefaultControl(requestctx.WithParticipantID("manager-1"), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String(""),
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSetDefaultControl_MissingParticipantId(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Pronouns: "she/her", CreatedAt: now},
	}

	svc := NewService(ts.build())
	_, err := svc.SetDefaultControl(context.Background(), &statev1.SetDefaultControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultControl_ParticipantNotFound(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}

	svc := NewService(ts.build())
	_, err := svc.SetDefaultControl(requestctx.WithParticipantID("manager-1"), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String("nonexistent"),
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestSetDefaultControl_Success_Unassigned(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Pronouns: "she/her", CreatedAt: now},
	}
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"participant_id":""}}`),
			}),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())

	resp, err := svc.SetDefaultControl(requestctx.WithParticipantID("manager-1"), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String(""),
	})
	if err != nil {
		t.Fatalf("SetDefaultControl returned error: %v", err)
	}
	if resp.CampaignId != "c1" {
		t.Errorf("Response CampaignId = %q, want %q", resp.CampaignId, "c1")
	}
	if resp.CharacterId != "ch1" {
		t.Errorf("Response CharacterId = %q, want %q", resp.CharacterId, "ch1")
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["participant_id"] != "" {
		t.Fatalf("participant_id = %q, want empty", payload.Fields["participant_id"])
	}
	if payload.Fields["avatar_set_id"] != assetcatalog.AvatarSetBlankV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Fields["avatar_set_id"], assetcatalog.AvatarSetBlankV1)
	}
	if payload.Fields["avatar_asset_id"] != "" {
		t.Fatalf("avatar_asset_id = %q, want empty", payload.Fields["avatar_asset_id"])
	}
	if _, ok := payload.Fields["pronouns"]; ok {
		t.Fatalf("pronouns field should be omitted, got %q", payload.Fields["pronouns"])
	}

	updated, err := ts.Character.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if updated.ParticipantID != "" {
		t.Fatalf("ParticipantID = %q, want empty", updated.ParticipantID)
	}
	if updated.Pronouns != "she/her" {
		t.Fatalf("Pronouns = %q, want %q", updated.Pronouns, "she/her")
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("character.updated"))
	}
}

func TestSetDefaultControl_Success_Participant(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Pronouns: "she/her", CreatedAt: now},
	}
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"manager-1": {
			ID:             "manager-1",
			CampaignID:     "c1",
			Name:           "Manager 1",
			CampaignAccess: participant.CampaignAccessManager,
			CreatedAt:      now,
		},
		"p1": {
			ID:            "p1",
			CampaignID:    "c1",
			Name:          "Player 1",
			AvatarSetID:   assetcatalog.AvatarSetPeopleV1,
			AvatarAssetID: "009",
			Pronouns:      "they/them",
			CreatedAt:     now,
		},
	}
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"participant_id":"p1"}}`),
			}),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())

	resp, err := svc.SetDefaultControl(requestctx.WithParticipantID("manager-1"), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String("p1"),
	})
	if err != nil {
		t.Fatalf("SetDefaultControl returned error: %v", err)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["participant_id"] != "p1" {
		t.Fatalf("participant_id = %q, want %q", payload.Fields["participant_id"], "p1")
	}
	if payload.Fields["avatar_set_id"] != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Fields["avatar_set_id"], assetcatalog.AvatarSetPeopleV1)
	}
	if payload.Fields["avatar_asset_id"] != "009" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.Fields["avatar_asset_id"], "009")
	}
	if _, ok := payload.Fields["pronouns"]; ok {
		t.Fatalf("pronouns field should be omitted, got %q", payload.Fields["pronouns"])
	}

	ctrl, err := ts.Character.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if ctrl.ParticipantID != "p1" {
		t.Errorf("ParticipantID = %q, want %q", ctrl.ParticipantID, "p1")
	}
	if ctrl.Pronouns != "she/her" {
		t.Fatalf("Pronouns = %q, want %q", ctrl.Pronouns, "she/her")
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("character.updated"))
	}
	_ = resp
}

func TestSetDefaultControl_UsesDomainEngine(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Pronouns: "she/her", CreatedAt: now},
	}
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"manager-1": {
			ID:             "manager-1",
			CampaignID:     "c1",
			Name:           "Manager 1",
			CampaignAccess: participant.CampaignAccessManager,
			CreatedAt:      now,
		},
		"p1": {ID: "p1", CampaignID: "c1", Name: "Player 1", CreatedAt: now},
	}

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"participant_id":"p1"}}`),
			}),
		},
	}}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), runtimekit.FixedClock(now), nil)

	resp, err := svc.SetDefaultControl(requestctx.WithParticipantID("manager-1"), &statev1.SetDefaultControlRequest{
		CampaignId:    "c1",
		CharacterId:   "ch1",
		ParticipantId: wrapperspb.String("p1"),
	})
	if err != nil {
		t.Fatalf("SetDefaultControl returned error: %v", err)
	}
	if resp.CharacterId != "ch1" {
		t.Fatalf("Response CharacterId = %q, want %q", resp.CharacterId, "ch1")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("character.update") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "character.update")
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("character.updated"))
	}
	updated, err := ts.Character.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if updated.ParticipantID != "p1" {
		t.Fatalf("ParticipantID = %q, want %q", updated.ParticipantID, "p1")
	}
}
