package projection

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection/testevent"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestApplyCampaignUpdated_StatusAndName(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", Status: campaign.StatusDraft, Name: "Old"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{
		Fields: map[string]any{
			"status": "ACTIVE",
			"name":   "  New Name  ",
		},
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := store.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if updated.Status != campaign.StatusActive {
		t.Fatalf("Status = %v, want %v", updated.Status, campaign.StatusActive)
	}
	if updated.Name != "New Name" {
		t.Fatalf("Name = %q, want %q", updated.Name, "New Name")
	}
	if !updated.UpdatedAt.Equal(stamp) {
		t.Fatalf("UpdatedAt = %v, want %v", updated.UpdatedAt, stamp)
	}
}

func TestApplySessionStarted_UsesEntityID(t *testing.T) {
	ctx := context.Background()
	sessionStore := &fakeSessionStore{}
	applier := Applier{Session: sessionStore, SessionInteraction: newFakeSessionInteractionStore()}

	payload := testevent.SessionStartedPayload{SessionID: "", SessionName: "Session 1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 14, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "sess-1", Type: testevent.TypeSessionStarted, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if sessionStore.last.ID != "sess-1" {
		t.Fatalf("Session ID = %q, want %q", sessionStore.last.ID, "sess-1")
	}
	if sessionStore.last.Status != session.StatusActive {
		t.Fatalf("Status = %v, want %v", sessionStore.last.Status, session.StatusActive)
	}
}

func TestApplySessionEnded(t *testing.T) {
	ctx := context.Background()
	sessionStore := &fakeSessionStore{}
	applier := Applier{Session: sessionStore, SessionInteraction: newFakeSessionInteractionStore()}

	payload := testevent.SessionEndedPayload{SessionID: "sess-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 14, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "sess-1", Type: testevent.TypeSessionEnded, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

func TestApplySessionEnded_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionEndedPayload{SessionID: "sess-1"})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeSessionEnded, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session store")
	}
}

func TestApplySessionEnded_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionEndedPayload{SessionID: "sess-1"})
	evt := testevent.Event{CampaignID: "", Type: testevent.TypeSessionEnded, PayloadJSON: data}
	applier := Applier{Session: &fakeSessionStore{}, SessionInteraction: newFakeSessionInteractionStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestParseCampaignStatus(t *testing.T) {
	tests := []struct {
		input string
		want  campaign.Status
		err   bool
	}{
		{"draft", campaign.StatusDraft, false},
		{"ACTIVE", campaign.StatusActive, false},
		{"completed", campaign.StatusCompleted, false},
		{"ARCHIVED", campaign.StatusArchived, false},
		{"CAMPAIGN_STATUS_DRAFT", campaign.StatusDraft, false},
		{"CAMPAIGN_STATUS_ACTIVE", campaign.StatusActive, false},
		{"CAMPAIGN_STATUS_COMPLETED", campaign.StatusCompleted, false},
		{"CAMPAIGN_STATUS_ARCHIVED", campaign.StatusArchived, false},
		{"", campaign.StatusUnspecified, true},
		{"   ", campaign.StatusUnspecified, true},
		{"unknown", campaign.StatusUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseCampaignStatus(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseCampaignStatus(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseCampaignStatus(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseGmMode(t *testing.T) {
	tests := []struct {
		input string
		want  campaign.GmMode
		err   bool
	}{
		{"human", campaign.GmModeHuman, false},
		{"AI", campaign.GmModeAI, false},
		{"hybrid", campaign.GmModeHybrid, false},
		{"GM_MODE_HUMAN", campaign.GmModeHuman, false},
		{"GM_MODE_AI", campaign.GmModeAI, false},
		{"GM_MODE_HYBRID", campaign.GmModeHybrid, false},
		{"", campaign.GmModeUnspecified, true},
		{"unknown", campaign.GmModeUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseGmMode(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseGmMode(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseGmMode(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseCampaignAccess(t *testing.T) {
	tests := []struct {
		input string
		want  participant.CampaignAccess
		err   bool
	}{
		{"member", participant.CampaignAccessMember, false},
		{"MANAGER", participant.CampaignAccessManager, false},
		{"owner", participant.CampaignAccessOwner, false},
		{"CAMPAIGN_ACCESS_MEMBER", participant.CampaignAccessMember, false},
		{"CAMPAIGN_ACCESS_MANAGER", participant.CampaignAccessManager, false},
		{"CAMPAIGN_ACCESS_OWNER", participant.CampaignAccessOwner, false},
		{"", participant.CampaignAccessUnspecified, true},
		{"admin", participant.CampaignAccessUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseCampaignAccess(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseCampaignAccess(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseCampaignAccess(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestApplyCampaignForked(t *testing.T) {
	ctx := context.Background()
	forkStore := newFakeCampaignForkStore()
	applier := Applier{CampaignFork: forkStore}

	payload := testevent.CampaignForkedPayload{
		ParentCampaignID: "parent-1",
		ForkEventSeq:     42,
		OriginCampaignID: "origin-1",
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignForked, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	m, err := forkStore.GetCampaignForkMetadata(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if m.ParentCampaignID != "parent-1" {
		t.Fatalf("ParentCampaignID = %q, want %q", m.ParentCampaignID, "parent-1")
	}
	if m.ForkEventSeq != 42 {
		t.Fatalf("ForkEventSeq = %d, want 42", m.ForkEventSeq)
	}
	if m.OriginCampaignID != "origin-1" {
		t.Fatalf("OriginCampaignID = %q, want %q", m.OriginCampaignID, "origin-1")
	}
}

func TestApplyCampaignForked_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignForkedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignForked, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing fork store")
	}
}

func TestApplyCampaignForked_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignForkedPayload{})
	evt := testevent.Event{CampaignID: "", Type: testevent.TypeCampaignForked, PayloadJSON: data}
	applier := Applier{CampaignFork: newFakeCampaignForkStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

// --- applyParticipantUpdated tests ---

func TestApplyCampaignCreated(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	applier := Applier{Campaign: store}

	payload := testevent.CampaignCreatedPayload{
		Name:        "Test Campaign",
		GameSystem:  "GAME_SYSTEM_DAGGERHEART",
		GmMode:      "human",
		ThemePrompt: "dark forest",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	evt := testevent.Event{EntityID: "camp-1", Type: testevent.TypeCampaignCreated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	c, err := store.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if c.Name != "Test Campaign" {
		t.Fatalf("Name = %q, want %q", c.Name, "Test Campaign")
	}
	if c.Status != campaign.StatusDraft {
		t.Fatalf("Status = %v, want Draft", c.Status)
	}
	if c.ThemePrompt != "dark forest" {
		t.Fatalf("ThemePrompt = %q", c.ThemePrompt)
	}
}

func TestApplyCampaignCreated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignCreatedPayload{Name: "X", GameSystem: "GAME_SYSTEM_DAGGERHEART", GmMode: "human"})
	evt := testevent.Event{EntityID: "camp-1", Type: testevent.TypeCampaignCreated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyCampaignCreated_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignCreatedPayload{Name: "X", GameSystem: "GAME_SYSTEM_DAGGERHEART", GmMode: "human"})
	evt := testevent.Event{EntityID: "", Type: testevent.TypeCampaignCreated, PayloadJSON: data}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

func TestApplyCampaignCreated_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	evt := testevent.Event{EntityID: "camp-1", Type: testevent.TypeCampaignCreated, PayloadJSON: []byte("{")}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyCampaignCreated_InvalidGameSystem(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignCreatedPayload{Name: "X", GameSystem: "", GmMode: "human"})
	evt := testevent.Event{EntityID: "camp-1", Type: testevent.TypeCampaignCreated, PayloadJSON: data}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid game system")
	}
}

func TestApplyCampaignCreated_InvalidGmMode(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignCreatedPayload{Name: "X", GameSystem: "DAGGERHEART", GmMode: ""})
	evt := testevent.Event{EntityID: "camp-1", Type: testevent.TypeCampaignCreated, PayloadJSON: data}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid gm mode")
	}
}

// --- applyCampaignUpdated additional tests ---

func TestApplyCampaignUpdated_ThemePrompt(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", Status: campaign.StatusDraft, ThemePrompt: "old"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"theme_prompt": "  new theme  "}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data, Timestamp: time.Now()}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	c, _ := store.Get(ctx, "camp-1")
	if c.ThemePrompt != "new theme" {
		t.Fatalf("ThemePrompt = %q, want %q", c.ThemePrompt, "new theme")
	}
}

func TestApplyCampaignUpdated_Locale(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{
		ID:     "camp-1",
		Status: campaign.StatusDraft,
		Locale: "en-US",
	}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"locale": "pt-BR"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data, Timestamp: time.Now()}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	c, _ := store.Get(ctx, "camp-1")
	if c.Locale != "pt-BR" {
		t.Fatalf("Locale = %v, want %v", c.Locale, "pt-BR")
	}
}

func TestApplyCampaignUpdated_EmptyFields(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

func TestApplyCampaignUpdated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignUpdatedPayload{Fields: map[string]any{"name": "X"}})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyCampaignUpdated_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignUpdatedPayload{Fields: map[string]any{"name": "X"}})
	evt := testevent.Event{CampaignID: "", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplyCampaignUpdated_InvalidNameType(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"name": 42}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid name type")
	}
}

func TestApplyCampaignUpdated_EmptyName(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"name": "  "}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestApplyCampaignUpdated_InvalidStatusType(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", Status: campaign.StatusDraft}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"status": 42}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid status type")
	}
}

func TestApplyCampaignUpdated_InvalidThemePromptType(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"theme_prompt": 42}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid theme_prompt type")
	}
}

func TestApplyCampaignUpdated_InvalidLocaleValue(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"locale": "es-ES"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid locale value")
	}
}

// --- applyParticipantJoined tests ---

func TestApplySessionStarted_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionStartedPayload{SessionID: "sess-1"})
	evt := testevent.Event{CampaignID: "", Type: testevent.TypeSessionStarted, PayloadJSON: data}
	applier := Applier{Session: &fakeSessionStore{}, SessionInteraction: newFakeSessionInteractionStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplySessionStarted_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionStartedPayload{SessionID: "sess-1"})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeSessionStarted, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session store")
	}
}

func TestApplySessionStarted_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionStartedPayload{SessionID: ""})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeSessionStarted, PayloadJSON: data}
	applier := Applier{Session: &fakeSessionStore{}, SessionInteraction: newFakeSessionInteractionStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

// --- applySessionEnded additional tests ---

func TestApplySessionEnded_EntityIDFallback(t *testing.T) {
	ctx := context.Background()
	sessionStore := &fakeSessionStore{}
	applier := Applier{Session: sessionStore, SessionInteraction: newFakeSessionInteractionStore()}

	payload := testevent.SessionEndedPayload{SessionID: ""}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "sess-fallback", Type: testevent.TypeSessionEnded, PayloadJSON: data, Timestamp: time.Now()}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

func TestApplySessionEnded_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionEndedPayload{SessionID: ""})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeSessionEnded, PayloadJSON: data}
	applier := Applier{Session: &fakeSessionStore{}, SessionInteraction: newFakeSessionInteractionStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

// --- applySessionGateAbandoned additional tests ---

func TestApplyCampaignForked_InvalidJSON(t *testing.T) {
	applier := Applier{CampaignFork: newFakeCampaignForkStore()}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignForked, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- marshalResolutionPayload missing branches ---

func TestApplySessionStarted_InvalidJSON(t *testing.T) {
	applier := Applier{Session: &fakeSessionStore{}, SessionInteraction: newFakeSessionInteractionStore()}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeSessionStarted, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applySessionEnded missing branches ---

func TestApplySessionEnded_InvalidJSON(t *testing.T) {
	applier := Applier{Session: &fakeSessionStore{}, SessionInteraction: newFakeSessionInteractionStore()}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeSessionEnded, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyCampaignUpdated missing branches ---

func TestApplyCampaignUpdated_InvalidJSON(t *testing.T) {
	applier := Applier{Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyCampaignUpdated_InvalidStatus(t *testing.T) {
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: campaignStore}
	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"status": "INVALID"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid status value")
	}
}

// --- applyParticipantJoined missing branches ---
