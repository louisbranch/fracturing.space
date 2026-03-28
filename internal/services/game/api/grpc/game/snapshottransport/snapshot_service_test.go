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
	"google.golang.org/grpc/codes"
)

func TestPatchCharacterState_CampaignNotFound(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: daggerhearttestkit.NewFakeDaggerheartStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:  "nonexistent",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterState_RequiresCharacterMutationPolicy(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: daggerhearttestkit.NewFakeDaggerheartStore(),
	})

	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 10, Hope: 3, Stress: 1},
		},
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestPatchCharacterState_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: daggerhearttestkit.NewFakeDaggerheartStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestPatchCharacterState_StateNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: daggerhearttestkit.NewFakeDaggerheartStore(),
	})
	_, err := svc.PatchCharacterState(requestctx.WithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "nonexistent",
	})
	assertStatusCode(t, err, codes.NotFound)
}
