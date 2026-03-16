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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SwapEquipment applies an equipment move through the shared character-command
// execution path.
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
	profile, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "equipment swap")
	if err != nil {
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
	if err := h.enrichArmorSwapPayload(ctx, campaignID, profile, &payload); err != nil {
		return nil, err
	}
	payloadJSON, err := json.Marshal(payload)
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

func (h *Handler) enrichArmorSwapPayload(ctx context.Context, campaignID string, profile projectionstore.DaggerheartCharacterProfile, payload *daggerheart.EquipmentSwapPayload) error {
	if payload == nil || payload.ItemType != "armor" {
		return nil
	}
	if h.deps.Content == nil {
		return status.Error(codes.Internal, "content store is not configured")
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, payload.CharacterID.String())
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}

	var currentArmor *contentstore.DaggerheartArmor
	if currentArmorID := strings.TrimSpace(profile.EquippedArmorID); currentArmorID != "" {
		armor, err := h.deps.Content.GetDaggerheartArmor(ctx, currentArmorID)
		if err != nil {
			return grpcerror.Internal("load equipped armor", err)
		}
		currentArmor = &armor
	}

	var nextArmor *contentstore.DaggerheartArmor
	if payload.To == "active" {
		armor, err := h.deps.Content.GetDaggerheartArmor(ctx, payload.ItemID)
		if err != nil {
			return grpcerror.Internal("load target armor", err)
		}
		nextArmor = &armor
	}

	base := daggerheart.RemoveArmorPassiveEffects(profile, currentArmor)
	nextProfile := daggerheart.ApplyArmorProfileEffects(profile.Level, base, nextArmor)
	nextArmorScore := nextProfile.ArmorScore
	nextArmorMax := nextProfile.ArmorMax
	nextEvasion := nextProfile.Evasion
	nextMajor := nextProfile.MajorThreshold
	nextSevere := nextProfile.SevereThreshold
	nextSpellcast := nextProfile.SpellcastRollBonus
	nextAgility := nextProfile.Agility
	nextStrength := nextProfile.Strength
	nextFinesse := nextProfile.Finesse
	nextInstinct := nextProfile.Instinct
	nextPresence := nextProfile.Presence
	nextKnowledge := nextProfile.Knowledge
	nextArmorTotal := daggerheart.RemapArmorCurrent(state, profile.ArmorMax, nextArmorMax)

	payload.EquippedArmorID = strings.TrimSpace(nextProfile.EquippedArmorID)
	payload.EvasionAfter = &nextEvasion
	payload.MajorThresholdAfter = &nextMajor
	payload.SevereThresholdAfter = &nextSevere
	payload.ArmorScoreAfter = &nextArmorScore
	payload.ArmorMaxAfter = &nextArmorMax
	payload.SpellcastRollBonusAfter = &nextSpellcast
	payload.AgilityAfter = &nextAgility
	payload.StrengthAfter = &nextStrength
	payload.FinesseAfter = &nextFinesse
	payload.InstinctAfter = &nextInstinct
	payload.PresenceAfter = &nextPresence
	payload.KnowledgeAfter = &nextKnowledge
	payload.ArmorAfter = &nextArmorTotal
	return nil
}
