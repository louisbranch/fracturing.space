package snapshottransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestGetSnapshot_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.GetSnapshot(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSnapshot_MissingCampaignId(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: gametest.NewFakeDaggerheartStore(),
		Character:   gametest.NewFakeCharacterStore(),
	})
	_, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSnapshot_CampaignNotFound(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: gametest.NewFakeDaggerheartStore(),
		Character:   gametest.NewFakeCharacterStore(),
	})
	_, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetSnapshot_RequiresCampaignReadPolicy(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
		Character:   gametest.NewFakeCharacterStore(),
	})

	_, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetSnapshot_CampaignArchivedAllowed(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	characterStore := gametest.NewFakeCharacterStore()

	campaignStore.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")
	dhStore.Snapshots["c1"] = projectionstore.DaggerheartSnapshot{CampaignID: "c1", GMFear: 5}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.GetSnapshotRequest{CampaignId: "c1"})
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
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	characterStore := gametest.NewFakeCharacterStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.Snapshots["c1"] = projectionstore.DaggerheartSnapshot{CampaignID: "c1", GMFear: 5}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.GetSnapshotRequest{CampaignId: "c1"})
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
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	characterStore := gametest.NewFakeCharacterStore()
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.Snapshots["c1"] = projectionstore.DaggerheartSnapshot{CampaignID: "c1", GMFear: 3}
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
		"ch2": {CampaignID: "c1", CharacterID: "ch2", Hp: 12, Hope: 2, Stress: 0},
	}
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, CreatedAt: now, UpdatedAt: now},
		"ch2": {ID: "ch2", CampaignID: "c1", Name: "Sidekick", Kind: character.KindPC, CreatedAt: now, UpdatedAt: now},
	}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.GetSnapshotRequest{CampaignId: "c1"})
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
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	characterStore := gametest.NewFakeCharacterStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 0 {
		t.Errorf("Snapshot GmFear = %d, want 0 (default)", dh.GetGmFear())
	}
}
