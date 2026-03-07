package app

import (
	"context"
	"strings"
)

// campaignParticipants centralizes this web behavior in one helper seam.
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
		normalized = append(normalized, normalizeCampaignParticipant(participant))
	}

	s.hydrateParticipantEditability(ctx, campaignID, normalized)

	sortByName(normalized, func(p CampaignParticipant) string { return p.Name }, func(p CampaignParticipant) string { return p.ID })

	return normalized, nil
}

// campaignParticipant centralizes this web behavior in one helper seam.
func (s service) campaignParticipant(ctx context.Context, campaignID string, participantID string) (CampaignParticipant, error) {
	campaignID = strings.TrimSpace(campaignID)
	participantID = strings.TrimSpace(participantID)
	if campaignID == "" || participantID == "" {
		return CampaignParticipant{}, nil
	}

	participant, err := s.readGateway.CampaignParticipant(ctx, campaignID, participantID)
	if err != nil {
		return CampaignParticipant{}, err
	}
	normalized := normalizeCampaignParticipant(participant)
	if strings.TrimSpace(normalized.ID) == "" {
		normalized.ID = participantID
	}
	if strings.TrimSpace(normalized.Name) == "Unknown participant" {
		normalized.Name = participantID
	}
	return normalized, nil
}

// campaignParticipantEditor centralizes this web behavior in one helper seam.
func (s service) campaignParticipantEditor(ctx context.Context, campaignID string, participantID string) (CampaignParticipantEditor, error) {
	campaignID = strings.TrimSpace(campaignID)
	participantID = strings.TrimSpace(participantID)
	if campaignID == "" || participantID == "" {
		return CampaignParticipantEditor{}, nil
	}

	participant, err := s.campaignParticipant(ctx, campaignID, participantID)
	if err != nil {
		return CampaignParticipantEditor{}, err
	}
	currentAccess := participantAccessCanonical(participant.CampaignAccess)
	target := &AuthorizationTarget{
		ResourceID:           participantID,
		TargetParticipantID:  participantID,
		TargetCampaignAccess: currentAccess,
		ParticipantOperation: ParticipantGovernanceOperationMutate,
	}
	if err := s.requireCampaignActionAccess(
		ctx,
		campaignID,
		campaignAuthzActionManage,
		campaignAuthzResourceParticipant,
		target,
		policyManageParticipant.denyKey,
		policyManageParticipant.denyMsg,
	); err != nil {
		return CampaignParticipantEditor{}, err
	}

	options := s.participantAccessOptions(ctx, campaignID, participant.ID, currentAccess)
	accessReadOnly := true
	for _, option := range options {
		if option.Allowed && option.Value != currentAccess {
			accessReadOnly = false
			break
		}
	}
	return CampaignParticipantEditor{
		Participant:    participant,
		AccessOptions:  options,
		AccessReadOnly: accessReadOnly,
	}, nil
}

// normalizeCampaignParticipant centralizes this web behavior in one helper seam.
func normalizeCampaignParticipant(participant CampaignParticipant) CampaignParticipant {
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
	return CampaignParticipant{
		ID:             participantID,
		UserID:         participantUserID,
		Name:           participantName,
		Role:           role,
		CampaignAccess: campaignAccess,
		Controller:     controller,
		Pronouns:       strings.TrimSpace(participant.Pronouns),
		AvatarURL:      strings.TrimSpace(participant.AvatarURL),
		CanEdit:        participant.CanEdit,
		EditReasonCode: strings.TrimSpace(participant.EditReasonCode),
	}
}

// hydrateParticipantEditability centralizes this web behavior in one helper seam.
func (s service) hydrateParticipantEditability(ctx context.Context, campaignID string, participants []CampaignParticipant) {
	if len(participants) == 0 {
		return
	}
	if s.authzGateway == nil {
		return
	}

	checks, indexesByCheckID := buildAuthorizationChecksByID(
		len(participants),
		func(idx int) string { return participants[idx].ID },
		func(checkID string, idx int) AuthorizationCheck {
			return AuthorizationCheck{
				CheckID:  checkID,
				Action:   campaignAuthzActionManage,
				Resource: campaignAuthzResourceParticipant,
				Target: &AuthorizationTarget{
					ResourceID:           checkID,
					TargetParticipantID:  checkID,
					TargetCampaignAccess: participantAccessCanonical(participants[idx].CampaignAccess),
					ParticipantOperation: ParticipantGovernanceOperationMutate,
				},
			}
		},
	)
	if len(checks) == 0 {
		return
	}

	decisions, err := s.authzGateway.BatchCanCampaignAction(ctx, campaignID, checks)
	if err != nil {
		return
	}

	applyAuthorizationDecisions(checks, indexesByCheckID, decisions, func(participantIndex int, decision AuthorizationDecision) {
		participants[participantIndex].EditReasonCode = strings.TrimSpace(decision.ReasonCode)
		if decision.Evaluated && decision.Allowed {
			participants[participantIndex].CanEdit = true
		}
	})
}

// participantAccessOptions centralizes this web behavior in one helper seam.
func (s service) participantAccessOptions(ctx context.Context, campaignID string, participantID string, targetAccess string) []CampaignParticipantAccessOption {
	options := make([]CampaignParticipantAccessOption, 0, len(participantAccessValues))
	for _, value := range participantAccessValues {
		options = append(options, CampaignParticipantAccessOption{Value: value})
	}
	if s.authzGateway == nil {
		return options
	}

	checks := make([]AuthorizationCheck, 0, len(participantAccessValues))
	for _, value := range participantAccessValues {
		checks = append(checks, AuthorizationCheck{
			CheckID:  value,
			Action:   campaignAuthzActionManage,
			Resource: campaignAuthzResourceParticipant,
			Target: &AuthorizationTarget{
				ResourceID:              strings.TrimSpace(participantID),
				TargetParticipantID:     strings.TrimSpace(participantID),
				TargetCampaignAccess:    strings.TrimSpace(targetAccess),
				RequestedCampaignAccess: value,
				ParticipantOperation:    ParticipantGovernanceOperationAccessChange,
			},
		})
	}

	decisions, err := s.authzGateway.BatchCanCampaignAction(ctx, campaignID, checks)
	if err != nil {
		return options
	}
	allowedChecks := allowedByCheckID(checks, decisions)
	for idx := range options {
		options[idx].Allowed = allowedChecks[options[idx].Value]
	}
	return options
}
