package character

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestDecideCharacterCreate_EmitsCharacterCreatedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","name":"  Aria  ","kind":"CHARACTER_KIND_PC","notes":"  new notes  "}`),
	}

	decision := Decide(State{}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.CampaignID != "camp-1" {
		t.Fatalf("event campaign id = %s, want %s", evt.CampaignID, "camp-1")
	}
	if evt.Type != event.Type("character.created") {
		t.Fatalf("event type = %s, want %s", evt.Type, "character.created")
	}
	if evt.EntityType != "character" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "character")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}
	if evt.ActorType != event.ActorTypeSystem {
		t.Fatalf("event actor type = %s, want %s", evt.ActorType, event.ActorTypeSystem)
	}

	var payload CreatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("payload character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.Name != "Aria" {
		t.Fatalf("payload name = %s, want %s", payload.Name, "Aria")
	}
	if payload.Kind != "pc" {
		t.Fatalf("payload kind = %s, want %s", payload.Kind, "pc")
	}
	if payload.Notes != "new notes" {
		t.Fatalf("payload notes = %s, want %s", payload.Notes, "new notes")
	}
}

func TestDecideCharacterCreate_NormalizesOwnerParticipantID(t *testing.T) {
	now := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("character.create"),
		ActorType:  command.ActorTypeParticipant,
		PayloadJSON: []byte(
			`{"character_id":"char-1","owner_participant_id":"  part-1  ","name":"Aria","kind":"PC"}`,
		),
	}

	decision := Decide(State{}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload CreatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.OwnerParticipantID != "part-1" {
		t.Fatalf("owner participant id = %q, want %q", payload.OwnerParticipantID, "part-1")
	}
}

func TestDecideCharacterCreate_MissingCharacterIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":" ","name":"Aria","kind":"PC"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterIDRequired)
	}
}

func TestDecideCharacterCreate_MissingNameRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","name":" ","kind":"PC"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterNameEmpty {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterNameEmpty)
	}
}

func TestDecideCharacterCreate_InvalidKindRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","name":"Aria","kind":"ALIEN"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterKindInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterKindInvalid)
	}
}

func TestDecideCharacterCreate_WhenAlreadyCreated_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","name":"Aria","kind":"PC"}`),
	}

	decision := Decide(State{Created: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterAlreadyExists {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterAlreadyExists)
	}
}

func TestDecideCharacterUpdate_EmitsCharacterUpdatedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"name":"  Aria  ","kind":"NPC","notes":"  new notes  ","participant_id":"  p-1  "}}`),
	}

	decision := Decide(State{Created: true}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("character.updated") {
		t.Fatalf("event type = %s, want %s", evt.Type, "character.updated")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload UpdatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("payload character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.Fields["name"] != "Aria" {
		t.Fatalf("payload name = %s, want %s", payload.Fields["name"], "Aria")
	}
	if payload.Fields["kind"] != "npc" {
		t.Fatalf("payload kind = %s, want %s", payload.Fields["kind"], "npc")
	}
	if payload.Fields["notes"] != "new notes" {
		t.Fatalf("payload notes = %s, want %s", payload.Fields["notes"], "new notes")
	}
	if payload.Fields["participant_id"] != "p-1" {
		t.Fatalf("payload participant id = %s, want %s", payload.Fields["participant_id"], "p-1")
	}
}

func TestDecideCharacterUpdate_NormalizesOwnerParticipantID(t *testing.T) {
	now := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeParticipant,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"owner_participant_id":"  part-2  "}}`),
	}

	decision := Decide(State{Created: true, OwnerParticipantID: "part-1"}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload UpdatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Fields["owner_participant_id"] != "part-2" {
		t.Fatalf("owner participant id = %q, want %q", payload.Fields["owner_participant_id"], "part-2")
	}
}

func TestDecideCharacterUpdate_EmptyOwnerParticipantIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeParticipant,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"owner_participant_id":"  "}}`),
	}

	decision := Decide(State{Created: true, OwnerParticipantID: "part-1"}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterOwnerParticipantID {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterOwnerParticipantID)
	}
}

func TestDecideCharacterUpdate_WhenNotCreatedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"name":"Aria"}}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterNotCreated {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterNotCreated)
	}
}

func TestDecideCharacterUpdate_EmptyFieldsRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{}}`),
	}

	decision := Decide(State{Created: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterUpdateEmpty {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterUpdateEmpty)
	}
}

func TestDecideCharacterUpdate_InvalidFieldRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"unknown":"value"}}`),
	}

	decision := Decide(State{Created: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterUpdateFieldInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterUpdateFieldInvalid)
	}
}

func TestDecideCharacterUpdate_InvalidKindRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"kind":"ALIEN"}}`),
	}

	decision := Decide(State{Created: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterKindInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterKindInvalid)
	}
}

func TestDecideCharacterUpdate_EmptyNameRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"name":"  "}}`),
	}

	decision := Decide(State{Created: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterNameEmpty {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterNameEmpty)
	}
}

func TestDecideCharacterUpdate_MissingCharacterIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":" ","fields":{"name":"Aria"}}`),
	}

	decision := Decide(State{Created: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterIDRequired)
	}
}

func TestDecideCharacterDelete_EmitsCharacterDeletedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.delete"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","reason":"  retired  "}`),
	}

	decision := Decide(State{Created: true}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("character.deleted") {
		t.Fatalf("event type = %s, want %s", evt.Type, "character.deleted")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload DeletePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("payload character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.Reason != "retired" {
		t.Fatalf("payload reason = %s, want %s", payload.Reason, "retired")
	}
}

func TestDecideCharacterDelete_WhenNotCreatedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.delete"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterNotCreated {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterNotCreated)
	}
}

func TestDecideCharacterDelete_MissingCharacterIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.delete"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":" "}`),
	}

	decision := Decide(State{Created: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterIDRequired)
	}
}

func TestDecideCharacterUpdate_WhenDeletedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"name":"Aria"}}`),
	}

	decision := Decide(State{Created: true, Deleted: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterNotCreated {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterNotCreated)
	}
}

func TestDecideCharacterDelete_WhenDeletedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.delete"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1"}`),
	}

	decision := Decide(State{Created: true, Deleted: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterNotCreated {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterNotCreated)
	}
}

func TestDecideCharacterProfileUpdate_EmitsProfileUpdatedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.profile_update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","system_profile":{"daggerheart":{"level":2}}}`),
	}

	decision := Decide(State{Created: true}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("character.profile_updated") {
		t.Fatalf("event type = %s, want %s", evt.Type, "character.profile_updated")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload ProfileUpdatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("payload character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.SystemProfile == nil {
		t.Fatal("expected system profile to be set")
	}
}

func TestDecideCharacterProfileUpdate_WhenNotCreatedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.profile_update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","system_profile":{"daggerheart":{}}}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterNotCreated {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterNotCreated)
	}
}

func TestDecideCharacterProfileUpdate_MissingCharacterIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.profile_update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":" ","system_profile":{"daggerheart":{}}}`),
	}

	decision := Decide(State{Created: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterIDRequired)
	}
}

func TestDecideCharacterCreate_DefaultsAvatarSelection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","name":"Aria","kind":"PC"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload CreatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.AvatarSetID == "" {
		t.Fatal("expected avatar set to be defaulted")
	}
	if payload.AvatarAssetID == "" {
		t.Fatal("expected avatar asset to be defaulted")
	}
}

func TestDecideCharacterCreate_InvalidAvatarSetRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","name":"Aria","kind":"PC","avatar_set_id":"missing"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterAvatarSetInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterAvatarSetInvalid)
	}
}

func TestDecideCharacterUpdate_AvatarSetAlsoNormalizesAvatarAsset(t *testing.T) {
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("character.update"),
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(
			`{"character_id":"char-1","fields":{"avatar_set_id":"avatar_set_v1"}}`,
		),
	}

	decision := Decide(State{
		Created:       true,
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "missing",
	}, cmd, nil)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload UpdatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Fields["avatar_set_id"] == "" {
		t.Fatal("expected avatar_set_id to be normalized")
	}
	if payload.Fields["avatar_asset_id"] == "" {
		t.Fatal("expected avatar_asset_id to be normalized alongside avatar_set_id")
	}
}

func TestDecideCharacterUpdate_InvalidAvatarAssetRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("character.update"),
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(
			`{"character_id":"char-1","fields":{"avatar_asset_id":"missing"}}`,
		),
	}

	decision := Decide(State{
		Created:       true,
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "001",
	}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterAvatarAssetInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterAvatarAssetInvalid)
	}
}
