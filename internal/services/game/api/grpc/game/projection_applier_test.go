package game

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestStoresApplier_ApplyCampaignAndParticipant(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	createPayload := campaign.CreatePayload{
		Name:         "Test Campaign",
		Locale:       "en-US",
		GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:       "GM_MODE_HUMAN",
		Intent:       "STARTER",
		AccessPolicy: "PUBLIC",
		ThemePrompt:  "A dark fantasy adventure",
		CoverAssetID: "camp-cover-02",
	}
	createJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("encode create payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: createJSON,
	}); err != nil {
		t.Fatalf("apply campaign.created: %v", err)
	}

	joinPayload := participant.JoinPayload{
		ParticipantID:  "part-1",
		UserID:         "user-1",
		Name:           "GM",
		Role:           "GM",
		Controller:     "HUMAN",
		CampaignAccess: "OWNER",
	}
	joinJSON, err := json.Marshal(joinPayload)
	if err != nil {
		t.Fatalf("encode join payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.joined"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "part-1",
		PayloadJSON: joinJSON,
	}); err != nil {
		t.Fatalf("apply participant.joined: %v", err)
	}

	campaign, err := stores.Campaign.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaign.Name != "Test Campaign" {
		t.Fatalf("campaign name = %q, want %q", campaign.Name, "Test Campaign")
	}
	if campaign.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("campaign system = %s, want %s", campaign.System, commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	}
	if campaign.CoverAssetID != "camp-cover-02" {
		t.Fatalf("campaign cover asset id = %q, want %q", campaign.CoverAssetID, "camp-cover-02")
	}
	if campaign.ParticipantCount != 1 {
		t.Fatalf("campaign participant count = %d, want 1", campaign.ParticipantCount)
	}

	participant, err := stores.Participant.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if participant.Name != "GM" {
		t.Fatalf("participant display name = %q, want %q", participant.Name, "GM")
	}
}

func TestStoresApplier_ApplyParticipantUpdated(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	createPayload := campaign.CreatePayload{
		Name:         "Test Campaign",
		Locale:       "en-US",
		GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:       "GM_MODE_HUMAN",
		Intent:       "STARTER",
		AccessPolicy: "PUBLIC",
	}
	createJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("encode create payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: createJSON,
	}); err != nil {
		t.Fatalf("apply campaign.created: %v", err)
	}

	joinPayload := participant.JoinPayload{
		ParticipantID:  "part-1",
		UserID:         "user-1",
		Name:           "GM",
		Role:           "GM",
		Controller:     "HUMAN",
		CampaignAccess: "OWNER",
	}
	joinJSON, err := json.Marshal(joinPayload)
	if err != nil {
		t.Fatalf("encode join payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.joined"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "part-1",
		PayloadJSON: joinJSON,
	}); err != nil {
		t.Fatalf("apply participant.joined: %v", err)
	}

	updatePayload := participant.UpdatePayload{
		ParticipantID: "part-1",
		Fields: map[string]string{
			"name": "Guide",
		},
	}
	updateJSON, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("encode update payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.updated"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "part-1",
		PayloadJSON: updateJSON,
	}); err != nil {
		t.Fatalf("apply participant.updated: %v", err)
	}

	updated, err := stores.Participant.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if updated.Name != "Guide" {
		t.Fatalf("display name = %q, want %q", updated.Name, "Guide")
	}
}

func TestStoresApplier_ApplyParticipantLeft(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	createPayload := campaign.CreatePayload{
		Name:         "Test Campaign",
		Locale:       "en-US",
		GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:       "GM_MODE_HUMAN",
		Intent:       "STARTER",
		AccessPolicy: "PUBLIC",
	}
	createJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("encode create payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: createJSON,
	}); err != nil {
		t.Fatalf("apply campaign.created: %v", err)
	}

	joinPayload := participant.JoinPayload{
		ParticipantID:  "part-1",
		UserID:         "user-1",
		Name:           "GM",
		Role:           "GM",
		Controller:     "HUMAN",
		CampaignAccess: "OWNER",
	}
	joinJSON, err := json.Marshal(joinPayload)
	if err != nil {
		t.Fatalf("encode join payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.joined"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "part-1",
		PayloadJSON: joinJSON,
	}); err != nil {
		t.Fatalf("apply participant.joined: %v", err)
	}

	leavePayload := participant.LeavePayload{
		ParticipantID: "part-1",
		Reason:        "left",
	}
	leaveJSON, err := json.Marshal(leavePayload)
	if err != nil {
		t.Fatalf("encode leave payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.left"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "part-1",
		PayloadJSON: leaveJSON,
	}); err != nil {
		t.Fatalf("apply participant.left: %v", err)
	}

	if _, err := stores.Participant.GetParticipant(ctx, "camp-1", "part-1"); err == nil {
		t.Fatal("expected participant to be deleted")
	}
	updated, err := stores.Campaign.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if updated.ParticipantCount != 0 {
		t.Fatalf("participant count = %d, want 0", updated.ParticipantCount)
	}
}

func TestStoresApplier_ApplyParticipantBindUnbindAndSeatReassign(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
		ClaimIndex:  newFakeClaimIndexStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	createPayload := campaign.CreatePayload{
		Name:         "Test Campaign",
		Locale:       "en-US",
		GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:       "GM_MODE_HUMAN",
		Intent:       "STARTER",
		AccessPolicy: "PUBLIC",
	}
	createJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("encode create payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: createJSON,
	}); err != nil {
		t.Fatalf("apply campaign.created: %v", err)
	}

	joinPayload := participant.JoinPayload{
		ParticipantID:  "part-1",
		UserID:         "",
		Name:           "GM",
		Role:           "GM",
		Controller:     "HUMAN",
		CampaignAccess: "OWNER",
	}
	joinJSON, err := json.Marshal(joinPayload)
	if err != nil {
		t.Fatalf("encode join payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.joined"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "part-1",
		PayloadJSON: joinJSON,
	}); err != nil {
		t.Fatalf("apply participant.joined: %v", err)
	}

	bindPayload := participant.BindPayload{ParticipantID: "part-1", UserID: "user-1"}
	bindJSON, err := json.Marshal(bindPayload)
	if err != nil {
		t.Fatalf("encode bind payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.bound"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "part-1",
		PayloadJSON: bindJSON,
	}); err != nil {
		t.Fatalf("apply participant.bound: %v", err)
	}

	bound, err := stores.Participant.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if bound.UserID != "user-1" {
		t.Fatalf("user_id = %q, want %q", bound.UserID, "user-1")
	}

	unbindPayload := participant.UnbindPayload{ParticipantID: "part-1", UserID: "user-1"}
	unbindJSON, err := json.Marshal(unbindPayload)
	if err != nil {
		t.Fatalf("encode unbind payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.unbound"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "part-1",
		PayloadJSON: unbindJSON,
	}); err != nil {
		t.Fatalf("apply participant.unbound: %v", err)
	}

	unbound, err := stores.Participant.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if unbound.UserID != "" {
		t.Fatalf("user_id = %q, want empty", unbound.UserID)
	}

	reassignPayload := participant.SeatReassignPayload{ParticipantID: "part-1", PriorUserID: "", UserID: "user-2"}
	reassignJSON, err := json.Marshal(reassignPayload)
	if err != nil {
		t.Fatalf("encode reassign payload: %v", err)
	}
	seatEvents := []event.Type{
		event.Type("seat.reassigned"),
		event.Type("participant.seat_reassigned"),
	}
	for _, eventType := range seatEvents {
		t.Run(string(eventType), func(t *testing.T) {
			if err := applier.Apply(ctx, event.Event{
				CampaignID:  "camp-1",
				Type:        eventType,
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "part-1",
				PayloadJSON: reassignJSON,
			}); err != nil {
				t.Fatalf("apply %s: %v", eventType, err)
			}
		})
	}

	reassigned, err := stores.Participant.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if reassigned.UserID != "user-2" {
		t.Fatalf("user_id = %q, want %q", reassigned.UserID, "user-2")
	}
}

func TestStoresApplier_ApplyCampaignUpdatedAndSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		Campaign: newFakeCampaignStore(),
		Session:  newFakeSessionStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 1, 0, 0, 0, time.UTC)

	createPayload := campaign.CreatePayload{
		Name:         "Test Campaign",
		Locale:       "en-US",
		GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:       "GM_MODE_HUMAN",
		Intent:       "STARTER",
		AccessPolicy: "PUBLIC",
	}
	createJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("encode create payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: createJSON,
	}); err != nil {
		t.Fatalf("apply campaign.created: %v", err)
	}

	updatePayload := campaign.UpdatePayload{Fields: map[string]string{"status": "active"}}
	updateJSON, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("encode update payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.updated"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: updateJSON,
	}); err != nil {
		t.Fatalf("apply campaign.updated: %v", err)
	}

	campaignRecord, err := stores.Campaign.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaignRecord.Status != campaign.StatusActive {
		t.Fatalf("campaign status = %v, want %v", campaignRecord.Status, campaign.StatusActive)
	}

	startPayload := session.StartPayload{SessionID: "sess-1", SessionName: "Opening"}
	startJSON, err := json.Marshal(startPayload)
	if err != nil {
		t.Fatalf("encode start payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		SessionID:   "sess-1",
		Type:        event.Type("session.started"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "session",
		EntityID:    "sess-1",
		PayloadJSON: startJSON,
	}); err != nil {
		t.Fatalf("apply session.started: %v", err)
	}

	sess, err := stores.Session.GetSession(ctx, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess.Status != session.StatusActive {
		t.Fatalf("session status = %v, want %v", sess.Status, session.StatusActive)
	}
	if sess.Name != "Opening" {
		t.Fatalf("session name = %q, want %q", sess.Name, "Opening")
	}

	endPayload := session.EndPayload{SessionID: "sess-1"}
	endJSON, err := json.Marshal(endPayload)
	if err != nil {
		t.Fatalf("encode end payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		SessionID:   "sess-1",
		Type:        event.Type("session.ended"),
		Timestamp:   now.Add(2 * time.Hour),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "session",
		EntityID:    "sess-1",
		PayloadJSON: endJSON,
	}); err != nil {
		t.Fatalf("apply session.ended: %v", err)
	}

	ended, err := stores.Session.GetSession(ctx, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("get session after end: %v", err)
	}
	if ended.Status != session.StatusEnded {
		t.Fatalf("session status after end = %v, want %v", ended.Status, session.StatusEnded)
	}
	if ended.EndedAt == nil {
		t.Fatal("expected ended session to have EndedAt")
	}
}

func TestStoresApplier_ApplyCampaignUpdated_CoverAssetID(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		Campaign: newFakeCampaignStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 1, 0, 0, 0, time.UTC)

	createPayload := campaign.CreatePayload{
		Name:         "Test Campaign",
		Locale:       "en-US",
		GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:       "GM_MODE_HUMAN",
		Intent:       "STARTER",
		AccessPolicy: "PUBLIC",
		CoverAssetID: "camp-cover-01",
	}
	createJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("encode create payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: createJSON,
	}); err != nil {
		t.Fatalf("apply campaign.created: %v", err)
	}

	updatePayload := campaign.UpdatePayload{Fields: map[string]string{"cover_asset_id": "camp-cover-04"}}
	updateJSON, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("encode update payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.updated"),
		Timestamp:   now.Add(time.Minute),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: updateJSON,
	}); err != nil {
		t.Fatalf("apply campaign.updated: %v", err)
	}

	campaignRecord, err := stores.Campaign.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaignRecord.CoverAssetID != "camp-cover-04" {
		t.Fatalf("campaign cover asset id = %q, want %q", campaignRecord.CoverAssetID, "camp-cover-04")
	}
}

func TestStoresApplier_ApplySessionGateLifecycle(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		SessionGate: newFakeSessionGateStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 2, 0, 0, 0, time.UTC)

	openPayload := session.GateOpenedPayload{
		GateID:   "gate-1",
		GateType: "spotlight",
		Reason:   "test",
		Metadata: map[string]any{"key": "value"},
	}
	openJSON, err := json.Marshal(openPayload)
	if err != nil {
		t.Fatalf("encode gate open payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		SessionID:   "sess-1",
		Type:        event.Type("session.gate_opened"),
		Timestamp:   now,
		ActorType:   event.ActorTypeParticipant,
		ActorID:     "part-1",
		EntityType:  "session_gate",
		EntityID:    "gate-1",
		PayloadJSON: openJSON,
	}); err != nil {
		t.Fatalf("apply session.gate_opened: %v", err)
	}

	gate, err := stores.SessionGate.GetSessionGate(ctx, "camp-1", "sess-1", "gate-1")
	if err != nil {
		t.Fatalf("get session gate: %v", err)
	}
	if gate.Status != session.GateStatusOpen {
		t.Fatalf("gate status = %s, want %s", gate.Status, session.GateStatusOpen)
	}
	if gate.CreatedByActorID != "part-1" {
		t.Fatalf("gate created actor id = %q, want %q", gate.CreatedByActorID, "part-1")
	}

	resolvePayload := session.GateResolvedPayload{GateID: "gate-1", Decision: "allow"}
	resolveJSON, err := json.Marshal(resolvePayload)
	if err != nil {
		t.Fatalf("encode gate resolve payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		SessionID:   "sess-1",
		Type:        event.Type("session.gate_resolved"),
		Timestamp:   now.Add(10 * time.Minute),
		ActorType:   event.ActorTypeSystem,
		ActorID:     "system",
		EntityType:  "session_gate",
		EntityID:    "gate-1",
		PayloadJSON: resolveJSON,
	}); err != nil {
		t.Fatalf("apply session.gate_resolved: %v", err)
	}

	resolved, err := stores.SessionGate.GetSessionGate(ctx, "camp-1", "sess-1", "gate-1")
	if err != nil {
		t.Fatalf("get resolved gate: %v", err)
	}
	if resolved.Status != session.GateStatusResolved {
		t.Fatalf("resolved gate status = %s, want %s", resolved.Status, session.GateStatusResolved)
	}
	if resolved.ResolvedByActorID != "system" {
		t.Fatalf("resolved actor id = %q, want %q", resolved.ResolvedByActorID, "system")
	}
}

func TestStoresApplier_ApplySessionSpotlightSetAndClear(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		SessionSpotlight: newFakeSessionSpotlightStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 3, 0, 0, 0, time.UTC)

	setPayload := session.SpotlightSetPayload{SpotlightType: "character", CharacterID: "char-1"}
	setJSON, err := json.Marshal(setPayload)
	if err != nil {
		t.Fatalf("encode spotlight set payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		SessionID:   "sess-1",
		Type:        event.Type("session.spotlight_set"),
		Timestamp:   now,
		ActorType:   event.ActorTypeParticipant,
		ActorID:     "part-1",
		EntityType:  "session",
		EntityID:    "sess-1",
		PayloadJSON: setJSON,
	}); err != nil {
		t.Fatalf("apply session.spotlight_set: %v", err)
	}

	spotlight, err := stores.SessionSpotlight.GetSessionSpotlight(ctx, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("get session spotlight: %v", err)
	}
	if spotlight.CharacterID != "char-1" {
		t.Fatalf("spotlight character id = %q, want %q", spotlight.CharacterID, "char-1")
	}

	clearPayload := session.SpotlightClearedPayload{Reason: "scene shift"}
	clearJSON, err := json.Marshal(clearPayload)
	if err != nil {
		t.Fatalf("encode spotlight clear payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		SessionID:   "sess-1",
		Type:        event.Type("session.spotlight_cleared"),
		Timestamp:   now.Add(15 * time.Minute),
		ActorType:   event.ActorTypeSystem,
		ActorID:     "system",
		EntityType:  "session",
		EntityID:    "sess-1",
		PayloadJSON: clearJSON,
	}); err != nil {
		t.Fatalf("apply session.spotlight_cleared: %v", err)
	}

	if _, err := stores.SessionSpotlight.GetSessionSpotlight(ctx, "camp-1", "sess-1"); err == nil {
		t.Fatal("expected spotlight to be cleared")
	}
}

func TestStoresApplier_ApplyInviteLifecycle(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		Campaign: newFakeCampaignStore(),
		Invite:   newFakeInviteStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 4, 0, 0, 0, time.UTC)

	createPayload := campaign.CreatePayload{
		Name:         "Test Campaign",
		Locale:       "en-US",
		GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:       "GM_MODE_HUMAN",
		Intent:       "STARTER",
		AccessPolicy: "PUBLIC",
	}
	createJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("encode create payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: createJSON,
	}); err != nil {
		t.Fatalf("apply campaign.created: %v", err)
	}

	invitePayload := invite.CreatePayload{
		InviteID:               "invite-1",
		ParticipantID:          "part-1",
		RecipientUserID:        "user-1",
		CreatedByParticipantID: "part-1",
		Status:                 string(invite.StatusPending),
	}
	inviteJSON, err := json.Marshal(invitePayload)
	if err != nil {
		t.Fatalf("encode invite payload: %v", err)
	}
	inviteCreatedAt := now.Add(5 * time.Minute)
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("invite.created"),
		Timestamp:   inviteCreatedAt,
		ActorType:   event.ActorTypeParticipant,
		ActorID:     "part-1",
		EntityType:  "invite",
		EntityID:    "invite-1",
		PayloadJSON: inviteJSON,
	}); err != nil {
		t.Fatalf("apply invite.created: %v", err)
	}

	stored, err := stores.Invite.GetInvite(ctx, "invite-1")
	if err != nil {
		t.Fatalf("get invite: %v", err)
	}
	if stored.Status != invite.StatusPending {
		t.Fatalf("invite status = %v, want %v", stored.Status, invite.StatusPending)
	}
	campaignRecord, err := stores.Campaign.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if !campaignRecord.UpdatedAt.Equal(inviteCreatedAt) {
		t.Fatalf("campaign updated_at = %v, want %v", campaignRecord.UpdatedAt, inviteCreatedAt)
	}

	claimPayload := invite.ClaimPayload{
		InviteID:      "invite-1",
		ParticipantID: "part-1",
		UserID:        "user-1",
		JWTID:         "jwt-1",
	}
	claimJSON, err := json.Marshal(claimPayload)
	if err != nil {
		t.Fatalf("encode claim payload: %v", err)
	}
	claimedAt := now.Add(10 * time.Minute)
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("invite.claimed"),
		Timestamp:   claimedAt,
		ActorType:   event.ActorTypeSystem,
		ActorID:     "system",
		EntityType:  "invite",
		EntityID:    "invite-1",
		PayloadJSON: claimJSON,
	}); err != nil {
		t.Fatalf("apply invite.claimed: %v", err)
	}

	claimed, err := stores.Invite.GetInvite(ctx, "invite-1")
	if err != nil {
		t.Fatalf("get claimed invite: %v", err)
	}
	if claimed.Status != invite.StatusClaimed {
		t.Fatalf("claimed invite status = %v, want %v", claimed.Status, invite.StatusClaimed)
	}
	campaignRecord, err = stores.Campaign.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign after claim: %v", err)
	}
	if !campaignRecord.UpdatedAt.Equal(claimedAt) {
		t.Fatalf("campaign updated_at after claim = %v, want %v", campaignRecord.UpdatedAt, claimedAt)
	}

	revokePayload := invite.RevokePayload{InviteID: "invite-1"}
	revokeJSON, err := json.Marshal(revokePayload)
	if err != nil {
		t.Fatalf("encode revoke payload: %v", err)
	}
	revokedAt := now.Add(15 * time.Minute)
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("invite.revoked"),
		Timestamp:   revokedAt,
		ActorType:   event.ActorTypeSystem,
		ActorID:     "system",
		EntityType:  "invite",
		EntityID:    "invite-1",
		PayloadJSON: revokeJSON,
	}); err != nil {
		t.Fatalf("apply invite.revoked: %v", err)
	}

	revoked, err := stores.Invite.GetInvite(ctx, "invite-1")
	if err != nil {
		t.Fatalf("get revoked invite: %v", err)
	}
	if revoked.Status != invite.StatusRevoked {
		t.Fatalf("revoked invite status = %v, want %v", revoked.Status, invite.StatusRevoked)
	}
	campaignRecord, err = stores.Campaign.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign after revoke: %v", err)
	}
	if !campaignRecord.UpdatedAt.Equal(revokedAt) {
		t.Fatalf("campaign updated_at after revoke = %v, want %v", campaignRecord.UpdatedAt, revokedAt)
	}
}

func TestStoresApplier_ApplyInviteUpdated(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		Campaign: newFakeCampaignStore(),
		Invite:   newFakeInviteStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 5, 0, 0, 0, time.UTC)

	createPayload := campaign.CreatePayload{
		Name:         "Test Campaign",
		Locale:       "en-US",
		GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:       "GM_MODE_HUMAN",
		Intent:       "STARTER",
		AccessPolicy: "PUBLIC",
	}
	createJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("encode create payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.created"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: createJSON,
	}); err != nil {
		t.Fatalf("apply campaign.created: %v", err)
	}

	invitePayload := invite.CreatePayload{
		InviteID:               "invite-2",
		ParticipantID:          "part-1",
		RecipientUserID:        "user-1",
		CreatedByParticipantID: "part-1",
		Status:                 string(invite.StatusPending),
	}
	inviteJSON, err := json.Marshal(invitePayload)
	if err != nil {
		t.Fatalf("encode invite payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("invite.created"),
		Timestamp:   now.Add(1 * time.Minute),
		ActorType:   event.ActorTypeParticipant,
		ActorID:     "part-1",
		EntityType:  "invite",
		EntityID:    "invite-2",
		PayloadJSON: inviteJSON,
	}); err != nil {
		t.Fatalf("apply invite.created: %v", err)
	}

	updatePayload := invite.UpdatePayload{
		InviteID: "invite-2",
		Status:   string(invite.StatusClaimed),
	}
	updateJSON, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("encode update payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("invite.updated"),
		Timestamp:   now.Add(2 * time.Minute),
		ActorType:   event.ActorTypeSystem,
		ActorID:     "system",
		EntityType:  "invite",
		EntityID:    "invite-2",
		PayloadJSON: updateJSON,
	}); err != nil {
		t.Fatalf("apply invite.updated: %v", err)
	}

	updated, err := stores.Invite.GetInvite(ctx, "invite-2")
	if err != nil {
		t.Fatalf("get updated invite: %v", err)
	}
	if updated.Status != invite.StatusClaimed {
		t.Fatalf("updated invite status = %v, want %v", updated.Status, invite.StatusClaimed)
	}
}

func TestStoresApplier_ApplyCampaignForked(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		CampaignFork: newFakeCampaignForkStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 6, 0, 0, 0, time.UTC)

	forkPayload := campaign.ForkPayload{
		ParentCampaignID: "parent-1",
		ForkEventSeq:     42,
		OriginCampaignID: "origin-1",
		CopyParticipants: true,
	}
	forkJSON, err := json.Marshal(forkPayload)
	if err != nil {
		t.Fatalf("encode fork payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.forked"),
		Timestamp:   now,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-1",
		PayloadJSON: forkJSON,
	}); err != nil {
		t.Fatalf("apply campaign.forked: %v", err)
	}

	metadata, err := stores.CampaignFork.GetCampaignForkMetadata(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get fork metadata: %v", err)
	}
	if metadata.ParentCampaignID != "parent-1" {
		t.Fatalf("parent campaign id = %q, want %q", metadata.ParentCampaignID, "parent-1")
	}
	if metadata.ForkEventSeq != 42 {
		t.Fatalf("fork event seq = %d, want %d", metadata.ForkEventSeq, 42)
	}
	if metadata.OriginCampaignID != "origin-1" {
		t.Fatalf("origin campaign id = %q, want %q", metadata.OriginCampaignID, "origin-1")
	}
}

func TestStoresApplier_ApplyCharacterLifecycle(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		Campaign:  newFakeCampaignStore(),
		Character: newFakeCharacterStore(),
	}
	applier := stores.Applier()
	now := time.Date(2026, 2, 14, 6, 30, 0, 0, time.UTC)

	campaignRecord := storage.CampaignRecord{
		ID:        "camp-1",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := stores.Campaign.Put(ctx, campaignRecord); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	createPayload := character.CreatePayload{
		CharacterID: "char-1",
		Name:        "Hero",
		Kind:        "PC",
		Notes:       "Notes",
	}
	createJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("encode create payload: %v", err)
	}
	createdAt := now.Add(1 * time.Minute)
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("character.created"),
		Timestamp:   createdAt,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "character",
		EntityID:    "char-1",
		PayloadJSON: createJSON,
	}); err != nil {
		t.Fatalf("apply character.created: %v", err)
	}

	created, err := stores.Character.GetCharacter(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get created character: %v", err)
	}
	if created.Name != "Hero" {
		t.Fatalf("character name = %q, want %q", created.Name, "Hero")
	}

	updatePayload := character.UpdatePayload{
		CharacterID: "char-1",
		Fields: map[string]string{
			"name":           "Hero Updated",
			"kind":           "NPC",
			"notes":          "New Notes",
			"participant_id": "part-1",
		},
	}
	updateJSON, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("encode update payload: %v", err)
	}
	updatedAt := now.Add(2 * time.Minute)
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("character.updated"),
		Timestamp:   updatedAt,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "character",
		EntityID:    "char-1",
		PayloadJSON: updateJSON,
	}); err != nil {
		t.Fatalf("apply character.updated: %v", err)
	}

	updated, err := stores.Character.GetCharacter(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get updated character: %v", err)
	}
	if updated.Name != "Hero Updated" {
		t.Fatalf("updated name = %q, want %q", updated.Name, "Hero Updated")
	}
	if updated.Kind != character.KindNPC {
		t.Fatalf("updated kind = %v, want %v", updated.Kind, character.KindNPC)
	}
	if updated.ParticipantID != "part-1" {
		t.Fatalf("updated participant_id = %q, want %q", updated.ParticipantID, "part-1")
	}

	deletePayload := character.DeletePayload{CharacterID: "char-1", Reason: "gone"}
	deleteJSON, err := json.Marshal(deletePayload)
	if err != nil {
		t.Fatalf("encode delete payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("character.deleted"),
		Timestamp:   now.Add(3 * time.Minute),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "character",
		EntityID:    "char-1",
		PayloadJSON: deleteJSON,
	}); err != nil {
		t.Fatalf("apply character.deleted: %v", err)
	}

	if _, err := stores.Character.GetCharacter(ctx, "camp-1", "char-1"); err == nil {
		t.Fatal("expected deleted character to be removed")
	}
}

func TestStoresApplier_ApplyCharacterProfileUpdated(t *testing.T) {
	ctx := context.Background()
	stores := Stores{
		Daggerheart: newFakeDaggerheartStore(),
	}
	applier := stores.Applier()

	profilePayload := character.ProfileUpdatePayload{
		CharacterID: "char-1",
		SystemProfile: map[string]any{
			"daggerheart": map[string]any{
				"level":            1,
				"hp_max":           6,
				"stress_max":       6,
				"evasion":          10,
				"major_threshold":  5,
				"severe_threshold": 10,
				"proficiency":      1,
				"armor_score":      1,
				"armor_max":        2,
				"agility":          1,
				"strength":         0,
				"finesse":          1,
				"instinct":         0,
				"presence":         0,
				"knowledge":        1,
				"experiences": []map[string]any{
					{"name": "Scout", "modifier": 2},
				},
			},
		},
	}
	profileJSON, err := json.Marshal(profilePayload)
	if err != nil {
		t.Fatalf("encode profile payload: %v", err)
	}
	if err := applier.Apply(ctx, event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("character.profile_updated"),
		Timestamp:   time.Date(2026, 2, 14, 7, 0, 0, 0, time.UTC),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "character",
		EntityID:    "char-1",
		PayloadJSON: profileJSON,
	}); err != nil {
		t.Fatalf("apply character.profile_updated: %v", err)
	}

	stored, err := stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get daggerheart profile: %v", err)
	}
	if stored.Level != 1 {
		t.Fatalf("profile level = %d, want %d", stored.Level, 1)
	}
}

type fakeClaimIndexStore struct {
	claims map[string]storage.ParticipantClaim
}

func newFakeClaimIndexStore() *fakeClaimIndexStore {
	return &fakeClaimIndexStore{claims: make(map[string]storage.ParticipantClaim)}
}

func (f *fakeClaimIndexStore) PutParticipantClaim(_ context.Context, campaignID, userID, participantID string, claimedAt time.Time) error {
	key := campaignID + ":" + userID
	f.claims[key] = storage.ParticipantClaim{
		CampaignID:    campaignID,
		UserID:        userID,
		ParticipantID: participantID,
		ClaimedAt:     claimedAt,
	}
	return nil
}

func (f *fakeClaimIndexStore) GetParticipantClaim(_ context.Context, campaignID, userID string) (storage.ParticipantClaim, error) {
	key := campaignID + ":" + userID
	claim, ok := f.claims[key]
	if !ok {
		return storage.ParticipantClaim{}, storage.ErrNotFound
	}
	return claim, nil
}

func (f *fakeClaimIndexStore) DeleteParticipantClaim(_ context.Context, campaignID, userID string) error {
	key := campaignID + ":" + userID
	if _, ok := f.claims[key]; !ok {
		return storage.ErrNotFound
	}
	delete(f.claims, key)
	return nil
}
