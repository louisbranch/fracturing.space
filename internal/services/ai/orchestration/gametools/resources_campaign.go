package gametools

import (
	"context"
	"fmt"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type campaignPayload struct {
	Campaign campaignListEntry `json:"campaign"`
}

type campaignListEntry struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	GmMode           string `json:"gm_mode"`
	Intent           string `json:"intent"`
	AccessPolicy     string `json:"access_policy"`
	ParticipantCount int    `json:"participant_count"`
	CharacterCount   int    `json:"character_count"`
	ThemePrompt      string `json:"theme_prompt"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	CompletedAt      string `json:"completed_at,omitempty"`
	ArchivedAt       string `json:"archived_at,omitempty"`
}

func (s *DirectSession) readCampaign(ctx context.Context, uri string) (string, error) {
	campaignID := strings.TrimPrefix(uri, "campaign://")
	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in URI")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Campaign.GetCampaign(callCtx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return "", fmt.Errorf("campaign not found")
		}
		return "", fmt.Errorf("get campaign failed: %w", err)
	}
	if resp == nil || resp.Campaign == nil {
		return "", fmt.Errorf("campaign response is missing")
	}
	campaign := resp.Campaign
	return marshalIndent(campaignPayload{
		Campaign: campaignProtoToEntry(campaign),
	})
}

// campaignProtoToEntry converts a proto Campaign to a campaignListEntry.
func campaignProtoToEntry(c *statev1.Campaign) campaignListEntry {
	return campaignListEntry{
		ID:               c.GetId(),
		Name:             c.GetName(),
		Status:           campaignStatusToString(c.GetStatus()),
		GmMode:           gmModeToString(c.GetGmMode()),
		Intent:           campaignIntentToString(c.GetIntent()),
		AccessPolicy:     campaignAccessPolicyToString(c.GetAccessPolicy()),
		ParticipantCount: int(c.GetParticipantCount()),
		CharacterCount:   int(c.GetCharacterCount()),
		ThemePrompt:      c.GetThemePrompt(),
		CreatedAt:        formatTimestamp(c.GetCreatedAt()),
		UpdatedAt:        formatTimestamp(c.GetUpdatedAt()),
		CompletedAt:      formatTimestamp(c.GetCompletedAt()),
		ArchivedAt:       formatTimestamp(c.GetArchivedAt()),
	}
}

type participantListEntry struct {
	ID         string `json:"id"`
	CampaignID string `json:"campaign_id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Controller string `json:"controller"`
	Pronouns   string `json:"pronouns"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type participantListPayload struct {
	Participants []participantListEntry `json:"participants"`
}

func (s *DirectSession) readParticipantList(ctx context.Context, uri string) (string, error) {
	campaignID, err := parseCampaignIDFromSuffixURI(uri, "participants")
	if err != nil {
		return "", err
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Participant.ListParticipants(callCtx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		return "", fmt.Errorf("participant list failed: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("participant list response is missing")
	}
	return marshalIndent(participantListPayload{
		Participants: participantProtosToEntries(resp.GetParticipants()),
	})
}

// participantProtosToEntries converts proto Participant messages to list entries.
func participantProtosToEntries(participants []*statev1.Participant) []participantListEntry {
	out := make([]participantListEntry, 0, len(participants))
	for _, p := range participants {
		out = append(out, participantListEntry{
			ID:         p.GetId(),
			CampaignID: p.GetCampaignId(),
			Name:       p.GetName(),
			Role:       participantRoleToString(p.GetRole()),
			Controller: controllerToString(p.GetController()),
			Pronouns:   sharedpronouns.FromProto(p.GetPronouns()),
			CreatedAt:  formatTimestamp(p.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(p.GetUpdatedAt()),
		})
	}
	return out
}

type characterListEntry struct {
	ID         string   `json:"id"`
	CampaignID string   `json:"campaign_id"`
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Notes      string   `json:"notes"`
	Pronouns   string   `json:"pronouns"`
	Aliases    []string `json:"aliases"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

type characterListPayload struct {
	Characters []characterListEntry `json:"characters"`
}

func (s *DirectSession) readCharacterList(ctx context.Context, uri string) (string, error) {
	campaignID, err := parseCampaignIDFromSuffixURI(uri, "characters")
	if err != nil {
		return "", err
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Character.ListCharacters(callCtx, &statev1.ListCharactersRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		return "", fmt.Errorf("character list failed: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("character list response is missing")
	}
	return marshalIndent(characterListPayload{
		Characters: characterProtosToEntries(resp.GetCharacters()),
	})
}

// characterProtosToEntries converts proto Character messages to list entries.
func characterProtosToEntries(characters []*statev1.Character) []characterListEntry {
	out := make([]characterListEntry, 0, len(characters))
	for _, c := range characters {
		aliases := c.GetAliases()
		if len(aliases) == 0 {
			aliases = []string{}
		} else {
			aliases = append([]string(nil), aliases...)
		}
		out = append(out, characterListEntry{
			ID:         c.GetId(),
			CampaignID: c.GetCampaignId(),
			Name:       c.GetName(),
			Kind:       characterKindToString(c.GetKind()),
			Notes:      c.GetNotes(),
			Pronouns:   sharedpronouns.FromProto(c.GetPronouns()),
			Aliases:    aliases,
			CreatedAt:  formatTimestamp(c.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(c.GetUpdatedAt()),
		})
	}
	return out
}

type sessionListEntry struct {
	ID         string `json:"id"`
	CampaignID string `json:"campaign_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	StartedAt  string `json:"started_at"`
	UpdatedAt  string `json:"updated_at"`
	EndedAt    string `json:"ended_at,omitempty"`
}

type sessionListPayload struct {
	Sessions []sessionListEntry `json:"sessions"`
}

func (s *DirectSession) readSessionList(ctx context.Context, uri string) (string, error) {
	campaignID, err := parseCampaignIDFromSuffixURI(uri, "sessions")
	if err != nil {
		return "", err
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Session.ListSessions(callCtx, &statev1.ListSessionsRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		return "", fmt.Errorf("session list failed: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("session list response is missing")
	}
	var payload sessionListPayload
	for _, session := range resp.GetSessions() {
		entry := sessionListEntry{
			ID:         session.GetId(),
			CampaignID: session.GetCampaignId(),
			Name:       session.GetName(),
			Status:     sessionStatusToString(session.GetStatus()),
			StartedAt:  formatTimestamp(session.GetStartedAt()),
			UpdatedAt:  formatTimestamp(session.GetUpdatedAt()),
		}
		if session.GetEndedAt() != nil {
			entry.EndedAt = formatTimestamp(session.GetEndedAt())
		}
		payload.Sessions = append(payload.Sessions, entry)
	}
	return marshalIndent(payload)
}

func (s *DirectSession) readSessionRecap(ctx context.Context, uri string) (string, error) {
	campaignID, sessionID, err := parseSessionRecapURI(uri)
	if err != nil {
		return "", err
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Session.GetSessionRecap(callCtx, &statev1.GetSessionRecapRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		return "", fmt.Errorf("get session recap failed: %w", err)
	}
	if resp == nil || resp.GetRecap() == nil {
		return "", fmt.Errorf("session recap response is missing")
	}
	return strings.TrimSpace(resp.GetRecap().GetMarkdown()), nil
}

type sceneListEntry struct {
	SceneID      string   `json:"scene_id"`
	SessionID    string   `json:"session_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Open         bool     `json:"open"`
	CharacterIDs []string `json:"character_ids,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	EndedAt      string   `json:"ended_at,omitempty"`
}

type sceneListPayload struct {
	Scenes []sceneListEntry `json:"scenes"`
}

func (s *DirectSession) readSceneList(ctx context.Context, uri string) (string, error) {
	campaignID, sessionID, err := parseSceneListURI(uri)
	if err != nil {
		return "", err
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Scene.ListScenes(callCtx, &statev1.ListScenesRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		PageSize:   20,
	})
	if err != nil {
		return "", fmt.Errorf("list scenes failed: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("scene list response is missing")
	}
	payload := sceneListPayload{Scenes: make([]sceneListEntry, 0, len(resp.GetScenes()))}
	for _, scene := range resp.GetScenes() {
		entry := sceneListEntry{
			SceneID:      scene.GetSceneId(),
			SessionID:    scene.GetSessionId(),
			Name:         scene.GetName(),
			Description:  scene.GetDescription(),
			Open:         scene.GetOpen(),
			CharacterIDs: append([]string(nil), scene.GetCharacterIds()...),
			CreatedAt:    formatTimestamp(scene.GetCreatedAt()),
			UpdatedAt:    formatTimestamp(scene.GetUpdatedAt()),
		}
		if scene.GetEndedAt() != nil {
			entry.EndedAt = formatTimestamp(scene.GetEndedAt())
		}
		payload.Scenes = append(payload.Scenes, entry)
	}
	return marshalIndent(payload)
}

func (s *DirectSession) readInteraction(ctx context.Context, uri string) (string, error) {
	campaignID, err := parseCampaignIDFromSuffixURI(uri, "interaction")
	if err != nil {
		return "", err
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Interaction.GetInteractionState(callCtx, &statev1.GetInteractionStateRequest{CampaignId: campaignID})
	if err != nil {
		return "", fmt.Errorf("get interaction state failed: %w", err)
	}
	if resp == nil || resp.State == nil {
		return "", fmt.Errorf("get interaction state response is missing")
	}
	return marshalIndent(interactionStateFromProto(resp.State))
}

func parseSessionRecapURI(uri string) (campaignID string, sessionID string, err error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(uri), "campaign://")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 4 || parts[1] != "sessions" || parts[3] != "recap" {
		return "", "", fmt.Errorf("invalid session recap URI: %s", uri)
	}
	campaignID = strings.TrimSpace(parts[0])
	sessionID = strings.TrimSpace(parts[2])
	if campaignID == "" || sessionID == "" {
		return "", "", fmt.Errorf("invalid session recap URI: %s", uri)
	}
	return campaignID, sessionID, nil
}
