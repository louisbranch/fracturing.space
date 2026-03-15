package charactermutationtransport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testCampaignStore struct {
	record storage.CampaignRecord
	err    error
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	if s.err != nil {
		return storage.CampaignRecord{}, s.err
	}
	return s.record, nil
}

type testDaggerheartStore struct {
	profiles map[string]projectionstore.DaggerheartCharacterProfile
	getErr   error
}

func (s *testDaggerheartStore) GetDaggerheartCharacterProfile(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
	if s.getErr != nil {
		return projectionstore.DaggerheartCharacterProfile{}, s.getErr
	}
	for _, profile := range s.profiles {
		return profile, nil
	}
	return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
}

func testProfile(campaignID, characterID string) projectionstore.DaggerheartCharacterProfile {
	return projectionstore.DaggerheartCharacterProfile{
		CampaignID:   campaignID,
		CharacterID:  characterID,
		Level:        1,
		GoldHandfuls: 1,
		GoldBags:     2,
		GoldChests:   3,
	}
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = &testDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"camp-1:char-1": testProfile("camp-1", "char-1"),
			},
		}
	}
	return NewHandler(deps)
}

func testContext() context.Context {
	ctx := grpcmeta.WithRequestID(context.Background(), "req-1")
	ctx = grpcmeta.WithInvocationID(ctx, "inv-1")
	return metadata.NewIncomingContext(ctx, metadata.Pairs(grpcmeta.SessionIDHeader, "sess-1"))
}

func TestHandlerRequireDependencies(t *testing.T) {
	tests := []struct {
		name string
		deps Dependencies
	}{
		{name: "missing campaign", deps: Dependencies{}},
		{name: "missing daggerheart", deps: Dependencies{Campaign: testCampaignStore{}}},
		{name: "missing executor", deps: Dependencies{Campaign: testCampaignStore{}, Daggerheart: &testDaggerheartStore{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewHandler(tt.deps).requireDependencies(); status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
		})
	}
}

func TestHandlerUpdateGoldRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.UpdateGold(testContext(), &pb.DaggerheartUpdateGoldRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerUpdateGoldSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"camp-1:char-1": testProfile("camp-1", "char-1"),
		},
	}
	var commandInput CharacterCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			commandInput = in
			store.profiles["camp-1:char-1"] = projectionstore.DaggerheartCharacterProfile{
				CampaignID:   "camp-1",
				CharacterID:  "char-1",
				Level:        1,
				GoldHandfuls: 4,
				GoldBags:     5,
				GoldChests:   6,
			}
			return nil
		},
	})

	resp, err := handler.UpdateGold(testContext(), &pb.DaggerheartUpdateGoldRequest{
		CampaignId:     "camp-1",
		CharacterId:    "char-1",
		HandfulsBefore: 1,
		HandfulsAfter:  4,
		BagsBefore:     2,
		BagsAfter:      5,
		ChestsBefore:   3,
		ChestsAfter:    6,
		Reason:         "loot",
	})
	if err != nil {
		t.Fatalf("UpdateGold returned error: %v", err)
	}
	if resp.Handfuls != 4 || resp.Bags != 5 || resp.Chests != 6 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if commandInput.CommandType != commandids.DaggerheartGoldUpdate {
		t.Fatalf("command type = %v, want %v", commandInput.CommandType, commandids.DaggerheartGoldUpdate)
	}
}

func TestHandlerAcquireDomainCardDefaultsDestination(t *testing.T) {
	var payload daggerheart.DomainCardAcquirePayload
	handler := newTestHandler(Dependencies{
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			if err := json.Unmarshal(in.PayloadJSON, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			return nil
		},
	})

	resp, err := handler.AcquireDomainCard(testContext(), &pb.DaggerheartAcquireDomainCardRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		CardId:      "card-1",
		CardLevel:   2,
	})
	if err != nil {
		t.Fatalf("AcquireDomainCard returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character id = %q, want %q", resp.CharacterId, "char-1")
	}
	if payload.Destination != "vault" {
		t.Fatalf("destination = %q, want vault", payload.Destination)
	}
}

func TestHandlerSwapEquipmentSuccess(t *testing.T) {
	var commandInput CharacterCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			commandInput = in
			return nil
		},
	})

	resp, err := handler.SwapEquipment(testContext(), &pb.DaggerheartSwapEquipmentRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		ItemId:      "item-1",
		ItemType:    "weapon",
		From:        "inventory",
		To:          "active",
		StressCost:  1,
	})
	if err != nil {
		t.Fatalf("SwapEquipment returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character id = %q, want %q", resp.CharacterId, "char-1")
	}
	if commandInput.CommandType != commandids.DaggerheartEquipmentSwap {
		t.Fatalf("command type = %v, want %v", commandInput.CommandType, commandids.DaggerheartEquipmentSwap)
	}
}

func TestHandlerUseConsumableSuccess(t *testing.T) {
	var commandInput CharacterCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			commandInput = in
			return nil
		},
	})

	resp, err := handler.UseConsumable(testContext(), &pb.DaggerheartUseConsumableRequest{
		CampaignId:     "camp-1",
		CharacterId:    "char-1",
		ConsumableId:   "cons-1",
		QuantityBefore: 2,
		QuantityAfter:  1,
	})
	if err != nil {
		t.Fatalf("UseConsumable returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character id = %q, want %q", resp.CharacterId, "char-1")
	}
	if commandInput.CommandType != commandids.DaggerheartConsumableUse {
		t.Fatalf("command type = %v, want %v", commandInput.CommandType, commandids.DaggerheartConsumableUse)
	}
}

func TestHandlerAcquireConsumableSuccess(t *testing.T) {
	var commandInput CharacterCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			commandInput = in
			return nil
		},
	})

	resp, err := handler.AcquireConsumable(testContext(), &pb.DaggerheartAcquireConsumableRequest{
		CampaignId:     "camp-1",
		CharacterId:    "char-1",
		ConsumableId:   "cons-1",
		QuantityBefore: 1,
		QuantityAfter:  3,
	})
	if err != nil {
		t.Fatalf("AcquireConsumable returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character id = %q, want %q", resp.CharacterId, "char-1")
	}
	if commandInput.CommandType != commandids.DaggerheartConsumableAcquire {
		t.Fatalf("command type = %v, want %v", commandInput.CommandType, commandids.DaggerheartConsumableAcquire)
	}
}

func TestHandlerApplyLevelUpRejectsInvalidIncrement(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error { return nil },
	})

	_, err := handler.ApplyLevelUp(testContext(), &pb.DaggerheartApplyLevelUpRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LevelAfter:  5,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerApplyLevelUpSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"camp-1:char-1": testProfile("camp-1", "char-1"),
		},
	}
	var commandInput CharacterCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			commandInput = in
			store.profiles["camp-1:char-1"] = projectionstore.DaggerheartCharacterProfile{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Level:       2,
			}
			return nil
		},
	})

	resp, err := handler.ApplyLevelUp(testContext(), &pb.DaggerheartApplyLevelUpRequest{
		CampaignId:      "camp-1",
		CharacterId:     "char-1",
		LevelAfter:      2,
		NewDomainCardId: "card-2",
		Advancements: []*pb.DaggerheartLevelUpAdvancement{
			{Type: "trait", Trait: "agility"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyLevelUp returned error: %v", err)
	}
	if resp.Level != 2 || resp.Tier != 2 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if commandInput.CommandType != commandids.DaggerheartLevelUpApply {
		t.Fatalf("command type = %v, want %v", commandInput.CommandType, commandids.DaggerheartLevelUpApply)
	}
}

func TestHandlerMapsProfileLookupErrors(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: &testDaggerheartStore{getErr: errors.New("boom")},
		ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.UpdateGold(testContext(), &pb.DaggerheartUpdateGoldRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
