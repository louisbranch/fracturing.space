package participant

import (
	"encoding/json"
	"testing"
	"time"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestDecideParticipantJoin_EmitsParticipantJoinedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":" user-1 ","name":"  Alice  ","role":"PLAYER","controller":"CONTROLLER_HUMAN","campaign_access":"CAMPAIGN_ACCESS_MEMBER"}`),
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
	if evt.Type != event.Type("participant.joined") {
		t.Fatalf("event type = %s, want %s", evt.Type, "participant.joined")
	}
	if evt.EntityType != "participant" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "participant")
	}
	if evt.EntityID != "p-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "p-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}
	if evt.ActorType != event.ActorTypeSystem {
		t.Fatalf("event actor type = %s, want %s", evt.ActorType, event.ActorTypeSystem)
	}

	var payload JoinPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.ParticipantID != "p-1" {
		t.Fatalf("payload participant id = %s, want %s", payload.ParticipantID, "p-1")
	}
	if payload.UserID != "user-1" {
		t.Fatalf("payload user id = %s, want %s", payload.UserID, "user-1")
	}
	if payload.Name != "Alice" {
		t.Fatalf("payload display name = %s, want %s", payload.Name, "Alice")
	}
	if payload.Role != "player" {
		t.Fatalf("payload role = %s, want %s", payload.Role, "player")
	}
	if payload.Controller != "human" {
		t.Fatalf("payload controller = %s, want %s", payload.Controller, "human")
	}
	if payload.CampaignAccess != "member" {
		t.Fatalf("payload campaign access = %s, want %s", payload.CampaignAccess, "member")
	}
}

func TestDecideParticipantJoin_DefaultsControllerAndAccess(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","name":"Alice","role":"GM"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload JoinPayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Controller != "human" {
		t.Fatalf("payload controller = %s, want %s", payload.Controller, "human")
	}
	if payload.CampaignAccess != "member" {
		t.Fatalf("payload campaign access = %s, want %s", payload.CampaignAccess, "member")
	}
}

func TestDecideParticipantJoin_WhenAlreadyJoined_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","name":"Alice","role":"PLAYER"}`),
	}

	decision := Decide(State{Joined: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantAlreadyJoined {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantAlreadyJoined)
	}
}

func TestDecideParticipantJoin_MissingParticipantID_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"  ","name":"Alice","role":"PLAYER"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantIDRequired)
	}
}

func TestDecideParticipantJoin_MissingName_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","name":"  ","role":"PLAYER"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantNameEmpty {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantNameEmpty)
	}
}

func TestDecideParticipantJoin_InvalidRole_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","name":"Alice","role":"ALIEN"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantRoleInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantRoleInvalid)
	}
}

func TestDecideParticipantJoin_InvalidController_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","name":"Alice","role":"PLAYER","controller":"ALIEN"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantControllerInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantControllerInvalid)
	}
}

func TestDecideParticipantJoin_InvalidAccess_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","name":"Alice","role":"PLAYER","campaign_access":"ALIEN"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantAccessInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantAccessInvalid)
	}
}

func TestDecideParticipantUpdate_EmitsParticipantUpdatedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","fields":{"user_id":" user-1 ","name":"  Alice  ","role":"ROLE_PLAYER","controller":"CONTROLLER_HUMAN","campaign_access":"CAMPAIGN_ACCESS_MEMBER"}}`),
	}

	decision := Decide(State{Joined: true}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("participant.updated") {
		t.Fatalf("event type = %s, want %s", evt.Type, "participant.updated")
	}
	if evt.EntityID != "p-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "p-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload UpdatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.ParticipantID != "p-1" {
		t.Fatalf("payload participant id = %s, want %s", payload.ParticipantID, "p-1")
	}
	if payload.Fields["user_id"] != "user-1" {
		t.Fatalf("payload user id = %s, want %s", payload.Fields["user_id"], "user-1")
	}
	if payload.Fields["name"] != "Alice" {
		t.Fatalf("payload display name = %s, want %s", payload.Fields["name"], "Alice")
	}
	if payload.Fields["role"] != "player" {
		t.Fatalf("payload role = %s, want %s", payload.Fields["role"], "player")
	}
	if payload.Fields["controller"] != "human" {
		t.Fatalf("payload controller = %s, want %s", payload.Fields["controller"], "human")
	}
	if payload.Fields["campaign_access"] != "member" {
		t.Fatalf("payload campaign access = %s, want %s", payload.Fields["campaign_access"], "member")
	}
}

func TestDecideParticipantUpdate_EmptyFieldsRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","fields":{}}`),
	}

	decision := Decide(State{Joined: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantUpdateEmpty {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantUpdateEmpty)
	}
}

func TestDecideParticipantUpdate_InvalidFieldRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","fields":{"unknown":"value"}}`),
	}

	decision := Decide(State{Joined: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantUpdateFieldInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantUpdateFieldInvalid)
	}
}

func TestDecideParticipantUpdate_WhenNotJoinedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","fields":{"name":"Alice"}}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantNotJoined {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantNotJoined)
	}
}

func TestDecideParticipantLeave_EmitsParticipantLeftEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.leave"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","reason":"  done  "}`),
	}

	decision := Decide(State{Joined: true}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("participant.left") {
		t.Fatalf("event type = %s, want %s", evt.Type, "participant.left")
	}
	if evt.EntityID != "p-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "p-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload LeavePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.ParticipantID != "p-1" {
		t.Fatalf("payload participant id = %s, want %s", payload.ParticipantID, "p-1")
	}
	if payload.Reason != "done" {
		t.Fatalf("payload reason = %s, want %s", payload.Reason, "done")
	}
}

func TestDecideParticipantLeave_WhenNotJoinedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.leave"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantNotJoined {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantNotJoined)
	}
}

func TestDecideParticipantLeave_MissingParticipantIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.leave"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":" "}`),
	}

	decision := Decide(State{Joined: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantIDRequired)
	}
}

func TestDecideParticipantBind_EmitsParticipantBoundEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.bind"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":" user-2 "}`),
	}

	decision := Decide(State{Joined: true}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("participant.bound") {
		t.Fatalf("event type = %s, want %s", evt.Type, "participant.bound")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload BindPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.ParticipantID != "p-1" {
		t.Fatalf("payload participant id = %s, want %s", payload.ParticipantID, "p-1")
	}
	if payload.UserID != "user-2" {
		t.Fatalf("payload user id = %s, want %s", payload.UserID, "user-2")
	}
}

func TestDecideParticipantBind_MissingUserIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.bind"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":" "}`),
	}

	decision := Decide(State{Joined: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantUserIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantUserIDRequired)
	}
}

func TestDecideParticipantBind_WhenNotJoinedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.bind"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"user-2"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantNotJoined {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantNotJoined)
	}
}

func TestDecideParticipantUnbind_EmitsParticipantUnboundEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.unbind"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"user-1","reason":"  done  "}`),
	}

	decision := Decide(State{Joined: true, UserID: "user-1"}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("participant.unbound") {
		t.Fatalf("event type = %s, want %s", evt.Type, "participant.unbound")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload UnbindPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.ParticipantID != "p-1" {
		t.Fatalf("payload participant id = %s, want %s", payload.ParticipantID, "p-1")
	}
	if payload.UserID != "user-1" {
		t.Fatalf("payload user id = %s, want %s", payload.UserID, "user-1")
	}
	if payload.Reason != "done" {
		t.Fatalf("payload reason = %s, want %s", payload.Reason, "done")
	}
}

func TestDecideParticipantUnbind_UserIDMismatchRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.unbind"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"user-2"}`),
	}

	decision := Decide(State{Joined: true, UserID: "user-1"}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantUserIDMismatch {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantUserIDMismatch)
	}
}

func TestDecideSeatReassign_EmitsSeatReassignedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	commandTypes := []command.Type{
		command.Type("seat.reassign"),
		command.Type("participant.seat.reassign"),
	}
	for _, cmdType := range commandTypes {
		t.Run(string(cmdType), func(t *testing.T) {
			cmd := command.Command{
				CampaignID:  "camp-1",
				Type:        cmdType,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"participant_id":"p-1","prior_user_id":"user-1","user_id":"user-2","reason":"  moved  "}`),
			}

			decision := Decide(State{Joined: true, UserID: "user-1"}, cmd, func() time.Time { return now })
			if len(decision.Rejections) != 0 {
				t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
			}
			if len(decision.Events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(decision.Events))
			}

			evt := decision.Events[0]
			if evt.Type != event.Type("participant.seat_reassigned") {
				t.Fatalf("event type = %s, want %s", evt.Type, "participant.seat_reassigned")
			}
			if !evt.Timestamp.Equal(now) {
				t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
			}

			var payload SeatReassignPayload
			if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if payload.ParticipantID != "p-1" {
				t.Fatalf("payload participant id = %s, want %s", payload.ParticipantID, "p-1")
			}
			if payload.PriorUserID != "user-1" {
				t.Fatalf("payload prior user id = %s, want %s", payload.PriorUserID, "user-1")
			}
			if payload.UserID != "user-2" {
				t.Fatalf("payload user id = %s, want %s", payload.UserID, "user-2")
			}
			if payload.Reason != "moved" {
				t.Fatalf("payload reason = %s, want %s", payload.Reason, "moved")
			}
		})
	}
}

func TestDecideSeatReassign_PriorUserMismatchRejected(t *testing.T) {
	commandTypes := []command.Type{
		command.Type("seat.reassign"),
		command.Type("participant.seat.reassign"),
	}
	for _, cmdType := range commandTypes {
		t.Run(string(cmdType), func(t *testing.T) {
			cmd := command.Command{
				CampaignID:  "camp-1",
				Type:        cmdType,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"participant_id":"p-1","prior_user_id":"user-2","user_id":"user-3"}`),
			}

			decision := Decide(State{Joined: true, UserID: "user-1"}, cmd, nil)
			if len(decision.Events) != 0 {
				t.Fatalf("expected no events, got %d", len(decision.Events))
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
			}
			if decision.Rejections[0].Code != rejectionCodeParticipantUserIDMismatch {
				t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantUserIDMismatch)
			}
		})
	}
}

func TestDecideSeatReassign_MissingUserIDRejected(t *testing.T) {
	commandTypes := []command.Type{
		command.Type("seat.reassign"),
		command.Type("participant.seat.reassign"),
	}
	for _, cmdType := range commandTypes {
		t.Run(string(cmdType), func(t *testing.T) {
			cmd := command.Command{
				CampaignID:  "camp-1",
				Type:        cmdType,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"participant_id":"p-1","user_id":" "}`),
			}

			decision := Decide(State{Joined: true}, cmd, nil)
			if len(decision.Events) != 0 {
				t.Fatalf("expected no events, got %d", len(decision.Events))
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
			}
			if decision.Rejections[0].Code != rejectionCodeParticipantUserIDRequired {
				t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantUserIDRequired)
			}
		})
	}
}

func TestDecideParticipantJoin_DefaultsAvatarSelection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","name":"Alice","role":"PLAYER"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload JoinPayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.AvatarSetID != assetcatalog.AvatarSetBlankV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.AvatarSetID, assetcatalog.AvatarSetBlankV1)
	}
	if payload.AvatarAssetID != "000" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.AvatarAssetID, "000")
	}
}

func TestDecideParticipantUpdate_UserIDClearedSetsBlankAvatar(t *testing.T) {
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("participant.update"),
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(
			`{"participant_id":"p-1","fields":{"user_id":""}}`,
		),
	}

	decision := Decide(State{
		Joined:        true,
		UserID:        "user-1",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "007",
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
	if payload.Fields["user_id"] != "" {
		t.Fatalf("user_id = %q, want empty", payload.Fields["user_id"])
	}
	if payload.Fields["avatar_set_id"] != assetcatalog.AvatarSetBlankV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Fields["avatar_set_id"], assetcatalog.AvatarSetBlankV1)
	}
	if payload.Fields["avatar_asset_id"] != "000" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.Fields["avatar_asset_id"], "000")
	}
}

func TestDecideParticipantJoin_InvalidAvatarSetRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"user-1","name":"Alice","role":"PLAYER","avatar_set_id":"missing"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantAvatarSetInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantAvatarSetInvalid)
	}
}

func TestDecideParticipantUpdate_AvatarSetAlsoNormalizesAvatarAsset(t *testing.T) {
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("participant.update"),
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(
			`{"participant_id":"p-1","fields":{"avatar_set_id":"avatar_set_v1"}}`,
		),
	}

	decision := Decide(State{
		Joined:        true,
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

func TestDecideParticipantUpdate_InvalidAvatarAssetRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("participant.update"),
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(
			`{"participant_id":"p-1","fields":{"avatar_asset_id":"missing"}}`,
		),
	}

	decision := Decide(State{
		Joined:        true,
		UserID:        "user-1",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "001",
	}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeParticipantAvatarAssetInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeParticipantAvatarAssetInvalid)
	}
}

func TestDecide_UnrecognizedCommandTypeRejected(t *testing.T) {
	decision := Decide(State{}, command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("participant.nonexistent"),
	}, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "COMMAND_TYPE_UNSUPPORTED" {
		t.Fatalf("rejection code = %s, want COMMAND_TYPE_UNSUPPORTED", decision.Rejections[0].Code)
	}
}

func TestDecide_MalformedJoinPayloadRejected(t *testing.T) {
	decision := Decide(State{}, command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		PayloadJSON: []byte(`{corrupt`),
	}, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "PAYLOAD_DECODE_FAILED" {
		t.Fatalf("rejection code = %s, want PAYLOAD_DECODE_FAILED", decision.Rejections[0].Code)
	}
}

func TestParticipantDecisionHandlersCoverSupportedCommands(t *testing.T) {
	expected := []command.Type{
		CommandTypeJoin,
		CommandTypeUpdate,
		CommandTypeLeave,
		CommandTypeBind,
		CommandTypeUnbind,
		CommandTypeSeatReassign,
		CommandTypeSeatReassignLegacy,
	}
	if len(participantDecisionHandlers) != len(expected) {
		t.Fatalf("handler count = %d, expected count = %d", len(participantDecisionHandlers), len(expected))
	}
	expectedSet := make(map[command.Type]struct{}, len(expected))
	for _, cmdType := range expected {
		expectedSet[cmdType] = struct{}{}
		if _, ok := participantDecisionHandlers[cmdType]; !ok {
			t.Fatalf("missing handler for command %s", cmdType)
		}
	}
	for cmdType := range participantDecisionHandlers {
		if _, ok := expectedSet[cmdType]; !ok {
			t.Fatalf("unexpected handler for command %s", cmdType)
		}
	}
}
