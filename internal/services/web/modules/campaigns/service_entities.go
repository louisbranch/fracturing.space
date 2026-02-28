package campaigns

import (
	"context"
	"sort"
	"strings"
)

// sortByName sorts items by a name key with ID tiebreaker.
func sortByName[T any](items []T, nameOf func(T) string, idOf func(T) string) {
	sort.SliceStable(items, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(nameOf(items[i])))
		right := strings.ToLower(strings.TrimSpace(nameOf(items[j])))
		if left == right {
			return strings.TrimSpace(idOf(items[i])) < strings.TrimSpace(idOf(items[j]))
		}
		return left < right
	})
}

func (s service) campaignParticipants(ctx context.Context, campaignID string) ([]CampaignParticipant, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignParticipant{}, nil
	}

	participants, err := s.readGateway.CampaignParticipants(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if len(participants) == 0 {
		return []CampaignParticipant{}, nil
	}

	normalized := make([]CampaignParticipant, 0, len(participants))
	for _, participant := range participants {
		participantID := strings.TrimSpace(participant.ID)
		participantUserID := strings.TrimSpace(participant.UserID)
		participantName := strings.TrimSpace(participant.Name)
		if participantName == "" {
			if participantID != "" {
				participantName = participantID
			} else {
				participantName = "Unknown participant"
			}
		}
		role := strings.TrimSpace(participant.Role)
		if role == "" {
			role = "Unspecified"
		}
		campaignAccess := strings.TrimSpace(participant.CampaignAccess)
		if campaignAccess == "" {
			campaignAccess = "Unspecified"
		}
		controller := strings.TrimSpace(participant.Controller)
		if controller == "" {
			controller = "Unspecified"
		}
		normalized = append(normalized, CampaignParticipant{
			ID:             participantID,
			UserID:         participantUserID,
			Name:           participantName,
			Role:           role,
			CampaignAccess: campaignAccess,
			Controller:     controller,
			Pronouns:       strings.TrimSpace(participant.Pronouns),
			AvatarURL:      strings.TrimSpace(participant.AvatarURL),
		})
	}

	sortByName(normalized, func(p CampaignParticipant) string { return p.Name }, func(p CampaignParticipant) string { return p.ID })

	return normalized, nil
}

func (s service) campaignCharacters(ctx context.Context, campaignID string) ([]CampaignCharacter, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignCharacter{}, nil
	}

	characters, err := s.readGateway.CampaignCharacters(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if len(characters) == 0 {
		return []CampaignCharacter{}, nil
	}

	normalized := make([]CampaignCharacter, 0, len(characters))
	for _, character := range characters {
		characterID := strings.TrimSpace(character.ID)
		characterName := strings.TrimSpace(character.Name)
		if characterName == "" {
			if characterID != "" {
				characterName = characterID
			} else {
				characterName = "Unknown character"
			}
		}
		kind := strings.TrimSpace(character.Kind)
		if kind == "" {
			kind = "Unspecified"
		}
		controller := strings.TrimSpace(character.Controller)
		if controller == "" {
			controller = "Unassigned"
		}
		normalized = append(normalized, CampaignCharacter{
			ID:         characterID,
			Name:       characterName,
			Kind:       kind,
			Controller: controller,
			Pronouns:   strings.TrimSpace(character.Pronouns),
			Aliases:    append([]string(nil), character.Aliases...),
			AvatarURL:  strings.TrimSpace(character.AvatarURL),
		})
	}

	sortByName(normalized, func(c CampaignCharacter) string { return c.Name }, func(c CampaignCharacter) string { return c.ID })

	s.hydrateCharacterEditability(ctx, campaignID, normalized)

	return normalized, nil
}

func (s service) campaignSessions(ctx context.Context, campaignID string) ([]CampaignSession, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignSession{}, nil
	}

	sessions, err := s.readGateway.CampaignSessions(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return []CampaignSession{}, nil
	}

	normalized := make([]CampaignSession, 0, len(sessions))
	for _, session := range sessions {
		sessionID := strings.TrimSpace(session.ID)
		sessionName := strings.TrimSpace(session.Name)
		if sessionName == "" {
			if sessionID != "" {
				sessionName = sessionID
			} else {
				sessionName = "Unnamed session"
			}
		}
		status := strings.TrimSpace(session.Status)
		if status == "" {
			status = "Unspecified"
		}
		normalized = append(normalized, CampaignSession{
			ID:        sessionID,
			Name:      sessionName,
			Status:    status,
			StartedAt: strings.TrimSpace(session.StartedAt),
			UpdatedAt: strings.TrimSpace(session.UpdatedAt),
			EndedAt:   strings.TrimSpace(session.EndedAt),
		})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		leftUpdated := strings.TrimSpace(normalized[i].UpdatedAt)
		rightUpdated := strings.TrimSpace(normalized[j].UpdatedAt)
		if leftUpdated == rightUpdated {
			return strings.TrimSpace(normalized[i].ID) < strings.TrimSpace(normalized[j].ID)
		}
		return leftUpdated > rightUpdated
	})

	return normalized, nil
}

func (s service) campaignInvites(ctx context.Context, campaignID string) ([]CampaignInvite, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignInvite{}, nil
	}

	invites, err := s.readGateway.CampaignInvites(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if len(invites) == 0 {
		return []CampaignInvite{}, nil
	}

	normalized := make([]CampaignInvite, 0, len(invites))
	for _, invite := range invites {
		status := strings.TrimSpace(invite.Status)
		if status == "" {
			status = "Unspecified"
		}
		normalized = append(normalized, CampaignInvite{
			ID:              strings.TrimSpace(invite.ID),
			ParticipantID:   strings.TrimSpace(invite.ParticipantID),
			RecipientUserID: strings.TrimSpace(invite.RecipientUserID),
			Status:          status,
		})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		leftID := strings.TrimSpace(normalized[i].ID)
		rightID := strings.TrimSpace(normalized[j].ID)
		return leftID < rightID
	})

	return normalized, nil
}
