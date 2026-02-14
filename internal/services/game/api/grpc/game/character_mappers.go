package game

import (
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Character proto conversion helpers.
func characterToProto(ch character.Character) *campaignv1.Character {
	pb := &campaignv1.Character{
		Id:         ch.ID,
		CampaignId: ch.CampaignID,
		Name:       ch.Name,
		Kind:       characterKindToProto(ch.Kind),
		Notes:      ch.Notes,
		CreatedAt:  timestamppb.New(ch.CreatedAt),
		UpdatedAt:  timestamppb.New(ch.UpdatedAt),
	}
	if strings.TrimSpace(ch.ParticipantID) != "" {
		pb.ParticipantId = wrapperspb.String(ch.ParticipantID)
	}
	return pb
}

func characterKindFromProto(kind campaignv1.CharacterKind) character.CharacterKind {
	switch kind {
	case campaignv1.CharacterKind_PC:
		return character.CharacterKindPC
	case campaignv1.CharacterKind_NPC:
		return character.CharacterKindNPC
	default:
		return character.CharacterKindUnspecified
	}
}

func characterKindToProto(kind character.CharacterKind) campaignv1.CharacterKind {
	switch kind {
	case character.CharacterKindPC:
		return campaignv1.CharacterKind_PC
	case character.CharacterKindNPC:
		return campaignv1.CharacterKind_NPC
	default:
		return campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}
