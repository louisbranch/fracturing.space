package projection

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection/testevent"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestApplyInviteCreated_UsesEntityID(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	inviteStore := newFakeInviteStore()
	applier := Applier{Campaign: campaignStore, Invite: inviteStore}

	payload := testevent.InviteCreatedPayload{
		InviteID:      "",
		ParticipantID: "part-1",
		Status:        "pending",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteCreated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	inv, err := inviteStore.GetInvite(ctx, "inv-1")
	if err != nil {
		t.Fatalf("get invite: %v", err)
	}
	if inv.ParticipantID != "part-1" {
		t.Fatalf("ParticipantID = %q, want %q", inv.ParticipantID, "part-1")
	}
	updatedCampaign, err := campaignStore.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if !updatedCampaign.UpdatedAt.Equal(stamp) {
		t.Fatalf("UpdatedAt = %v, want %v", updatedCampaign.UpdatedAt, stamp)
	}
}

func TestApplyInviteRevoked(t *testing.T) {
	ctx := context.Background()
	inviteStore := newFakeInviteStore()
	inviteStore.invites["inv-1"] = storage.InviteRecord{ID: "inv-1", CampaignID: "camp-1", Status: invite.StatusPending}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Invite: inviteStore, Campaign: campaignStore}

	payload := testevent.InviteRevokedPayload{InviteID: "inv-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteRevoked, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if inviteStore.updatedStatus["inv-1"] != invite.StatusRevoked {
		t.Fatalf("expected invite status revoked, got %v", inviteStore.updatedStatus["inv-1"])
	}
	updated, _ := campaignStore.Get(ctx, "camp-1")
	if !updated.UpdatedAt.Equal(stamp) {
		t.Fatalf("campaign UpdatedAt = %v, want %v", updated.UpdatedAt, stamp)
	}
}

func TestApplyInviteRevoked_MissingStores(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteRevokedPayload{InviteID: "inv-1"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteRevoked, PayloadJSON: data}

	// Missing invite store
	if err := (Applier{Campaign: newProjectionCampaignStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing invite store")
	}
	// Missing campaign store
	if err := (Applier{Invite: newFakeInviteStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyInviteRevoked_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteRevokedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeInviteRevoked, PayloadJSON: data}
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

func TestApplyInviteUpdated(t *testing.T) {
	ctx := context.Background()
	inviteStore := newFakeInviteStore()
	applier := Applier{Invite: inviteStore}

	payload := testevent.InviteUpdatedPayload{InviteID: "inv-1", Status: "REVOKED"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 11, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteUpdated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if inviteStore.updatedStatus["inv-1"] != invite.StatusRevoked {
		t.Fatalf("expected invite status revoked, got %v", inviteStore.updatedStatus["inv-1"])
	}
}

func TestApplyInviteUpdated_EntityIDFallback(t *testing.T) {
	ctx := context.Background()
	inviteStore := newFakeInviteStore()
	applier := Applier{Invite: inviteStore}

	payload := testevent.InviteUpdatedPayload{InviteID: "", Status: "CLAIMED"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-2", Type: testevent.TypeInviteUpdated, PayloadJSON: data, Timestamp: time.Now()}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, ok := inviteStore.updatedStatus["inv-2"]; !ok {
		t.Fatal("expected invite update to use EntityID as fallback")
	}
}

func TestApplyInviteUpdated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteUpdatedPayload{InviteID: "inv-1", Status: "PENDING"})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteUpdated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing invite store")
	}
}

func TestApplyInviteUpdated_MissingInviteID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteUpdatedPayload{InviteID: "", Status: "PENDING"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeInviteUpdated, PayloadJSON: data}
	applier := Applier{Invite: newFakeInviteStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing invite ID")
	}
}

func TestParseInviteStatus(t *testing.T) {
	tests := []struct {
		input string
		want  invite.Status
		err   bool
	}{
		{"pending", invite.StatusPending, false},
		{"CLAIMED", invite.StatusClaimed, false},
		{"revoked", invite.StatusRevoked, false},
		{"PENDING", invite.StatusPending, false},
		{"", invite.StatusUnspecified, true},
		{"unknown", invite.StatusUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseInviteStatus(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseInviteStatus(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseInviteStatus(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestApplyInviteClaimed(t *testing.T) {
	ctx := context.Background()
	inviteStore := newFakeInviteStore()
	inviteStore.invites["inv-1"] = storage.InviteRecord{ID: "inv-1", CampaignID: "camp-1", Status: invite.StatusPending}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Invite: inviteStore, Campaign: cStore}

	payload := testevent.InviteClaimedPayload{InviteID: "inv-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 17, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteClaimed, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if inviteStore.updatedStatus["inv-1"] != invite.StatusClaimed {
		t.Fatalf("status = %v, want claimed", inviteStore.updatedStatus["inv-1"])
	}
	c, _ := cStore.Get(ctx, "camp-1")
	if !c.UpdatedAt.Equal(stamp) {
		t.Fatalf("campaign UpdatedAt = %v, want %v", c.UpdatedAt, stamp)
	}
}

func TestApplyInviteClaimed_MismatchID(t *testing.T) {
	ctx := context.Background()
	inviteStore := newFakeInviteStore()
	cStore := newProjectionCampaignStore()
	applier := Applier{Invite: inviteStore, Campaign: cStore}

	payload := testevent.InviteClaimedPayload{InviteID: "inv-other"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteClaimed, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invite ID mismatch")
	}
}

func TestApplyInviteClaimed_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteClaimedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteClaimed, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyInviteClaimed_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteClaimedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeInviteClaimed, PayloadJSON: data}
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

// --- applyCharacterCreated tests ---

func TestApplyInviteCreated_MissingStores(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteCreatedPayload{InviteID: "inv-1", ParticipantID: "p1", Status: "pending"})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteCreated, PayloadJSON: data}

	if err := (Applier{Campaign: newProjectionCampaignStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing invite store")
	}
	if err := (Applier{Invite: newFakeInviteStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyInviteCreated_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteCreatedPayload{InviteID: "inv-1", ParticipantID: "p1", Status: "pending"})
	evt := testevent.Event{CampaignID: "", Type: testevent.TypeInviteCreated, PayloadJSON: data}
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplyInviteCreated_MissingInviteID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteCreatedPayload{InviteID: "", ParticipantID: "p1", Status: "pending"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeInviteCreated, PayloadJSON: data}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Invite: newFakeInviteStore(), Campaign: campaignStore}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing invite ID")
	}
}

func TestApplyInviteCreated_MissingParticipantID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteCreatedPayload{InviteID: "inv-1", ParticipantID: "", Status: "pending"})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteCreated, PayloadJSON: data}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Invite: newFakeInviteStore(), Campaign: campaignStore}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing participant ID")
	}
}

// --- applySessionStarted additional tests ---

func TestApplyInviteClaimed_MissingCampaignStore(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteClaimed, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyInviteClaimed_InvalidJSON(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteClaimed, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyInviteRevoked missing branches ---

func TestApplyInviteRevoked_MismatchID(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.InviteRevokedPayload{InviteID: "inv-2"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteRevoked, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invite id mismatch")
	}
}

func TestApplyInviteRevoked_InvalidJSON(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteRevoked, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyCharacterCreated missing branches ---

func TestApplyInviteCreated_InvalidStatus(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	payload := map[string]any{"invite_id": "inv-1", "participant_id": "part-1", "status": "INVALID"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteCreated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid invite status")
	}
}

func TestApplyInviteCreated_InvalidJSON(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteCreated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyInviteUpdated missing branches ---

func TestApplyInviteUpdated_InvalidJSON(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteUpdated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyInviteUpdated_InvalidStatus(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore()}
	payload := map[string]any{"invite_id": "inv-1", "status": "INVALID"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteUpdated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid invite status")
	}
}

// --- applySessionStarted missing branches ---
