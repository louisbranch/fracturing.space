package charactermutationtransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerSupportHelpers(t *testing.T) {
	t.Parallel()

	t.Run("validate character preconditions rejects non-daggerheart campaigns", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(Dependencies{
			Campaign: testCampaignStore{record: storage.CampaignRecord{
				ID:     "camp-1",
				System: systembridge.SystemID("blades"),
				Status: campaign.StatusActive,
			}},
			ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error { return nil },
		})

		_, err := handler.validateCharacterPreconditions(context.Background(), "camp-1", "char-1", "consumable use")
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want FailedPrecondition", status.Code(err))
		}
	})

	t.Run("validate level up preconditions rejects non-daggerheart campaigns", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(Dependencies{
			Campaign: testCampaignStore{record: storage.CampaignRecord{
				ID:     "camp-1",
				System: systembridge.SystemID("blades"),
				Status: campaign.StatusActive,
			}},
			ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error { return nil },
		})

		_, err := handler.validateLevelUpPreconditions(context.Background(), "camp-1", "char-1")
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want FailedPrecondition", status.Code(err))
		}
	})

	t.Run("execute character command surfaces dependency and executor errors", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(Dependencies{})
		if err := handler.executeCharacterCommand(context.Background(), CharacterCommandInput{}); status.Code(err) != codes.Internal {
			t.Fatalf("status code = %v, want Internal", status.Code(err))
		}

		boom := errors.New("boom")
		handler = newTestHandler(Dependencies{
			ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error { return boom },
		})
		if err := handler.executeCharacterCommand(context.Background(), CharacterCommandInput{}); !errors.Is(err, boom) {
			t.Fatalf("executeCharacterCommand() error = %v, want boom", err)
		}
	})
}

func TestInventoryAndProgressionHandlersPropagateExecutorFailures(t *testing.T) {
	t.Parallel()

	boom := status.Error(codes.Internal, "boom")
	handler := newTestHandler(Dependencies{
		Daggerheart: &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {
					CampaignID:      "camp-1",
					CharacterID:     "char-1",
					Level:           1,
					EquippedArmorID: "armor.leather",
				},
			},
			stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
				"char-1": {{CharacterID: "char-1"}},
			},
		},
		Content: contentStoreStub{
			armors: map[string]contentstore.DaggerheartArmor{
				"armor.leather": {ID: "armor.leather"},
				"armor.chain":   {ID: "armor.chain"},
			},
		},
		ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error { return boom },
	})

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "use consumable",
			run: func() error {
				_, err := handler.UseConsumable(testContext(), &pb.DaggerheartUseConsumableRequest{
					CampaignId:   "camp-1",
					CharacterId:  "char-1",
					ConsumableId: "cons-1",
				})
				return err
			},
		},
		{
			name: "acquire consumable",
			run: func() error {
				_, err := handler.AcquireConsumable(testContext(), &pb.DaggerheartAcquireConsumableRequest{
					CampaignId:   "camp-1",
					CharacterId:  "char-1",
					ConsumableId: "cons-1",
				})
				return err
			},
		},
		{
			name: "acquire domain card",
			run: func() error {
				_, err := handler.AcquireDomainCard(testContext(), &pb.DaggerheartAcquireDomainCardRequest{
					CampaignId:  "camp-1",
					CharacterId: "char-1",
					CardId:      "card-1",
				})
				return err
			},
		},
		{
			name: "swap equipment",
			run: func() error {
				_, err := handler.SwapEquipment(testContext(), &pb.DaggerheartSwapEquipmentRequest{
					CampaignId:  "camp-1",
					CharacterId: "char-1",
					ItemId:      "armor.chain",
					ItemType:    "armor",
					To:          "active",
				})
				return err
			},
		},
		{
			name: "update gold",
			run: func() error {
				_, err := handler.UpdateGold(testContext(), &pb.DaggerheartUpdateGoldRequest{
					CampaignId:  "camp-1",
					CharacterId: "char-1",
				})
				return err
			},
		},
		{
			name: "apply level up",
			run: func() error {
				_, err := handler.ApplyLevelUp(testContext(), &pb.DaggerheartApplyLevelUpRequest{
					CampaignId:  "camp-1",
					CharacterId: "char-1",
					LevelAfter:  2,
					Advancements: []*pb.DaggerheartLevelUpAdvancement{
						{Type: "trait", Trait: "agility"},
					},
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.run()
			if status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want Internal", status.Code(err))
			}
		})
	}
}
