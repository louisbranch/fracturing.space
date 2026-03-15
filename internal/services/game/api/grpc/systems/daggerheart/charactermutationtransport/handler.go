package charactermutationtransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the Daggerheart character progression and inventory transport
// endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a character mutation transport handler from explicit
// campaign/profile reads and a character-command callback.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) UpdateGold(ctx context.Context, in *pb.DaggerheartUpdateGoldRequest) (*pb.DaggerheartUpdateGoldResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update gold request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "gold update"); err != nil {
		return nil, err
	}

	payloadJSON, err := json.Marshal(daggerheart.GoldUpdatePayload{
		CharacterID:    ids.CharacterID(characterID),
		HandfulsBefore: int(in.GetHandfulsBefore()),
		HandfulsAfter:  int(in.GetHandfulsAfter()),
		BagsBefore:     int(in.GetBagsBefore()),
		BagsAfter:      int(in.GetBagsAfter()),
		ChestsBefore:   int(in.GetChestsBefore()),
		ChestsAfter:    int(in.GetChestsAfter()),
		Reason:         strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartGoldUpdate,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "gold update did not emit an event",
		ApplyErrMessage: "apply gold update event",
	}); err != nil {
		return nil, err
	}

	updatedProfile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart profile", err)
	}
	return &pb.DaggerheartUpdateGoldResponse{
		CharacterId: characterID,
		Handfuls:    int32(updatedProfile.GoldHandfuls),
		Bags:        int32(updatedProfile.GoldBags),
		Chests:      int32(updatedProfile.GoldChests),
	}, nil
}

func (h *Handler) AcquireDomainCard(ctx context.Context, in *pb.DaggerheartAcquireDomainCardRequest) (*pb.DaggerheartAcquireDomainCardResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "acquire domain card request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	cardID, err := validate.RequiredID(in.GetCardId(), "card id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "domain card acquire"); err != nil {
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
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartDomainCardAcquire,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "domain card acquire did not emit an event",
		ApplyErrMessage: "apply domain card acquire event",
	}); err != nil {
		return nil, err
	}

	return &pb.DaggerheartAcquireDomainCardResponse{CharacterId: characterID}, nil
}

func (h *Handler) SwapEquipment(ctx context.Context, in *pb.DaggerheartSwapEquipmentRequest) (*pb.DaggerheartSwapEquipmentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "swap equipment request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	itemID, err := validate.RequiredID(in.GetItemId(), "item id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "equipment swap"); err != nil {
		return nil, err
	}

	payloadJSON, err := json.Marshal(daggerheart.EquipmentSwapPayload{
		CharacterID: ids.CharacterID(characterID),
		ItemID:      itemID,
		ItemType:    strings.TrimSpace(in.GetItemType()),
		From:        strings.TrimSpace(in.GetFrom()),
		To:          strings.TrimSpace(in.GetTo()),
		StressCost:  int(in.GetStressCost()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartEquipmentSwap,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "equipment swap did not emit an event",
		ApplyErrMessage: "apply equipment swap event",
	}); err != nil {
		return nil, err
	}

	return &pb.DaggerheartSwapEquipmentResponse{CharacterId: characterID}, nil
}

func (h *Handler) UseConsumable(ctx context.Context, in *pb.DaggerheartUseConsumableRequest) (*pb.DaggerheartUseConsumableResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "use consumable request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	consumableID, err := validate.RequiredID(in.GetConsumableId(), "consumable id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "consumable use"); err != nil {
		return nil, err
	}

	payloadJSON, err := json.Marshal(daggerheart.ConsumableUsePayload{
		CharacterID:    ids.CharacterID(characterID),
		ConsumableID:   consumableID,
		QuantityBefore: int(in.GetQuantityBefore()),
		QuantityAfter:  int(in.GetQuantityAfter()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartConsumableUse,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "consumable use did not emit an event",
		ApplyErrMessage: "apply consumable use event",
	}); err != nil {
		return nil, err
	}

	return &pb.DaggerheartUseConsumableResponse{CharacterId: characterID}, nil
}

func (h *Handler) AcquireConsumable(ctx context.Context, in *pb.DaggerheartAcquireConsumableRequest) (*pb.DaggerheartAcquireConsumableResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "acquire consumable request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	consumableID, err := validate.RequiredID(in.GetConsumableId(), "consumable id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "consumable acquire"); err != nil {
		return nil, err
	}

	payloadJSON, err := json.Marshal(daggerheart.ConsumableAcquirePayload{
		CharacterID:    ids.CharacterID(characterID),
		ConsumableID:   consumableID,
		QuantityBefore: int(in.GetQuantityBefore()),
		QuantityAfter:  int(in.GetQuantityAfter()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartConsumableAcquire,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "consumable acquire did not emit an event",
		ApplyErrMessage: "apply consumable acquire event",
	}); err != nil {
		return nil, err
	}

	return &pb.DaggerheartAcquireConsumableResponse{CharacterId: characterID}, nil
}

func (h *Handler) ApplyLevelUp(ctx context.Context, in *pb.DaggerheartApplyLevelUpRequest) (*pb.DaggerheartApplyLevelUpResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply level up request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return nil, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}

	profile, err := h.validateLevelUpPreconditions(ctx, campaignID, characterID)
	if err != nil {
		return nil, err
	}

	levelBefore := profile.Level
	if levelBefore == 0 {
		levelBefore = 1
	}
	levelAfter := int(in.GetLevelAfter())
	if levelAfter <= 0 {
		return nil, status.Error(codes.InvalidArgument, "level_after must be positive")
	}
	if levelAfter != levelBefore+1 {
		return nil, status.Error(codes.InvalidArgument, "level_after must be exactly one level higher than current level")
	}

	advancements := make([]daggerheart.LevelUpAdvancementPayload, 0, len(in.GetAdvancements()))
	for _, adv := range in.GetAdvancements() {
		entry := daggerheart.LevelUpAdvancementPayload{
			Type:            strings.TrimSpace(adv.GetType()),
			Trait:           strings.TrimSpace(adv.GetTrait()),
			DomainCardID:    strings.TrimSpace(adv.GetDomainCardId()),
			DomainCardLevel: int(adv.GetDomainCardLevel()),
			SubclassCardID:  strings.TrimSpace(adv.GetSubclassCardId()),
		}
		if adv.GetMulticlass() != nil {
			entry.Multiclass = &daggerheart.LevelUpMulticlassPayload{
				SecondaryClassID:    strings.TrimSpace(adv.GetMulticlass().GetSecondaryClassId()),
				SecondarySubclassID: strings.TrimSpace(adv.GetMulticlass().GetSecondarySubclassId()),
				FoundationCardID:    strings.TrimSpace(adv.GetMulticlass().GetFoundationCardId()),
				SpellcastTrait:      strings.TrimSpace(adv.GetMulticlass().GetSpellcastTrait()),
				DomainID:            strings.TrimSpace(adv.GetMulticlass().GetDomainId()),
			}
		}
		advancements = append(advancements, entry)
	}

	payloadJSON, err := json.Marshal(daggerheart.LevelUpApplyPayload{
		CharacterID:        ids.CharacterID(characterID),
		LevelBefore:        levelBefore,
		LevelAfter:         levelAfter,
		Advancements:       advancements,
		NewDomainCardID:    strings.TrimSpace(in.GetNewDomainCardId()),
		NewDomainCardLevel: int(in.GetNewDomainCardLevel()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartLevelUpApply,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "level up did not emit an event",
		ApplyErrMessage: "apply level up event",
	}); err != nil {
		return nil, err
	}

	updatedProfile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart profile", err)
	}
	return &pb.DaggerheartApplyLevelUpResponse{
		CharacterId: characterID,
		Level:       int32(updatedProfile.Level),
		Tier:        int32(tierForLevel(updatedProfile.Level)),
	}, nil
}

func (h *Handler) requireDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.ExecuteCharacterCommand == nil:
		return status.Error(codes.Internal, "character command executor is not configured")
	default:
		return nil
	}
}

func (h *Handler) validateCharacterPreconditions(ctx context.Context, campaignID, characterID, operationName string) (projectionstore.DaggerheartCharacterProfile, error) {
	if err := h.requireDependencies(); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, handleDomainError(err)
	}
	if err := requireDaggerheartSystemf(record, "campaign system does not support daggerheart %s", operationName); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, err
	}
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, handleDomainError(err)
	}
	return profile, nil
}

func (h *Handler) validateLevelUpPreconditions(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(record, "campaign system does not support daggerheart level up"); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, err
	}
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, handleDomainError(err)
	}
	return profile, nil
}

func (h *Handler) executeCharacterCommand(ctx context.Context, in CharacterCommandInput) error {
	if err := h.requireDependencies(); err != nil {
		return err
	}
	return h.deps.ExecuteCharacterCommand(ctx, in)
}
