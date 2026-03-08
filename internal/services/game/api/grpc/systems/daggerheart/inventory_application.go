package daggerheart

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// validateInventoryPreconditions checks common preconditions for inventory commands.
func (s *DaggerheartService) validateInventoryPreconditions(ctx context.Context, campaignID, characterID, operationName string) error {
	if err := s.requireDependencies(dependencyCampaignStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return err
	}
	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return handleDomainError(err)
	}
	if err := requireDaggerheartSystemf(c, "campaign system does not support daggerheart %s", operationName); err != nil {
		return err
	}

	// Verify character exists.
	if _, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID); err != nil {
		return handleDomainError(err)
	}
	return nil
}

// executeDomainCommand marshals a payload, builds a daggerheart system command, and executes it.
func (s *DaggerheartService) executeDomainCommand(ctx context.Context, campaignID, characterID string, cmdType command.Type, payloadJSON []byte, missingEventMsg, applyErrMsg string) error {
	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err := s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    ids.CampaignID(campaignID),
		Type:          cmdType,
		ActorType:     command.ActorTypeSystem,
		SessionID:     ids.SessionID(strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))),
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics(missingEventMsg, applyErrMsg))
	return err
}

func (s *DaggerheartService) runUpdateGold(ctx context.Context, in *pb.DaggerheartUpdateGoldRequest) (*pb.DaggerheartUpdateGoldResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update gold request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	if err := s.validateInventoryPreconditions(ctx, campaignID, characterID, "gold update"); err != nil {
		return nil, err
	}

	payload := daggerheart.GoldUpdatePayload{
		CharacterID:    ids.CharacterID(characterID),
		HandfulsBefore: int(in.GetHandfulsBefore()),
		HandfulsAfter:  int(in.GetHandfulsAfter()),
		BagsBefore:     int(in.GetBagsBefore()),
		BagsAfter:      int(in.GetBagsAfter()),
		ChestsBefore:   int(in.GetChestsBefore()),
		ChestsAfter:    int(in.GetChestsAfter()),
		Reason:         strings.TrimSpace(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	if err := s.executeDomainCommand(ctx, campaignID, characterID,
		commandTypeDaggerheartGoldUpdate, payloadJSON,
		"gold update did not emit an event", "apply gold update event"); err != nil {
		return nil, err
	}

	updatedProfile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart profile: %v", err)
	}
	return &pb.DaggerheartUpdateGoldResponse{
		CharacterId: characterID,
		Handfuls:    int32(updatedProfile.GoldHandfuls),
		Bags:        int32(updatedProfile.GoldBags),
		Chests:      int32(updatedProfile.GoldChests),
	}, nil
}

func (s *DaggerheartService) runAcquireDomainCard(ctx context.Context, in *pb.DaggerheartAcquireDomainCardRequest) (*pb.DaggerheartAcquireDomainCardResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "acquire domain card request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	cardID := strings.TrimSpace(in.GetCardId())
	if cardID == "" {
		return nil, status.Error(codes.InvalidArgument, "card id is required")
	}
	if err := s.validateInventoryPreconditions(ctx, campaignID, characterID, "domain card acquire"); err != nil {
		return nil, err
	}

	payload := daggerheart.DomainCardAcquirePayload{
		CharacterID: ids.CharacterID(characterID),
		CardID:      cardID,
		CardLevel:   int(in.GetCardLevel()),
		Destination: strings.TrimSpace(in.GetDestination()),
	}
	if payload.Destination == "" {
		payload.Destination = "vault"
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	if err := s.executeDomainCommand(ctx, campaignID, characterID,
		commandTypeDaggerheartDomainCardAcquire, payloadJSON,
		"domain card acquire did not emit an event", "apply domain card acquire event"); err != nil {
		return nil, err
	}

	return &pb.DaggerheartAcquireDomainCardResponse{
		CharacterId: characterID,
	}, nil
}

func (s *DaggerheartService) runSwapEquipment(ctx context.Context, in *pb.DaggerheartSwapEquipmentRequest) (*pb.DaggerheartSwapEquipmentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "swap equipment request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	itemID := strings.TrimSpace(in.GetItemId())
	if itemID == "" {
		return nil, status.Error(codes.InvalidArgument, "item id is required")
	}
	if err := s.validateInventoryPreconditions(ctx, campaignID, characterID, "equipment swap"); err != nil {
		return nil, err
	}

	payload := daggerheart.EquipmentSwapPayload{
		CharacterID: ids.CharacterID(characterID),
		ItemID:      itemID,
		ItemType:    strings.TrimSpace(in.GetItemType()),
		From:        strings.TrimSpace(in.GetFrom()),
		To:          strings.TrimSpace(in.GetTo()),
		StressCost:  int(in.GetStressCost()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	if err := s.executeDomainCommand(ctx, campaignID, characterID,
		commandTypeDaggerheartEquipmentSwap, payloadJSON,
		"equipment swap did not emit an event", "apply equipment swap event"); err != nil {
		return nil, err
	}

	return &pb.DaggerheartSwapEquipmentResponse{
		CharacterId: characterID,
	}, nil
}

func (s *DaggerheartService) runUseConsumable(ctx context.Context, in *pb.DaggerheartUseConsumableRequest) (*pb.DaggerheartUseConsumableResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "use consumable request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	consumableID := strings.TrimSpace(in.GetConsumableId())
	if consumableID == "" {
		return nil, status.Error(codes.InvalidArgument, "consumable id is required")
	}
	if err := s.validateInventoryPreconditions(ctx, campaignID, characterID, "consumable use"); err != nil {
		return nil, err
	}

	payload := daggerheart.ConsumableUsePayload{
		CharacterID:    ids.CharacterID(characterID),
		ConsumableID:   consumableID,
		QuantityBefore: int(in.GetQuantityBefore()),
		QuantityAfter:  int(in.GetQuantityAfter()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	if err := s.executeDomainCommand(ctx, campaignID, characterID,
		commandTypeDaggerheartConsumableUse, payloadJSON,
		"consumable use did not emit an event", "apply consumable use event"); err != nil {
		return nil, err
	}

	return &pb.DaggerheartUseConsumableResponse{
		CharacterId: characterID,
	}, nil
}

func (s *DaggerheartService) runAcquireConsumable(ctx context.Context, in *pb.DaggerheartAcquireConsumableRequest) (*pb.DaggerheartAcquireConsumableResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "acquire consumable request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	consumableID := strings.TrimSpace(in.GetConsumableId())
	if consumableID == "" {
		return nil, status.Error(codes.InvalidArgument, "consumable id is required")
	}
	if err := s.validateInventoryPreconditions(ctx, campaignID, characterID, "consumable acquire"); err != nil {
		return nil, err
	}

	payload := daggerheart.ConsumableAcquirePayload{
		CharacterID:    ids.CharacterID(characterID),
		ConsumableID:   consumableID,
		QuantityBefore: int(in.GetQuantityBefore()),
		QuantityAfter:  int(in.GetQuantityAfter()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	if err := s.executeDomainCommand(ctx, campaignID, characterID,
		commandTypeDaggerheartConsumableAcquire, payloadJSON,
		"consumable acquire did not emit an event", "apply consumable acquire event"); err != nil {
		return nil, err
	}

	return &pb.DaggerheartAcquireConsumableResponse{
		CharacterId: characterID,
	}, nil
}
