package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestSceneLifecycle(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-scene", now)

	// Put a scene.
	rec := storage.SceneRecord{
		CampaignID:  "camp-scene",
		SceneID:     "sc-1",
		SessionID:   "sess-1",
		Name:        "Battle",
		Description: "A fierce battle",
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.PutScene(ctx, rec); err != nil {
		t.Fatalf("put scene: %v", err)
	}

	// Get the scene.
	got, err := store.GetScene(ctx, "camp-scene", "sc-1")
	if err != nil {
		t.Fatalf("get scene: %v", err)
	}
	if got.Name != "Battle" {
		t.Errorf("name = %q, want %q", got.Name, "Battle")
	}
	if got.Description != "A fierce battle" {
		t.Errorf("description = %q, want %q", got.Description, "A fierce battle")
	}
	if !got.Active {
		t.Error("expected active")
	}
	if got.SessionID != "sess-1" {
		t.Errorf("session_id = %q, want %q", got.SessionID, "sess-1")
	}

	// Update via PutScene (upsert).
	rec.Name = "Updated Battle"
	rec.UpdatedAt = now.Add(time.Hour)
	if err := store.PutScene(ctx, rec); err != nil {
		t.Fatalf("put scene update: %v", err)
	}
	got, _ = store.GetScene(ctx, "camp-scene", "sc-1")
	if got.Name != "Updated Battle" {
		t.Errorf("name after update = %q, want %q", got.Name, "Updated Battle")
	}

	// End the scene.
	endedAt := now.Add(2 * time.Hour)
	if err := store.EndScene(ctx, "camp-scene", "sc-1", endedAt); err != nil {
		t.Fatalf("end scene: %v", err)
	}
	got, _ = store.GetScene(ctx, "camp-scene", "sc-1")
	if got.Active {
		t.Error("expected inactive after end")
	}
	if got.EndedAt == nil {
		t.Fatal("expected ended_at")
	}
	if !got.EndedAt.Equal(endedAt.UTC()) {
		t.Errorf("ended_at = %v, want %v", got.EndedAt, endedAt.UTC())
	}
}

func TestSceneGetNotFound(t *testing.T) {
	store := openTestStore(t)
	_, err := store.GetScene(context.Background(), "camp-x", "nonexistent")
	if err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSceneEndNotFound(t *testing.T) {
	store := openTestStore(t)
	err := store.EndScene(context.Background(), "camp-x", "nonexistent", time.Now())
	if err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListScenes(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-list-scenes", now)

	// Seed 3 scenes in the same session.
	for i, name := range []string{"sc-1", "sc-2", "sc-3"} {
		if err := store.PutScene(ctx, storage.SceneRecord{
			CampaignID: "camp-list-scenes",
			SceneID:    name,
			SessionID:  "sess-1",
			Name:       "Scene " + name,
			Active:     true,
			CreatedAt:  now.Add(time.Duration(i) * time.Minute),
			UpdatedAt:  now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("put scene %s: %v", name, err)
		}
	}

	// Seed one in a different session.
	if err := store.PutScene(ctx, storage.SceneRecord{
		CampaignID: "camp-list-scenes",
		SceneID:    "sc-other",
		SessionID:  "sess-2",
		Name:       "Other",
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("put other scene: %v", err)
	}

	// List all in sess-1.
	page, err := store.ListScenes(ctx, "camp-list-scenes", "sess-1", 10, "")
	if err != nil {
		t.Fatalf("list scenes: %v", err)
	}
	if len(page.Scenes) != 3 {
		t.Fatalf("scene count = %d, want 3", len(page.Scenes))
	}

	// Pagination.
	page1, err := store.ListScenes(ctx, "camp-list-scenes", "sess-1", 2, "")
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(page1.Scenes) != 2 {
		t.Fatalf("page 1 count = %d, want 2", len(page1.Scenes))
	}
	if page1.NextPageToken == "" {
		t.Fatal("expected next page token")
	}
	page2, err := store.ListScenes(ctx, "camp-list-scenes", "sess-1", 2, page1.NextPageToken)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(page2.Scenes) != 1 {
		t.Fatalf("page 2 count = %d, want 1", len(page2.Scenes))
	}

	// List from a different session — should only get 1.
	otherPage, err := store.ListScenes(ctx, "camp-list-scenes", "sess-2", 10, "")
	if err != nil {
		t.Fatalf("list other session: %v", err)
	}
	if len(otherPage.Scenes) != 1 {
		t.Fatalf("other session count = %d, want 1", len(otherPage.Scenes))
	}
}

func TestListActiveScenes(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-active", now)

	for _, name := range []string{"sc-a", "sc-b"} {
		if err := store.PutScene(ctx, storage.SceneRecord{
			CampaignID: "camp-active", SceneID: name, SessionID: "sess-1",
			Name: name, Active: true, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("put scene: %v", err)
		}
	}
	if err := store.EndScene(ctx, "camp-active", "sc-b", now.Add(time.Hour)); err != nil {
		t.Fatalf("end scene: %v", err)
	}

	active, err := store.ListActiveScenes(ctx, "camp-active")
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(active) != 1 || active[0].SceneID != "sc-a" {
		t.Fatalf("active scenes = %v, want [sc-a]", active)
	}
}

func TestSceneCharacterLifecycle(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-sc-char", now)

	// Add characters to a scene.
	if err := store.PutSceneCharacter(ctx, storage.SceneCharacterRecord{
		CampaignID: "camp-sc-char", SceneID: "sc-1", CharacterID: "char-1", AddedAt: now,
	}); err != nil {
		t.Fatalf("put scene character: %v", err)
	}
	if err := store.PutSceneCharacter(ctx, storage.SceneCharacterRecord{
		CampaignID: "camp-sc-char", SceneID: "sc-1", CharacterID: "char-2", AddedAt: now,
	}); err != nil {
		t.Fatalf("put scene character 2: %v", err)
	}

	// List characters.
	chars, err := store.ListSceneCharacters(ctx, "camp-sc-char", "sc-1")
	if err != nil {
		t.Fatalf("list scene characters: %v", err)
	}
	if len(chars) != 2 {
		t.Fatalf("char count = %d, want 2", len(chars))
	}

	// Remove one.
	if err := store.DeleteSceneCharacter(ctx, "camp-sc-char", "sc-1", "char-1"); err != nil {
		t.Fatalf("delete scene character: %v", err)
	}
	chars, _ = store.ListSceneCharacters(ctx, "camp-sc-char", "sc-1")
	if len(chars) != 1 {
		t.Fatalf("char count after delete = %d, want 1", len(chars))
	}
	if chars[0].CharacterID != "char-2" {
		t.Errorf("remaining char = %q, want %q", chars[0].CharacterID, "char-2")
	}

	// List for empty scene.
	empty, err := store.ListSceneCharacters(ctx, "camp-sc-char", "nonexistent")
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty, got %d", len(empty))
	}
}

func TestSceneGateLifecycle(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-sc-gate", now)

	// Put a gate.
	gate := storage.SceneGate{
		CampaignID:         "camp-sc-gate",
		SceneID:            "sc-1",
		GateID:             "gate-1",
		GateType:           "gm_consequence",
		Status:             session.GateStatusOpen,
		Reason:             "test",
		CreatedAt:          now,
		CreatedByActorType: "system",
		MetadataJSON:       []byte(`{"key":"value"}`),
	}
	if err := store.PutSceneGate(ctx, gate); err != nil {
		t.Fatalf("put scene gate: %v", err)
	}

	// Get the gate.
	got, err := store.GetSceneGate(ctx, "camp-sc-gate", "sc-1", "gate-1")
	if err != nil {
		t.Fatalf("get scene gate: %v", err)
	}
	if got.GateType != "gm_consequence" {
		t.Errorf("gate_type = %q, want %q", got.GateType, "gm_consequence")
	}
	if got.Status != session.GateStatusOpen {
		t.Errorf("status = %q, want %q", got.Status, session.GateStatusOpen)
	}

	// Get open gate.
	open, err := store.GetOpenSceneGate(ctx, "camp-sc-gate", "sc-1")
	if err != nil {
		t.Fatalf("get open gate: %v", err)
	}
	if open.GateID != "gate-1" {
		t.Errorf("open gate_id = %q, want %q", open.GateID, "gate-1")
	}

	// Resolve the gate.
	resolvedAt := now.Add(time.Hour)
	gate.Status = session.GateStatusResolved
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = "system"
	gate.ResolutionJSON = []byte(`{"decision":"proceed"}`)
	if err := store.PutSceneGate(ctx, gate); err != nil {
		t.Fatalf("put resolved gate: %v", err)
	}

	// No open gate now.
	_, err = store.GetOpenSceneGate(ctx, "camp-sc-gate", "sc-1")
	if err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound for open gate, got %v", err)
	}

	// Get not-found gate.
	_, err = store.GetSceneGate(ctx, "camp-sc-gate", "sc-1", "nonexistent")
	if err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- Validation error paths ---

func TestPutScene_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		rec  storage.SceneRecord
	}{
		{"missing campaign_id", storage.SceneRecord{SceneID: "s1", SessionID: "sess-1", CreatedAt: now, UpdatedAt: now}},
		{"missing scene_id", storage.SceneRecord{CampaignID: "c1", SessionID: "sess-1", CreatedAt: now, UpdatedAt: now}},
		{"missing session_id", storage.SceneRecord{CampaignID: "c1", SceneID: "s1", CreatedAt: now, UpdatedAt: now}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.PutScene(ctx, tt.rec); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPutScene_ContextCancelled(t *testing.T) {
	store := openTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := store.PutScene(ctx, storage.SceneRecord{CampaignID: "c1", SceneID: "s1", SessionID: "sess-1"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestPutSceneCharacter_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		rec  storage.SceneCharacterRecord
	}{
		{"missing campaign_id", storage.SceneCharacterRecord{SceneID: "s1", CharacterID: "c1", AddedAt: now}},
		{"missing scene_id", storage.SceneCharacterRecord{CampaignID: "c1", CharacterID: "c1", AddedAt: now}},
		{"missing character_id", storage.SceneCharacterRecord{CampaignID: "c1", SceneID: "s1", AddedAt: now}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.PutSceneCharacter(ctx, tt.rec); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDeleteSceneCharacter_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		campaignID  string
		sceneID     string
		characterID string
	}{
		{"missing campaign_id", "", "s1", "c1"},
		{"missing scene_id", "c1", "", "c1"},
		{"missing character_id", "c1", "s1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.DeleteSceneCharacter(ctx, tt.campaignID, tt.sceneID, tt.characterID); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPutSceneGate_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	base := storage.SceneGate{
		CampaignID: "c1", SceneID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen, CreatedAt: now,
	}
	tests := []struct {
		name string
		gate storage.SceneGate
	}{
		{"missing campaign_id", func() storage.SceneGate { g := base; g.CampaignID = ""; return g }()},
		{"missing scene_id", func() storage.SceneGate { g := base; g.SceneID = ""; return g }()},
		{"missing gate_id", func() storage.SceneGate { g := base; g.GateID = ""; return g }()},
		{"missing gate_type", func() storage.SceneGate { g := base; g.GateType = ""; return g }()},
		{"missing status", func() storage.SceneGate { g := base; g.Status = ""; return g }()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.PutSceneGate(ctx, tt.gate); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPutSceneSpotlight_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	base := storage.SceneSpotlight{
		CampaignID: "c1", SceneID: "s1", SpotlightType: "gm", UpdatedAt: now,
	}
	tests := []struct {
		name      string
		spotlight storage.SceneSpotlight
	}{
		{"missing campaign_id", func() storage.SceneSpotlight { s := base; s.CampaignID = ""; return s }()},
		{"missing scene_id", func() storage.SceneSpotlight { s := base; s.SceneID = ""; return s }()},
		{"missing spotlight_type", func() storage.SceneSpotlight { s := base; s.SpotlightType = ""; return s }()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.PutSceneSpotlight(ctx, tt.spotlight); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestEndScene_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		campaignID string
		sceneID    string
	}{
		{"missing campaign_id", "", "s1"},
		{"missing scene_id", "c1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.EndScene(ctx, tt.campaignID, tt.sceneID, time.Now()); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestGetScene_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		campaignID string
		sceneID    string
	}{
		{"missing campaign_id", "", "s1"},
		{"missing scene_id", "c1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := store.GetScene(ctx, tt.campaignID, tt.sceneID); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestListScenes_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		campaignID string
		sessionID  string
		pageSize   int
	}{
		{"missing campaign_id", "", "s1", 10},
		{"missing session_id", "c1", "", 10},
		{"zero page_size", "c1", "s1", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := store.ListScenes(ctx, tt.campaignID, tt.sessionID, tt.pageSize, ""); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestListActiveScenes_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	_, err := store.ListActiveScenes(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty campaign_id")
	}
}

func TestListSceneCharacters_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		campaignID string
		sceneID    string
	}{
		{"missing campaign_id", "", "s1"},
		{"missing scene_id", "c1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := store.ListSceneCharacters(ctx, tt.campaignID, tt.sceneID); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestGetSceneGate_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		campaignID string
		sceneID    string
		gateID     string
	}{
		{"missing campaign_id", "", "s1", "g1"},
		{"missing scene_id", "c1", "", "g1"},
		{"missing gate_id", "c1", "s1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := store.GetSceneGate(ctx, tt.campaignID, tt.sceneID, tt.gateID); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestGetOpenSceneGate_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		campaignID string
		sceneID    string
	}{
		{"missing campaign_id", "", "s1"},
		{"missing scene_id", "c1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := store.GetOpenSceneGate(ctx, tt.campaignID, tt.sceneID); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestGetSceneSpotlight_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		campaignID string
		sceneID    string
	}{
		{"missing campaign_id", "", "s1"},
		{"missing scene_id", "c1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := store.GetSceneSpotlight(ctx, tt.campaignID, tt.sceneID); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestClearSceneSpotlight_ValidationErrors(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		campaignID string
		sceneID    string
	}{
		{"missing campaign_id", "", "s1"},
		{"missing scene_id", "c1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.ClearSceneSpotlight(ctx, tt.campaignID, tt.sceneID); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestSceneSpotlightLifecycle(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-sc-spot", now)

	// Put spotlight.
	spotlight := storage.SceneSpotlight{
		CampaignID:         "camp-sc-spot",
		SceneID:            "sc-1",
		SpotlightType:      "character",
		CharacterID:        "char-1",
		UpdatedAt:          now,
		UpdatedByActorType: "system",
	}
	if err := store.PutSceneSpotlight(ctx, spotlight); err != nil {
		t.Fatalf("put scene spotlight: %v", err)
	}

	// Get spotlight.
	got, err := store.GetSceneSpotlight(ctx, "camp-sc-spot", "sc-1")
	if err != nil {
		t.Fatalf("get scene spotlight: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Errorf("character_id = %q, want %q", got.CharacterID, "char-1")
	}
	if string(got.SpotlightType) != "character" {
		t.Errorf("spotlight_type = %q, want %q", got.SpotlightType, "character")
	}

	// Clear spotlight.
	if err := store.ClearSceneSpotlight(ctx, "camp-sc-spot", "sc-1"); err != nil {
		t.Fatalf("clear scene spotlight: %v", err)
	}

	// Get after clear.
	_, err = store.GetSceneSpotlight(ctx, "camp-sc-spot", "sc-1")
	if err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound after clear, got %v", err)
	}

	// Get not-found spotlight.
	_, err = store.GetSceneSpotlight(ctx, "camp-sc-spot", "nonexistent")
	if err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
