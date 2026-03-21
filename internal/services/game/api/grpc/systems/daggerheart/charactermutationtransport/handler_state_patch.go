package charactermutationtransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyCharacterStatePatch applies a direct HP/Hope/Stress/Armor patch to a
// character and returns the post-write state values.
func (h *Handler) ApplyCharacterStatePatch(ctx context.Context, in *pb.DaggerheartApplyCharacterStatePatchRequest) (*pb.DaggerheartApplyCharacterStatePatchResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply character state patch request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "state patch"); err != nil {
		return nil, err
	}

	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}

	payload := daggerheart.CharacterStatePatchPayload{
		CharacterID: ids.CharacterID(characterID),
		Source:      strings.TrimSpace(in.GetSource()),
	}
	if in.Hp != nil {
		payload.HPBefore = intPtr(state.Hp)
		payload.HPAfter = intPtr(int(in.GetHp()))
	}
	if in.Hope != nil {
		payload.HopeBefore = intPtr(state.Hope)
		payload.HopeAfter = intPtr(int(in.GetHope()))
	}
	if in.Stress != nil {
		payload.StressBefore = intPtr(state.Stress)
		payload.StressAfter = intPtr(int(in.GetStress()))
	}
	if in.Armor != nil {
		payload.ArmorBefore = intPtr(state.Armor)
		payload.ArmorAfter = intPtr(int(in.GetArmor()))
	}

	if src := in.GetMutationSource(); src != nil {
		payload.MutationSource = &daggerheart.MutationSource{
			Type:        src.GetType().String(),
			Description: src.GetDescription(),
			SourceID:    src.GetSourceId(),
		}
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartCharacterStatePatch,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "character state patch did not emit an event",
		ApplyErrMessage: "apply character state patch event",
	}); err != nil {
		return nil, err
	}

	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load character state", err)
	}
	hp := int32(updated.Hp)
	hope := int32(updated.Hope)
	stress := int32(updated.Stress)
	armor := int32(updated.Armor)
	return &pb.DaggerheartApplyCharacterStatePatchResponse{
		CharacterId: characterID,
		Hp:          &hp,
		Hope:        &hope,
		Stress:      &stress,
		Armor:       &armor,
	}, nil
}
