package generator

import (
	"context"
	"fmt"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
)

// createCharacters creates the specified number of characters for a campaign.
// Characters are assigned as PCs to participants when available, with remaining as NPCs.
func (g *Generator) createCharacters(ctx context.Context, campaignID string, count int, participants []*statev1.Participant) ([]*statev1.Character, error) {
	if count < 1 {
		return nil, nil
	}

	characters := make([]*statev1.Character, 0, count)

	// Count players (non-GM participants) who can control PCs
	var playerParticipants []*statev1.Participant
	for _, p := range participants {
		if p.Role == statev1.ParticipantRole_PLAYER {
			playerParticipants = append(playerParticipants, p)
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

		// Assign PC to corresponding player
		if i < len(playerParticipants) {
			player := playerParticipants[i]
			_, err := g.characters.SetDefaultControl(ctx, &statev1.SetDefaultControlRequest{
				CampaignId:  campaignID,
				CharacterId: character.Id,
				Controller: &statev1.CharacterController{
					Controller: &statev1.CharacterController_Participant{
						Participant: &statev1.ParticipantController{
							ParticipantId: player.Id,
						},
					},
				},
			})
			if err != nil {
				return nil, fmt.Errorf("SetDefaultControl for character %s: %w", character.Id, err)
			}
		}
	}

	return characters, nil
}
