package snapshottransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
)

func TestPatchCharacterState_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()

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

	_, err := svc.PatchCharacterState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 10, Hope: 5, Stress: 1},
		},
	})
	assertStatusCode(t, err, codes.Internal)
}
