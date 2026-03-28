package generator

import (
	"context"
	"fmt"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// createCharacters creates the specified number of characters for a campaign.
// Characters are assigned as PCs to participants when available, with remaining as NPCs.
// Ownership is assigned players first, then GM/fallback participants.
func (g *Generator) createCharacters(ctx context.Context, campaignID string, count int, participants []*statev1.Participant) ([]*statev1.Character, error) {
	if count < 1 {
		return nil, nil
	}

	characters := make([]*statev1.Character, 0, count)

	// Collect players and GM for controller assignment
	var playerParticipants []*statev1.Participant
	var gmParticipant *statev1.Participant
	var fallbackParticipant *statev1.Participant
	for _, p := range participants {
		if p == nil {
			continue
		}
		if fallbackParticipant == nil {
			fallbackParticipant = p
		}
		if p.Role == statev1.ParticipantRole_PLAYER {
			playerParticipants = append(playerParticipants, p)
			continue
		}
		if p.Role == statev1.ParticipantRole_GM && gmParticipant == nil {
			gmParticipant = p
		}
	}

	for i := 0; i < count; i++ {
		// First characters are PCs (one per player), rest are NPCs
		kind := statev1.CharacterKind_NPC
		var notes string
		if i < len(playerParticipants) {
			kind = statev1.CharacterKind_PC
		} else {
			notes = g.wb.NPCDescription()
		}

		resp, err := g.characters.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
			CampaignId: campaignID,
			Name:       g.wb.CharacterName(),
			Kind:       kind,
			Notes:      notes,
		})
		if err != nil {
			return nil, fmt.Errorf("CreateCharacter %d: %w", i+1, err)
		}

		character := resp.Character
		characters = append(characters, character)

		// Assign an owner (players first, then GM).
		var ownerParticipant *statev1.Participant
		if i < len(playerParticipants) {
			ownerParticipant = playerParticipants[i]
		} else if gmParticipant != nil {
			ownerParticipant = gmParticipant
		} else if fallbackParticipant != nil {
			ownerParticipant = fallbackParticipant
		}
		if ownerParticipant == nil {
			return nil, fmt.Errorf("no participants available to assign owner for character %s", character.Id)
		}
		updateResp, err := g.characters.UpdateCharacter(ctx, &statev1.UpdateCharacterRequest{
			CampaignId:         campaignID,
			CharacterId:        character.Id,
			OwnerParticipantId: wrapperspb.String(ownerParticipant.Id),
		})
		if err != nil {
			return nil, fmt.Errorf("UpdateCharacter(owner) for character %s: %w", character.Id, err)
		}
		if updateResp.GetCharacter() != nil {
			characters[len(characters)-1] = updateResp.GetCharacter()
		}
	}

	return characters, nil
}
