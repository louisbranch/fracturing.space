package campaignbootstrap

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func TestDecide_EmitsCampaignAndParticipantEvents(t *testing.T) {
	now := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       campaign.CommandTypeCreateWithParticipants,
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(`{
			"campaign":{"name":"Sunfall","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN"},
			"participants":[{"participant_id":" p-1 ","user_id":" user-1 ","name":" Alice ","role":"PLAYER"}]
		}`),
	}

	decision := Decide(campaign.State{}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(decision.Events))
	}

	campaignEvent := decision.Events[0]
	if campaignEvent.Type != campaign.EventTypeCreated {
		t.Fatalf("campaign event type = %s, want %s", campaignEvent.Type, campaign.EventTypeCreated)
	}
	if !campaignEvent.Timestamp.Equal(now) {
		t.Fatalf("campaign event timestamp = %s, want %s", campaignEvent.Timestamp, now)
	}

	joinEvent := decision.Events[1]
	if joinEvent.Type != participant.EventTypeJoined {
		t.Fatalf("participant event type = %s, want %s", joinEvent.Type, participant.EventTypeJoined)
	}
	if joinEvent.EntityType != "participant" {
		t.Fatalf("participant event entity_type = %s, want participant", joinEvent.EntityType)
	}
	if joinEvent.EntityID != "p-1" {
		t.Fatalf("participant event entity_id = %s, want p-1", joinEvent.EntityID)
	}
	if !joinEvent.Timestamp.Equal(now) {
		t.Fatalf("participant event timestamp = %s, want %s", joinEvent.Timestamp, now)
	}

	var joinPayload participant.JoinPayload
	if err := json.Unmarshal(joinEvent.PayloadJSON, &joinPayload); err != nil {
		t.Fatalf("decode join payload: %v", err)
	}
	if joinPayload.ParticipantID != "p-1" {
		t.Fatalf("participant_id = %q, want %q", joinPayload.ParticipantID, "p-1")
	}
	if joinPayload.UserID != "user-1" {
		t.Fatalf("user_id = %q, want %q", joinPayload.UserID, "user-1")
	}
	if joinPayload.Name != "Alice" {
		t.Fatalf("name = %q, want %q", joinPayload.Name, "Alice")
	}
	if joinPayload.Role != "player" {
		t.Fatalf("role = %q, want %q", joinPayload.Role, "player")
	}
	if joinPayload.Controller != "human" {
		t.Fatalf("controller = %q, want %q", joinPayload.Controller, "human")
	}
	if joinPayload.CampaignAccess != "member" {
		t.Fatalf("campaign_access = %q, want %q", joinPayload.CampaignAccess, "member")
	}
}

func TestDecide_EmptyParticipantsRejected(t *testing.T) {
	decision := Decide(campaign.State{}, command.Command{
		CampaignID: "camp-1",
		Type:       campaign.CommandTypeCreateWithParticipants,
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(`{
			"campaign":{"name":"Sunfall","game_system":"daggerheart","gm_mode":"human"},
			"participants":[]
		}`),
	}, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "CAMPAIGN_PARTICIPANTS_REQUIRED" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "CAMPAIGN_PARTICIPANTS_REQUIRED")
	}
}

func TestDecide_DuplicateParticipantIDsRejected(t *testing.T) {
	decision := Decide(campaign.State{}, command.Command{
		CampaignID: "camp-1",
		Type:       campaign.CommandTypeCreateWithParticipants,
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(`{
			"campaign":{"name":"Sunfall","game_system":"daggerheart","gm_mode":"human"},
			"participants":[
				{"participant_id":"p-1","user_id":"user-1","name":"Alice","role":"player"},
				{"participant_id":" p-1 ","user_id":"user-2","name":"Bob","role":"player"}
			]
		}`),
	}, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "CAMPAIGN_PARTICIPANT_DUPLICATE" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "CAMPAIGN_PARTICIPANT_DUPLICATE")
	}
}

func TestDecide_JoinValidationRejected(t *testing.T) {
	decision := Decide(campaign.State{}, command.Command{
		CampaignID: "camp-1",
		Type:       campaign.CommandTypeCreateWithParticipants,
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(`{
			"campaign":{"name":"Sunfall","game_system":"daggerheart","gm_mode":"human"},
			"participants":[{"participant_id":" ","user_id":"user-1","name":"Alice","role":"player"}]
		}`),
	}, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "PARTICIPANT_ID_REQUIRED" {
		t.Fatalf("rejection code = %s, want PARTICIPANT_ID_REQUIRED", decision.Rejections[0].Code)
	}
}

func TestDecide_WhenCampaignAlreadyCreatedRejected(t *testing.T) {
	decision := Decide(campaign.State{Created: true}, command.Command{
		CampaignID: "camp-1",
		Type:       campaign.CommandTypeCreateWithParticipants,
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(`{
			"campaign":{"name":"Sunfall","game_system":"daggerheart","gm_mode":"human"},
			"participants":[{"participant_id":"p-1","user_id":"user-1","name":"Alice","role":"player"}]
		}`),
	}, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "CAMPAIGN_ALREADY_EXISTS" {
		t.Fatalf("rejection code = %s, want CAMPAIGN_ALREADY_EXISTS", decision.Rejections[0].Code)
	}
}

func TestDecide_MalformedPayloadRejected(t *testing.T) {
	decision := Decide(campaign.State{}, command.Command{
		CampaignID:  "camp-1",
		Type:        campaign.CommandTypeCreateWithParticipants,
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{`),
	}, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != command.RejectionCodePayloadDecodeFailed {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, command.RejectionCodePayloadDecodeFailed)
	}
}
