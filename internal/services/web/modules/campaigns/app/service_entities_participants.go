package app

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

// CampaignParticipants centralizes this web behavior in one helper seam.
func (s participantReadService) CampaignParticipants(ctx context.Context, campaignID string) ([]CampaignParticipant, error) {
	return s.campaignParticipants(ctx, campaignID)
}

// CampaignParticipantCreator centralizes this web behavior in one helper seam.
func (s participantReadService) CampaignParticipantCreator(ctx context.Context, campaignID string) (CampaignParticipantCreator, error) {
	return s.campaignParticipantCreator(ctx, campaignID)
}

// CampaignParticipantEditor centralizes this web behavior in one helper seam.
func (s participantReadService) CampaignParticipantEditor(ctx context.Context, campaignID string, participantID string) (CampaignParticipantEditor, error) {
	return s.campaignParticipantEditor(ctx, campaignID, participantID)
}

// campaignParticipants centralizes this web behavior in one helper seam.
func (s participantReadService) campaignParticipants(ctx context.Context, campaignID string) ([]CampaignParticipant, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignParticipant{}, nil
	}

	participants, err := s.read.CampaignParticipants(ctx, campaignID)
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
func campaignParticipant(ctx context.Context, read CampaignParticipantReadGateway, campaignID string, participantID string) (CampaignParticipant, error) {
	campaignID = strings.TrimSpace(campaignID)
	participantID = strings.TrimSpace(participantID)
	if campaignID == "" || participantID == "" {
		return CampaignParticipant{}, nil
	}

	participant, err := read.CampaignParticipant(ctx, campaignID, participantID)
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

// campaignParticipantCreator centralizes this web behavior in one helper seam.
func (s participantReadService) campaignParticipantCreator(ctx context.Context, campaignID string) (CampaignParticipantCreator, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignParticipantCreator{}, nil
	}
	if err := s.auth.requireManageParticipants(ctx, campaignID); err != nil {
		return CampaignParticipantCreator{}, err
	}

	workspace, err := participantWorkspace(ctx, s.workspace, campaignID)
	if err != nil {
		return CampaignParticipantCreator{}, err
	}

	return CampaignParticipantCreator{
		Role:           participantRolePlayerValue,
		CampaignAccess: participantAccessMember,
		AllowGMRole:    !campaignDisallowsHumanGMParticipants(workspace.GMMode),
		AccessOptions:  s.participantAccessOptions(ctx, campaignID, "", ""),
	}, nil
}

// campaignParticipantEditor centralizes this web behavior in one helper seam.
func (s participantReadService) campaignParticipantEditor(ctx context.Context, campaignID string, participantID string) (CampaignParticipantEditor, error) {
	campaignID = strings.TrimSpace(campaignID)
	participantID = strings.TrimSpace(participantID)
	if campaignID == "" || participantID == "" {
		return CampaignParticipantEditor{}, nil
	}

	participant, err := campaignParticipant(ctx, s.read, campaignID, participantID)
	if err != nil {
		return CampaignParticipantEditor{}, err
	}
	workspace, err := participantWorkspace(ctx, s.workspace, campaignID)
	if err != nil {
		return CampaignParticipantEditor{}, err
	}
	currentAccess := participantAccessCanonical(participant.CampaignAccess)
	if participantIsSelfOwned(ctx, participant) {
		editor := CampaignParticipantEditor{
			Participant:  participant,
			AllowGMRole:  !campaignDisallowsHumanGMParticipants(workspace.GMMode),
			RoleReadOnly: true,
			AccessOptions: []CampaignParticipantAccessOption{{
				Value:   currentAccess,
				Allowed: true,
			}},
			AccessReadOnly: true,
		}
		if participantControllerCanonical(participant.Controller) == participantControllerAI {
			editor.Participant.Role = "GM"
			editor.Participant.CampaignAccess = "Member"
			editor.AccessOptions = []CampaignParticipantAccessOption{{Value: participantAccessMember, Allowed: true}}
		}
		return editor, nil
	}
	target := &AuthorizationTarget{
		ResourceID:           participantID,
		TargetParticipantID:  participantID,
		TargetCampaignAccess: currentAccess,
		ParticipantOperation: ParticipantGovernanceOperationMutate,
	}
	if err := s.auth.requireCampaignActionAccess(
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
	editor := CampaignParticipantEditor{
		Participant:    participant,
		AllowGMRole:    !campaignDisallowsHumanGMParticipants(workspace.GMMode),
		AccessOptions:  options,
		AccessReadOnly: accessReadOnly,
	}
	if participantControllerCanonical(participant.Controller) == participantControllerAI {
		editor.Participant.Role = "GM"
		editor.Participant.CampaignAccess = "Member"
		editor.RoleReadOnly = true
		editor.AccessReadOnly = true
		editor.AccessOptions = []CampaignParticipantAccessOption{{Value: participantAccessMember, Allowed: true}}
	}
	return editor, nil
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
func (s participantReadService) hydrateParticipantEditability(ctx context.Context, campaignID string, participants []CampaignParticipant) {
	if len(participants) == 0 {
		return
	}
	for idx := range participants {
		if participantIsSelfOwned(ctx, participants[idx]) {
			participants[idx].CanEdit = true
			participants[idx].EditReasonCode = "SELF_OWNED_PARTICIPANT"
		}
	}
	if s.batchAuthorization == nil {
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

	decisions, err := s.batchAuthorization.BatchCanCampaignAction(ctx, campaignID, checks)
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

// participantIsSelfOwned identifies the narrow self-edit case so web can expose
// the existing participant editor without broadening campaign-governance checks.
func participantIsSelfOwned(ctx context.Context, participant CampaignParticipant) bool {
	viewerUserID := strings.TrimSpace(grpcauthctx.UserIDFromOutgoingContext(ctx))
	if viewerUserID == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(participant.UserID), viewerUserID)
}

// participantAccessOptions centralizes this web behavior in one helper seam.
func (s participantReadService) participantAccessOptions(ctx context.Context, campaignID string, participantID string, targetAccess string) []CampaignParticipantAccessOption {
	options := make([]CampaignParticipantAccessOption, 0, len(participantAccessValues))
	for _, value := range participantAccessValues {
		options = append(options, CampaignParticipantAccessOption{Value: value})
	}
	if s.batchAuthorization == nil {
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

	decisions, err := s.batchAuthorization.BatchCanCampaignAction(ctx, campaignID, checks)
	if err != nil {
		return options
	}
	allowedChecks := allowedByCheckID(checks, decisions)
	for idx := range options {
		options[idx].Allowed = allowedChecks[options[idx].Value]
	}
	return options
}

// participantWorkspace loads the participant-owned workspace policy inputs used
// by participant editor and mutation flows.
func participantWorkspace(ctx context.Context, workspaceGateway CampaignWorkspaceReadGateway, campaignID string) (CampaignWorkspace, error) {
	workspace, err := loadCampaignWorkspace(ctx, workspaceGateway, campaignID)
	if err != nil {
		return CampaignWorkspace{}, err
	}
	return normalizeCampaignWorkspace(campaignID, workspace), nil
}
