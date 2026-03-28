package snapshottransport

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	daggerhearttestkit "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/testkit"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
)

func TestPatchCharacterState_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.PatchCharacterState(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_MissingCampaignId(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: daggerhearttestkit.NewFakeDaggerheartStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_MissingCharacterId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: daggerhearttestkit.NewFakeDaggerheartStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_InvalidHope(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := daggerhearttestkit.NewFakeDaggerheartStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
	})
	_, err := svc.PatchCharacterState(requestctx.WithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 15, Hope: 7, Stress: 1}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_InvalidStress(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := daggerhearttestkit.NewFakeDaggerheartStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	dhStore.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
	})
	_, err := svc.PatchCharacterState(requestctx.WithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 15, Hope: 3, Stress: 10}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_InvalidHp(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := daggerhearttestkit.NewFakeDaggerheartStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	dhStore.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
	})
	_, err := svc.PatchCharacterState(requestctx.WithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 25, Hope: 3, Stress: 1}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
