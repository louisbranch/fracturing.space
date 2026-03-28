package projection

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection/testevent"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestApplyParticipantUnbound_RejectsMismatch(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	payload := testevent.ParticipantUnboundPayload{UserID: "user-2"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUnbound, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestApplySeatReassigned_UpdatesClaims(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-old"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Events: testEventRegistry(t), Participant: participantStore, Campaign: campaignStore, ClaimIndex: claimStore}

	payload := testevent.SeatReassignedPayload{UserID: "user-new", PriorUserID: "user-old"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 12, 30, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeSeatReassigned, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := participantStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if updated.UserID != "user-new" {
		t.Fatalf("UserID = %q, want %q", updated.UserID, "user-new")
	}
	if !claimStore.lastPutOK || claimStore.lastPut.UserID != "user-new" {
		t.Fatal("expected claim to be recorded for new user")
	}
}

func TestParseParticipantRole(t *testing.T) {
	tests := []struct {
		input string
		want  participant.Role
		err   bool
	}{
		{"gm", participant.RoleGM, false},
		{"PLAYER", participant.RolePlayer, false},
		{"GM", participant.RoleGM, false},
		{"player", participant.RolePlayer, false},
		{"", participant.RoleUnspecified, true},
		{"observer", participant.RoleUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseParticipantRole(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseParticipantRole(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseParticipantRole(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseParticipantController(t *testing.T) {
	tests := []struct {
		input string
		want  participant.Controller
		err   bool
	}{
		{"human", participant.ControllerHuman, false},
		{"AI", participant.ControllerAI, false},
		{"CONTROLLER_HUMAN", participant.ControllerHuman, false},
		{"CONTROLLER_AI", participant.ControllerAI, false},
		{"", participant.ControllerUnspecified, true},
		{"bot", participant.ControllerUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseParticipantController(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseParticipantController(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseParticipantController(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestApplyParticipantUpdated(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	pStore.participants["camp-1:part-1"] = storage.ParticipantRecord{
		ID: "part-1", CampaignID: "camp-1", UserID: "user-1",
		Name: "Old Name", Role: participant.RolePlayer,
		Controller: participant.ControllerHuman, CampaignAccess: participant.CampaignAccessMember,
	}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: pStore, Campaign: cStore}

	payload := testevent.ParticipantUpdatedPayload{Fields: map[string]any{
		"name":            "New Name",
		"role":            "GM",
		"controller":      "AI",
		"campaign_access": "OWNER",
		"user_id":         "user-2",
	}}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 15, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := pStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Role != participant.RoleGM {
		t.Fatalf("Role = %v, want GM", updated.Role)
	}
	if updated.Controller != participant.ControllerAI {
		t.Fatalf("Controller = %v, want AI", updated.Controller)
	}
	if updated.CampaignAccess != participant.CampaignAccessOwner {
		t.Fatalf("CampaignAccess = %v, want OWNER", updated.CampaignAccess)
	}
	if updated.UserID != "user-2" {
		t.Fatalf("UserID = %q, want %q", updated.UserID, "user-2")
	}
}

func TestApplyParticipantUpdated_EmptyFields(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	cStore := newProjectionCampaignStore()
	applier := Applier{Participant: pStore, Campaign: cStore}

	payload := testevent.ParticipantUpdatedPayload{Fields: map[string]any{}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply with empty fields should succeed: %v", err)
	}
}

func TestApplyParticipantUpdated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"name": "x"}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing participant store")
	}
}

// --- applyParticipantLeft tests ---

func TestApplyParticipantLeft(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	pStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", ParticipantCount: 3}
	applier := Applier{Participant: pStore, Campaign: cStore}

	data, _ := json.Marshal(testevent.ParticipantLeftPayload{})
	stamp := time.Date(2026, 2, 11, 15, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeLeft, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := pStore.GetParticipant(ctx, "camp-1", "part-1"); err == nil {
		t.Fatal("expected participant to be deleted")
	}
	c, _ := cStore.Get(ctx, "camp-1")
	// Count is derived from actual store records (0 remaining), not arithmetic.
	if c.ParticipantCount != 0 {
		t.Fatalf("ParticipantCount = %d, want 0", c.ParticipantCount)
	}
}

func TestApplyParticipantLeft_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantLeftPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeLeft, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyParticipantLeft_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantLeftPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: participant.EventTypeLeft, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

// --- applyParticipantBound tests ---

func TestApplyParticipantBound(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	pStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Participant: pStore, Campaign: cStore, ClaimIndex: claimStore}

	payload := testevent.ParticipantBoundPayload{UserID: "user-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 16, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeBound, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := pStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if updated.UserID != "user-1" {
		t.Fatalf("UserID = %q, want %q", updated.UserID, "user-1")
	}
	if !claimStore.lastPutOK || claimStore.lastPut.UserID != "user-1" {
		t.Fatal("expected claim to be recorded")
	}
}

func TestApplyParticipantBound_MissingUserID(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	pStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	cStore := newProjectionCampaignStore()
	applier := Applier{Participant: pStore, Campaign: cStore}

	payload := testevent.ParticipantBoundPayload{UserID: ""}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeBound, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing user ID")
	}
}

func TestApplyParticipantBound_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantBoundPayload{UserID: "u1"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeBound, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

// --- applyParticipantUnbound tests ---

func TestApplyParticipantUnbound_Success(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	pStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-1"}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Participant: pStore, Campaign: cStore, ClaimIndex: claimStore}

	payload := testevent.ParticipantUnboundPayload{UserID: "user-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 16, 30, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUnbound, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := pStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if updated.UserID != "" {
		t.Fatalf("UserID = %q, want empty", updated.UserID)
	}
	if len(claimStore.deleted) != 1 || claimStore.deleted[0] != "user-1" {
		t.Fatal("expected claim to be deleted for user-1")
	}
}

func TestApplyParticipantUnbound_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantUnboundPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUnbound, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyParticipantUnbound_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantUnboundPayload{})
	evt := testevent.Event{CampaignID: "", EntityID: "part-1", Type: participant.EventTypeUnbound, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

// --- applyInviteClaimed tests ---

func TestApplyParticipantJoined(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", ParticipantCount: 0}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	payload := testevent.ParticipantJoinedPayload{
		UserID:         "user-1",
		Name:           "Alice",
		Role:           "player",
		Controller:     "human",
		CampaignAccess: "member",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeJoined, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	p, err := participantStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if p.Name != "Alice" {
		t.Fatalf("Name = %q, want Alice", p.Name)
	}
	c, _ := campaignStore.Get(ctx, "camp-1")
	if c.ParticipantCount != 1 {
		t.Fatalf("ParticipantCount = %d, want 1", c.ParticipantCount)
	}
}

func TestApplyParticipantJoined_IdempotentCount(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", ParticipantCount: 0}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	payload := testevent.ParticipantJoinedPayload{
		UserID: "user-1", Name: "Alice", Role: "player",
		Controller: "human", CampaignAccess: "member",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeJoined, PayloadJSON: data, Timestamp: stamp}

	// Apply the same event twice (idempotent replay).
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	c, _ := campaignStore.Get(ctx, "camp-1")
	if c.ParticipantCount != 1 {
		t.Fatalf("ParticipantCount = %d, want 1 (idempotent)", c.ParticipantCount)
	}
}

func TestApplyParticipantJoined_MissingStores(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantJoinedPayload{UserID: "u", Name: "A", Role: "player", Controller: "human", CampaignAccess: "member"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeJoined, PayloadJSON: data}

	if err := (Applier{Campaign: newProjectionCampaignStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing participant store")
	}
	if err := (Applier{Participant: newProjectionParticipantStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyParticipantJoined_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantJoinedPayload{UserID: "u", Name: "A", Role: "player", Controller: "human", CampaignAccess: "member"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: participant.EventTypeJoined, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

func TestApplyParticipantJoined_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantJoinedPayload{UserID: "u", Name: "A", Role: "player", Controller: "human", CampaignAccess: "member"})
	evt := testevent.Event{CampaignID: "", EntityID: "part-1", Type: participant.EventTypeJoined, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

// --- applySeatReassigned additional tests ---

func TestApplySeatReassigned_MissingStores(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "user-new"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeSeatReassigned, PayloadJSON: data}

	if err := (Applier{Campaign: newProjectionCampaignStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing participant store")
	}
	if err := (Applier{Participant: newProjectionParticipantStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplySeatReassigned_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "user-new"})
	evt := testevent.Event{CampaignID: "", EntityID: "part-1", Type: participant.EventTypeSeatReassigned, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplySeatReassigned_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "user-new"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: participant.EventTypeSeatReassigned, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

func TestApplySeatReassigned_MissingUserID(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-old"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "", PriorUserID: "user-old"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeSeatReassigned, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing user ID")
	}
}

func TestApplySeatReassigned_PriorUserMismatch(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-old"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "user-new", PriorUserID: "user-wrong"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeSeatReassigned, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for prior user mismatch")
	}
}

func TestApplySeatReassigned_NoClaims(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-old"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Events: testEventRegistry(t), Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "user-new"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeSeatReassigned, PayloadJSON: data, Timestamp: time.Now()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	p, _ := participantStore.GetParticipant(ctx, "camp-1", "part-1")
	if p.UserID != "user-new" {
		t.Fatalf("UserID = %q, want user-new", p.UserID)
	}
}

// --- applyInviteCreated additional tests ---

func TestApplyParticipantUpdated_InvalidUserIDType(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"user_id": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid user_id type")
	}
}

func TestApplyParticipantUpdated_InvalidNameType(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"name": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid display_name type")
	}
}

func TestApplyParticipantUpdated_EmptyName(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"name": "  "}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for empty display name")
	}
}

func TestApplyParticipantUpdated_InvalidRoleType(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"role": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid role type")
	}
}

func TestApplyParticipantUpdated_InvalidControllerType(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"controller": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid controller type")
	}
}

func TestApplyParticipantUpdated_InvalidAccessType(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"campaign_access": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid campaign_access type")
	}
}

// --- applyCharacterUpdated type assertion errors ---

func TestApplyParticipantLeft_MissingCampaignStore(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeLeft, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyParticipantLeft_MissingCampaignID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "  ", EntityID: "part-1", Type: participant.EventTypeLeft, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyParticipantLeft_ZeroCount(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", ParticipantCount: 0}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeLeft, PayloadJSON: []byte("{}"), Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	c, _ := campaignStore.Get(ctx, "camp-1")
	if c.ParticipantCount != 0 {
		t.Fatalf("ParticipantCount = %d, want 0", c.ParticipantCount)
	}
}

// --- applyParticipantBound missing branches ---

func TestApplyParticipantBound_MissingCampaignStore(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeBound, PayloadJSON: []byte(`{"user_id":"u1"}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyParticipantBound_MissingCampaignID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "  ", EntityID: "part-1", Type: participant.EventTypeBound, PayloadJSON: []byte(`{"user_id":"u1"}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyParticipantBound_MissingEntityID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "  ", Type: participant.EventTypeBound, PayloadJSON: []byte(`{"user_id":"u1"}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity id")
	}
}

func TestApplyParticipantBound_InvalidJSON(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeBound, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyParticipantUnbound missing branches ---

func TestApplyParticipantUnbound_MissingEntityID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "  ", Type: participant.EventTypeUnbound, PayloadJSON: []byte(`{"user_id":"u1"}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity id")
	}
}

func TestApplyParticipantUnbound_InvalidJSON(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUnbound, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyParticipantUnbound_NilClaimIndex(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "u1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}
	payload := testevent.ParticipantUnboundPayload{}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUnbound, PayloadJSON: data, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	p, _ := participantStore.GetParticipant(ctx, "camp-1", "part-1")
	if p.UserID != "" {
		t.Fatalf("UserID = %q, want empty", p.UserID)
	}
}

// --- applyInviteClaimed missing branches ---

func TestApplyParticipantJoined_InvalidJSON(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeJoined, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyCharacterUpdated missing branches ---

func TestApplyParticipantUpdated_MissingCampaignID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "  ", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: []byte(`{"fields":{"name":"X"}}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyParticipantUpdated_MissingEntityID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "  ", Type: participant.EventTypeUpdated, PayloadJSON: []byte(`{"fields":{"name":"X"}}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity id")
	}
}

func TestApplyParticipantUpdated_InvalidJSON(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyParticipantUpdated_InvalidRole(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}
	payload := testevent.ParticipantUpdatedPayload{Fields: map[string]any{"role": "ALIEN"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestApplyParticipantUpdated_InvalidController(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}
	payload := testevent.ParticipantUpdatedPayload{Fields: map[string]any{"controller": "ALIEN"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid controller")
	}
}

func TestApplyParticipantUpdated_InvalidAccess(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}
	payload := testevent.ParticipantUpdatedPayload{Fields: map[string]any{"campaign_access": "ALIEN"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid access")
	}
}

// --- applyParticipantJoined parser error branches ---

func TestApplyParticipantJoined_InvalidRole(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.ParticipantJoinedPayload{Name: "A", Role: "ALIEN", Controller: "HUMAN", CampaignAccess: "READ_ONLY"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeJoined, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestApplyParticipantJoined_InvalidController(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.ParticipantJoinedPayload{Name: "A", Role: "PLAYER", Controller: "ALIEN", CampaignAccess: "READ_ONLY"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeJoined, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid controller")
	}
}

func TestApplyParticipantJoined_InvalidAccess(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.ParticipantJoinedPayload{Name: "A", Role: "PLAYER", Controller: "HUMAN", CampaignAccess: "ALIEN"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeJoined, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid access")
	}
}

// --- applyParticipantBound with ClaimIndex ---

func TestApplyParticipantBound_WithClaimIndex(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Participant: participantStore, Campaign: campaignStore, ClaimIndex: claimStore}
	payload := testevent.ParticipantBoundPayload{UserID: "u1"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeBound, PayloadJSON: data, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !claimStore.lastPutOK {
		t.Fatal("expected claim to be written")
	}
}

// --- applyParticipantUnbound with ClaimIndex ---

func TestApplyParticipantUnbound_WithClaimIndex(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "u1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Participant: participantStore, Campaign: campaignStore, ClaimIndex: claimStore}
	payload := testevent.ParticipantUnboundPayload{UserID: "u1"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: participant.EventTypeUnbound, PayloadJSON: data, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(claimStore.deleted) != 1 || claimStore.deleted[0] != "u1" {
		t.Fatalf("expected claim deletion for u1, got %v", claimStore.deleted)
	}
}

// --- applyDaggerheartCharacterProfileReplaced validation branches ---
