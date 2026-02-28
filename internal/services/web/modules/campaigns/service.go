package campaigns

import (
	"context"
	"sort"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// CampaignSummary is a transport-safe summary for campaign listings.
type CampaignSummary struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Theme             string `json:"theme"`
	CoverImageURL     string `json:"coverImageUrl"`
	ParticipantCount  string `json:"participantCount"`
	CharacterCount    string `json:"characterCount"`
	CreatedAtUnixNano int64  `json:"createdAtUnixNano"`
}

// CampaignWorkspace stores campaign details used by campaign workspace routes.
type CampaignWorkspace struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Theme         string `json:"theme"`
	System        string `json:"system"`
	GMMode        string `json:"gmMode"`
	CoverImageURL string `json:"coverImageUrl"`
}

// CampaignParticipant stores participant details used by campaign participants pages.
type CampaignParticipant struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	CampaignAccess string `json:"campaignAccess"`
	Controller     string `json:"controller"`
	Pronouns       string `json:"pronouns"`
	AvatarURL      string `json:"avatarUrl"`
}

// CampaignCharacter stores character details used by campaign characters pages.
type CampaignCharacter struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	Controller     string   `json:"controller"`
	Pronouns       string   `json:"pronouns"`
	Aliases        []string `json:"aliases"`
	AvatarURL      string   `json:"avatarUrl"`
	CanEdit        bool     `json:"canEdit"`
	EditReasonCode string   `json:"editReasonCode"`
}

// CampaignSession stores session details used by campaign sessions pages.
type CampaignSession struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	StartedAt string `json:"startedAt"`
	UpdatedAt string `json:"updatedAt"`
	EndedAt   string `json:"endedAt"`
}

// CampaignInvite stores invite details used by campaign invites pages.
type CampaignInvite struct {
	ID              string `json:"id"`
	ParticipantID   string `json:"participantId"`
	RecipientUserID string `json:"recipientUserId"`
	Status          string `json:"status"`
}

// CreateCampaignInput stores create-campaign form values.
type CreateCampaignInput struct {
	Name        string
	Locale      commonv1.Locale
	System      commonv1.GameSystem
	GMMode      statev1.GmMode
	ThemePrompt string
}

// CreateCampaignResult stores create-campaign response values.
type CreateCampaignResult struct {
	CampaignID string
}

type campaignReadGateway interface {
	ListCampaigns(context.Context) ([]CampaignSummary, error)
	CampaignName(context.Context, string) (string, error)
	CampaignWorkspace(context.Context, string) (CampaignWorkspace, error)
	CampaignParticipants(context.Context, string) ([]CampaignParticipant, error)
	CampaignCharacters(context.Context, string) ([]CampaignCharacter, error)
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error)
}

type campaignMutationGateway interface {
	StartSession(context.Context, string) error
	EndSession(context.Context, string) error
	UpdateParticipants(context.Context, string) error
	CreateCharacter(context.Context, string) error
	UpdateCharacter(context.Context, string) error
	ControlCharacter(context.Context, string) error
	CreateInvite(context.Context, string) error
	RevokeInvite(context.Context, string) error
}

type campaignAuthorizationDecision struct {
	CheckID    string
	Evaluated  bool
	Allowed    bool
	ReasonCode string
}

type campaignAuthorizationCheck struct {
	CheckID  string
	Action   statev1.AuthorizationAction
	Resource statev1.AuthorizationResource
	Target   *statev1.AuthorizationTarget
}

type campaignAuthorizationGateway interface {
	CanCampaignAction(context.Context, string, statev1.AuthorizationAction, statev1.AuthorizationResource, *statev1.AuthorizationTarget) (campaignAuthorizationDecision, error)
}

type campaignBatchAuthorizationGateway interface {
	BatchCanCampaignAction(context.Context, string, []campaignAuthorizationCheck) ([]campaignAuthorizationDecision, error)
}

// CampaignGateway loads campaign summaries and applies workspace mutations.
type CampaignGateway interface {
	campaignReadGateway
	campaignMutationGateway
}

type service struct {
	readGateway     campaignReadGateway
	mutationGateway campaignMutationGateway
}

type unavailableGateway struct{}

func (unavailableGateway) ListCampaigns(context.Context) ([]CampaignSummary, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignName(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignWorkspace(context.Context, string) (CampaignWorkspace, error) {
	return CampaignWorkspace{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignParticipants(context.Context, string) ([]CampaignParticipant, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignCharacters(context.Context, string) ([]CampaignCharacter, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignSessions(context.Context, string) ([]CampaignSession, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignInvites(context.Context, string) ([]CampaignInvite, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error) {
	return CreateCampaignResult{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) StartSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) EndSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) UpdateParticipants(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CreateCharacter(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) UpdateCharacter(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) ControlCharacter(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CreateInvite(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) RevokeInvite(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func newService(gateway CampaignGateway) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{readGateway: gateway, mutationGateway: gateway}
}

func (s service) listCampaigns(ctx context.Context) ([]CampaignSummary, error) {
	items, err := s.readGateway.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []CampaignSummary{}, nil
	}
	sorted := append([]CampaignSummary(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].CreatedAtUnixNano > sorted[j].CreatedAtUnixNano
	})
	return sorted, nil
}

func (s service) createCampaign(ctx context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	if strings.TrimSpace(input.Name) == "" {
		return CreateCampaignResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_name_is_required", "campaign name is required")
	}
	created, err := s.readGateway.CreateCampaign(ctx, input)
	if err != nil {
		return CreateCampaignResult{}, err
	}
	if strings.TrimSpace(created.CampaignID) == "" {
		return CreateCampaignResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_campaign_id_was_empty", "created campaign id was empty")
	}
	return created, nil
}

func (s service) campaignName(ctx context.Context, campaignID string) string {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return ""
	}
	name, err := s.readGateway.CampaignName(ctx, campaignID)
	if err != nil {
		return campaignID
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return campaignID
	}
	return name
}

func (s service) campaignWorkspace(ctx context.Context, campaignID string) (CampaignWorkspace, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignWorkspace{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	workspace, err := s.readGateway.CampaignWorkspace(ctx, campaignID)
	if err != nil {
		return CampaignWorkspace{}, err
	}
	workspace.ID = campaignID
	workspace.Name = strings.TrimSpace(workspace.Name)
	if workspace.Name == "" {
		workspace.Name = campaignID
	}
	workspace.Theme = strings.TrimSpace(workspace.Theme)
	workspace.System = strings.TrimSpace(workspace.System)
	if workspace.System == "" {
		workspace.System = "Unspecified"
	}
	workspace.GMMode = strings.TrimSpace(workspace.GMMode)
	if workspace.GMMode == "" {
		workspace.GMMode = "Unspecified"
	}
	workspace.CoverImageURL = strings.TrimSpace(workspace.CoverImageURL)
	if workspace.CoverImageURL == "" {
		workspace.CoverImageURL = campaignCoverImageURL("", campaignID, "", "")
	}
	return workspace, nil
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

	sort.SliceStable(normalized, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(normalized[i].Name))
		rightName := strings.ToLower(strings.TrimSpace(normalized[j].Name))
		if leftName == rightName {
			return strings.TrimSpace(normalized[i].ID) < strings.TrimSpace(normalized[j].ID)
		}
		return leftName < rightName
	})

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

	sort.SliceStable(normalized, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(normalized[i].Name))
		rightName := strings.ToLower(strings.TrimSpace(normalized[j].Name))
		if leftName == rightName {
			return strings.TrimSpace(normalized[i].ID) < strings.TrimSpace(normalized[j].ID)
		}
		return leftName < rightName
	})

	s.hydrateCharacterEditability(ctx, campaignID, normalized)

	return normalized, nil
}

func (s service) hydrateCharacterEditability(ctx context.Context, campaignID string, characters []CampaignCharacter) {
	if len(characters) == 0 {
		return
	}
	checker, ok := s.readGateway.(campaignBatchAuthorizationGateway)
	if !ok {
		return
	}

	checks := make([]campaignAuthorizationCheck, 0, len(characters))
	indexesByCheckID := make(map[string][]int, len(characters))
	for idx := range characters {
		characterID := strings.TrimSpace(characters[idx].ID)
		if characterID == "" {
			continue
		}
		indexesByCheckID[characterID] = append(indexesByCheckID[characterID], idx)
		if len(indexesByCheckID[characterID]) > 1 {
			continue
		}
		checks = append(checks, campaignAuthorizationCheck{
			CheckID:  characterID,
			Action:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
			Resource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
			Target: &statev1.AuthorizationTarget{
				ResourceId: characterID,
			},
		})
	}
	if len(checks) == 0 {
		return
	}

	decisions, err := checker.BatchCanCampaignAction(ctx, campaignID, checks)
	if err != nil {
		return
	}

	for idx, decision := range decisions {
		checkID := strings.TrimSpace(decision.CheckID)
		if checkID == "" && idx < len(checks) {
			checkID = strings.TrimSpace(checks[idx].CheckID)
		}
		if checkID == "" {
			continue
		}
		characterIndexes, found := indexesByCheckID[checkID]
		if !found {
			continue
		}
		for _, characterIndex := range characterIndexes {
			characters[characterIndex].EditReasonCode = strings.TrimSpace(decision.ReasonCode)
			if decision.Evaluated && decision.Allowed {
				characters[characterIndex].CanEdit = true
			}
		}
	}
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

func (s service) startSession(ctx context.Context, campaignID string) error {
	if err := s.requireCampaignActionAccess(
		ctx,
		campaignID,
		statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
		statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION,
		nil,
		"error.web.message.manager_or_owner_access_required_for_session_action",
		"manager or owner access required for session action",
	); err != nil {
		return err
	}
	return s.mutationGateway.StartSession(ctx, campaignID)
}

func (s service) endSession(ctx context.Context, campaignID string) error {
	if err := s.requireCampaignActionAccess(
		ctx,
		campaignID,
		statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
		statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION,
		nil,
		"error.web.message.manager_or_owner_access_required_for_session_action",
		"manager or owner access required for session action",
	); err != nil {
		return err
	}
	return s.mutationGateway.EndSession(ctx, campaignID)
}

func (s service) updateParticipants(ctx context.Context, campaignID string) error {
	if err := s.requireCampaignActionAccess(
		ctx,
		campaignID,
		statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
		statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
		nil,
		"error.web.message.manager_or_owner_access_required_for_participant_action",
		"manager or owner access required for participant action",
	); err != nil {
		return err
	}
	return s.mutationGateway.UpdateParticipants(ctx, campaignID)
}

func (s service) createCharacter(ctx context.Context, campaignID string) error {
	if err := s.requireCampaignActionAccess(
		ctx,
		campaignID,
		statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
		statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
		nil,
		"error.web.message.campaign_membership_required_for_character_action",
		"campaign membership required for character action",
	); err != nil {
		return err
	}
	return s.mutationGateway.CreateCharacter(ctx, campaignID)
}

func (s service) updateCharacter(ctx context.Context, campaignID string) error {
	if err := s.requireCampaignActionAccess(
		ctx,
		campaignID,
		statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
		statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
		nil,
		"error.web.message.campaign_membership_required_for_character_action",
		"campaign membership required for character action",
	); err != nil {
		return err
	}
	return s.mutationGateway.UpdateCharacter(ctx, campaignID)
}

func (s service) controlCharacter(ctx context.Context, campaignID string) error {
	if err := s.requireCampaignActionAccess(
		ctx,
		campaignID,
		statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
		statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
		nil,
		"error.web.message.manager_or_owner_access_required_for_character_action",
		"manager or owner access required for character action",
	); err != nil {
		return err
	}
	return s.mutationGateway.ControlCharacter(ctx, campaignID)
}

func (s service) createInvite(ctx context.Context, campaignID string) error {
	if err := s.requireCampaignActionAccess(
		ctx,
		campaignID,
		statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
		statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_INVITE,
		nil,
		"error.web.message.manager_or_owner_access_required_for_invite_action",
		"manager or owner access required for invite action",
	); err != nil {
		return err
	}
	return s.mutationGateway.CreateInvite(ctx, campaignID)
}

func (s service) revokeInvite(ctx context.Context, campaignID string) error {
	if err := s.requireCampaignActionAccess(
		ctx,
		campaignID,
		statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
		statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_INVITE,
		nil,
		"error.web.message.manager_or_owner_access_required_for_invite_action",
		"manager or owner access required for invite action",
	); err != nil {
		return err
	}
	return s.mutationGateway.RevokeInvite(ctx, campaignID)
}

func (s service) requireCampaignActionAccess(
	ctx context.Context,
	campaignID string,
	action statev1.AuthorizationAction,
	resource statev1.AuthorizationResource,
	target *statev1.AuthorizationTarget,
	denyMessageKey string,
	denyMessage string,
) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	checker, ok := s.readGateway.(campaignAuthorizationGateway)
	if !ok {
		return apperrors.EK(apperrors.KindForbidden, denyMessageKey, denyMessage)
	}
	decision, err := checker.CanCampaignAction(
		ctx,
		campaignID,
		action,
		resource,
		target,
	)
	if err != nil {
		return apperrors.EK(apperrors.KindForbidden, denyMessageKey, denyMessage)
	}
	if !decision.Evaluated || !decision.Allowed {
		return apperrors.EK(apperrors.KindForbidden, denyMessageKey, denyMessage)
	}
	return nil
}
