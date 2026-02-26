package game

import (
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Character proto conversion helpers.
func characterToProto(ch storage.CharacterRecord) *campaignv1.Character {
	pb := &campaignv1.Character{
		Id:            ch.ID,
		CampaignId:    ch.CampaignID,
		Name:          ch.Name,
		Kind:          characterKindToProto(ch.Kind),
		Notes:         ch.Notes,
		AvatarSetId:   ch.AvatarSetID,
		AvatarAssetId: ch.AvatarAssetID,
		Pronouns:      ch.Pronouns,
		Aliases:       append([]string(nil), ch.Aliases...),
		CreatedAt:     timestamppb.New(ch.CreatedAt),
		UpdatedAt:     timestamppb.New(ch.UpdatedAt),
	}
	if strings.TrimSpace(ch.ParticipantID) != "" {
		pb.ParticipantId = wrapperspb.String(ch.ParticipantID)
	}
	return pb
}

func characterKindFromProto(kind campaignv1.CharacterKind) character.Kind {
	switch kind {
	case campaignv1.CharacterKind_PC:
		return character.KindPC
	case campaignv1.CharacterKind_NPC:
		return character.KindNPC
	default:
		return character.KindUnspecified
	}
}

func characterKindToProto(kind character.Kind) campaignv1.CharacterKind {
	switch kind {
	case character.KindPC:
		return campaignv1.CharacterKind_PC
	case character.KindNPC:
		return campaignv1.CharacterKind_NPC
	default:
		return campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}
