package game

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestGetSnapshot_NilRequest(t *testing.T) {
	svc := NewSnapshotService(Stores{})
	_, err := svc.GetSnapshot(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSnapshot_MissingCampaignId(t *testing.T) {
	svc := NewSnapshotService(Stores{
		Campaign:    newFakeCampaignStore(),
		Daggerheart: newFakeDaggerheartStore(),
		Character:   newFakeCharacterStore(),
	})
	_, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSnapshot_CampaignNotFound(t *testing.T) {
	svc := NewSnapshotService(Stores{
		Campaign:    newFakeCampaignStore(),
		Daggerheart: newFakeDaggerheartStore(),
		Character:   newFakeCharacterStore(),
	})
	_, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetSnapshot_CampaignArchivedAllowed(t *testing.T) {
	// GetSnapshot uses CampaignOpRead which is allowed for all campaign statuses,
	// including archived campaigns. This allows viewing historical campaign state.
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	characterStore := newFakeCharacterStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusArchived,
	}
	dhStore.snapshots["c1"] = storage.DaggerheartSnapshot{CampaignID: "c1", GMFear: 5}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("GetSnapshot response has nil snapshot")
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 5 {
		t.Errorf("Snapshot GmFear = %d, want %d", dh.GetGmFear(), 5)
	}
}

func TestGetSnapshot_Success_NoCharacters(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	characterStore := newFakeCharacterStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	dhStore.snapshots["c1"] = storage.DaggerheartSnapshot{CampaignID: "c1", GMFear: 5}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("GetSnapshot response has nil snapshot")
	}
	if resp.Snapshot.CampaignId != "c1" {
		t.Errorf("Snapshot CampaignId = %q, want %q", resp.Snapshot.CampaignId, "c1")
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 5 {
		t.Errorf("Snapshot GmFear = %d, want %d", dh.GetGmFear(), 5)
	}
	if len(resp.Snapshot.CharacterStates) != 0 {
		t.Errorf("Snapshot CharacterStates = %d, want 0", len(resp.Snapshot.CharacterStates))
	}
}

func TestGetSnapshot_Success_WithCharacters(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	characterStore := newFakeCharacterStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	dhStore.snapshots["c1"] = storage.DaggerheartSnapshot{CampaignID: "c1", GMFear: 3}
	dhStore.states["c1"] = map[string]storage.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
		"ch2": {CampaignID: "c1", CharacterID: "ch2", Hp: 12, Hope: 2, Stress: 0},
	}
	characterStore.characters["c1"] = map[string]character.Character{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
		"ch2": {ID: "ch2", CampaignID: "c1", Name: "Sidekick", CreatedAt: now},
	}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 3 {
		t.Errorf("Snapshot GmFear = %d, want %d", dh.GetGmFear(), 3)
	}
	if len(resp.Snapshot.CharacterStates) != 2 {
		t.Errorf("Snapshot CharacterStates = %d, want 2", len(resp.Snapshot.CharacterStates))
	}
}

func TestGetSnapshot_Success_DefaultGmFear(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	characterStore := newFakeCharacterStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	// No DaggerheartSnapshot entry - should default to 0

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 0 {
		t.Errorf("Snapshot GmFear = %d, want 0 (default)", dh.GetGmFear())
	}
}

func TestPatchCharacterState_NilRequest(t *testing.T) {
	svc := NewSnapshotService(Stores{})
	_, err := svc.PatchCharacterState(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_MissingCampaignId(t *testing.T) {
	svc := NewSnapshotService(Stores{
		Campaign:    newFakeCampaignStore(),
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_MissingCharacterId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_CampaignNotFound(t *testing.T) {
	svc := NewSnapshotService(Stores{
		Campaign:    newFakeCampaignStore(),
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:  "nonexistent",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterState_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusArchived}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestPatchCharacterState_StateNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "nonexistent",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterState_InvalidHope(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	dhStore.states["c1"] = map[string]storage.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Event:       eventStore,
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 15, Hope: 7, Stress: 1}}, // Hope max is 6
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_InvalidStress(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	dhStore.states["c1"] = map[string]storage.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Event:       eventStore,
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 15, Hope: 3, Stress: 10}}, // Stress max is 6
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_InvalidHp(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	dhStore.states["c1"] = map[string]storage.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Event:       eventStore,
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 25, Hope: 3, Stress: 1}}, // Hp max is 18
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	dhStore.states["c1"] = map[string]storage.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Event:       eventStore,
	})

	resp, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 10, Hope: 5, Stress: 3}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterState returned error: %v", err)
	}
	if resp.State == nil {
		t.Fatal("PatchCharacterState response has nil state")
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetHope() != 5 {
		t.Errorf("State Hope = %d, want %d", dh.GetHope(), 5)
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetStress() != 3 {
		t.Errorf("State Stress = %d, want %d", dh.GetStress(), 3)
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetHp() != 10 {
		t.Errorf("State Hp = %d, want %d", dh.GetHp(), 10)
	}

	// Verify persisted
	dhStored, _ := dhStore.GetDaggerheartCharacterState(context.Background(), "c1", "ch1")
	if dhStored.Hope != 5 {
		t.Errorf("Stored Hope = %d, want %d", dhStored.Hope, 5)
	}
	if dhStored.Hp != 10 {
		t.Errorf("Stored Hp = %d, want %d", dhStored.Hp, 10)
	}

	if len(eventStore.events["c1"]) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventStore.events["c1"]))
	}
	if eventStore.events["c1"][0].Type != daggerheart.EventTypeCharacterStatePatched {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, daggerheart.EventTypeCharacterStatePatched)
	}
}

func TestPatchCharacterState_SetToZero(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	dhStore.states["c1"] = map[string]storage.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 5, Stress: 3},
	}
	dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Event:       eventStore,
	})

	resp, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 0, Hope: 0, Stress: 0}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterState returned error: %v", err)
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetHope() != 0 {
		t.Errorf("State Hope = %d, want 0", dh.GetHope())
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetHp() != 0 {
		t.Errorf("State Hp = %d, want 0", dh.GetHp())
	}

	// Verify persisted
	dhStored, _ := dhStore.GetDaggerheartCharacterState(context.Background(), "c1", "ch1")
	if dhStored.Hope != 0 {
		t.Errorf("Stored Hope = %d, want 0", dhStored.Hope)
	}
	if dhStored.Hp != 0 {
		t.Errorf("Stored Hp = %d, want 0", dhStored.Hp)
	}
}

func TestUpdateSnapshotState_NilRequest(t *testing.T) {
	svc := NewSnapshotService(Stores{})
	_, err := svc.UpdateSnapshotState(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateSnapshotState_MissingCampaignId(t *testing.T) {
	svc := NewSnapshotService(Stores{
		Campaign:    newFakeCampaignStore(),
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateSnapshotState_CampaignNotFound(t *testing.T) {
	svc := NewSnapshotService(Stores{
		Campaign:    newFakeCampaignStore(),
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "nonexistent",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestUpdateSnapshotState_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusArchived}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestUpdateSnapshotState_NegativeGmFear(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: newFakeDaggerheartStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: -1},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateSnapshotState_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Event:       eventStore,
	})

	resp, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 7},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 7 {
		t.Errorf("Response GmFear = %d, want %d", dh.GetGmFear(), 7)
	}

	// Verify persisted
	stored, err := dhStore.GetDaggerheartSnapshot(context.Background(), "c1")
	if err != nil {
		t.Fatalf("DaggerheartSnapshot not persisted: %v", err)
	}
	if stored.GMFear != 7 {
		t.Errorf("Stored GMFear = %d, want %d", stored.GMFear, 7)
	}

	if len(eventStore.events["c1"]) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventStore.events["c1"]))
	}
	if eventStore.events["c1"][0].Type != daggerheart.EventTypeGMFearChanged {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, daggerheart.EventTypeGMFearChanged)
	}
}

func TestUpdateSnapshotState_UpdateExisting(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	dhStore.snapshots["c1"] = storage.DaggerheartSnapshot{CampaignID: "c1", GMFear: 3}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Event:       eventStore,
	})

	resp, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 10},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 10 {
		t.Errorf("Response GmFear = %d, want %d", dh.GetGmFear(), 10)
	}

	// Verify updated
	stored, _ := dhStore.GetDaggerheartSnapshot(context.Background(), "c1")
	if stored.GMFear != 10 {
		t.Errorf("Stored GMFear = %d, want %d", stored.GMFear, 10)
	}
}

func TestUpdateSnapshotState_SetToZero(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	dhStore.snapshots["c1"] = storage.DaggerheartSnapshot{CampaignID: "c1", GMFear: 5}

	svc := NewSnapshotService(Stores{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Event:       eventStore,
	})

	resp, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 0},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 0 {
		t.Errorf("Response GmFear = %d, want 0", dh.GetGmFear())
	}
}
