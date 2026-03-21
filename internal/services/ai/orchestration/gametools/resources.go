package gametools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// readResource dispatches a resource URI to the correct gRPC reader and
// returns the text content.
func (s *DirectSession) readResource(ctx context.Context, uri string) (string, error) {
	switch {
	case uri == "daggerheart://rules/version":
		return s.readDaggerheartRulesVersion(ctx)

	case strings.HasPrefix(uri, "daggerheart://campaign/") && strings.HasSuffix(uri, "/snapshot"):
		return s.readDaggerheartSnapshot(ctx, uri)

	case uri == "context://current":
		return s.readContextCurrent()

	case matchCampaignArtifactURI(uri):
		return s.readCampaignArtifact(ctx, uri)

	case strings.HasSuffix(uri, "/interaction"):
		return s.readInteraction(ctx, uri)

	case strings.HasSuffix(uri, "/scenes"):
		return s.readSceneList(ctx, uri)

	case strings.HasSuffix(uri, "/participants"):
		return s.readParticipantList(ctx, uri)

	case strings.HasSuffix(uri, "/characters"):
		return s.readCharacterList(ctx, uri)

	case strings.HasSuffix(uri, "/sessions"):
		return s.readSessionList(ctx, uri)

	case strings.HasPrefix(uri, "campaign://") && !strings.Contains(strings.TrimPrefix(uri, "campaign://"), "/"):
		return s.readCampaign(ctx, uri)

	default:
		return "", fmt.Errorf("unknown resource URI: %s", uri)
	}
}

// --- context://current ---

func (s *DirectSession) readContextCurrent() (string, error) {
	type contextPayload struct {
		Context struct {
			CampaignID    *string `json:"campaign_id"`
			SessionID     *string `json:"session_id"`
			ParticipantID *string `json:"participant_id"`
		} `json:"context"`
	}
	var p contextPayload
	if s.sc.CampaignID != "" {
		p.Context.CampaignID = &s.sc.CampaignID
	}
	if s.sc.SessionID != "" {
		p.Context.SessionID = &s.sc.SessionID
	}
	if s.sc.ParticipantID != "" {
		p.Context.ParticipantID = &s.sc.ParticipantID
	}
	return marshalIndent(p)
}

// --- daggerheart://rules/version ---

func (s *DirectSession) readDaggerheartRulesVersion(ctx context.Context) (string, error) {
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Daggerheart.RulesVersion(callCtx, &pb.RulesVersionRequest{})
	if err != nil {
		return "", fmt.Errorf("rules version failed: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("rules version response is missing")
	}

	outcomes := make([]string, 0, len(resp.GetOutcomes()))
	for _, o := range resp.GetOutcomes() {
		outcomes = append(outcomes, o.String())
	}
	return marshalIndent(rulesVersionResult{
		System:         resp.GetSystem(),
		Module:         resp.GetModule(),
		RulesVersion:   resp.GetRulesVersion(),
		DiceModel:      resp.GetDiceModel(),
		TotalFormula:   resp.GetTotalFormula(),
		CritRule:       resp.GetCritRule(),
		DifficultyRule: resp.GetDifficultyRule(),
		Outcomes:       outcomes,
	})
}

// --- campaign://{id} ---

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
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound {
				return "", fmt.Errorf("campaign not found")
			}
		}
		return "", fmt.Errorf("get campaign failed: %w", err)
	}
	if resp == nil || resp.Campaign == nil {
		return "", fmt.Errorf("campaign response is missing")
	}
	c := resp.Campaign
	return marshalIndent(campaignPayload{
		Campaign: campaignListEntry{
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
		},
	})
}

// --- campaign://{id}/participants ---

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
	var payload participantListPayload
	for _, p := range resp.GetParticipants() {
		payload.Participants = append(payload.Participants, participantListEntry{
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
	return marshalIndent(payload)
}

// --- campaign://{id}/characters ---

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
	var payload characterListPayload
	for _, ch := range resp.GetCharacters() {
		aliases := ch.GetAliases()
		if len(aliases) == 0 {
			aliases = []string{}
		} else {
			aliases = append([]string(nil), aliases...)
		}
		payload.Characters = append(payload.Characters, characterListEntry{
			ID:         ch.GetId(),
			CampaignID: ch.GetCampaignId(),
			Name:       ch.GetName(),
			Kind:       characterKindToString(ch.GetKind()),
			Notes:      ch.GetNotes(),
			Pronouns:   sharedpronouns.FromProto(ch.GetPronouns()),
			Aliases:    aliases,
			CreatedAt:  formatTimestamp(ch.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(ch.GetUpdatedAt()),
		})
	}
	return marshalIndent(payload)
}

// --- campaign://{id}/sessions ---

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
	for _, sess := range resp.GetSessions() {
		entry := sessionListEntry{
			ID:         sess.GetId(),
			CampaignID: sess.GetCampaignId(),
			Name:       sess.GetName(),
			Status:     sessionStatusToString(sess.GetStatus()),
			StartedAt:  formatTimestamp(sess.GetStartedAt()),
			UpdatedAt:  formatTimestamp(sess.GetUpdatedAt()),
		}
		if sess.GetEndedAt() != nil {
			entry.EndedAt = formatTimestamp(sess.GetEndedAt())
		}
		payload.Sessions = append(payload.Sessions, entry)
	}
	return marshalIndent(payload)
}

// --- campaign://{id}/sessions/{sid}/scenes ---

type sceneListEntry struct {
	SceneID      string   `json:"scene_id"`
	SessionID    string   `json:"session_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Active       bool     `json:"active"`
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
	for _, sc := range resp.GetScenes() {
		entry := sceneListEntry{
			SceneID:      sc.GetSceneId(),
			SessionID:    sc.GetSessionId(),
			Name:         sc.GetName(),
			Description:  sc.GetDescription(),
			Active:       sc.GetActive(),
			CharacterIDs: append([]string(nil), sc.GetCharacterIds()...),
			CreatedAt:    formatTimestamp(sc.GetCreatedAt()),
			UpdatedAt:    formatTimestamp(sc.GetUpdatedAt()),
		}
		if sc.GetEndedAt() != nil {
			entry.EndedAt = formatTimestamp(sc.GetEndedAt())
		}
		payload.Scenes = append(payload.Scenes, entry)
	}
	return marshalIndent(payload)
}

// --- campaign://{id}/interaction ---

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

// --- campaign://{id}/artifacts/{path} ---

func (s *DirectSession) readCampaignArtifact(ctx context.Context, uri string) (string, error) {
	campaignID, artifactPath, err := parseArtifactURI(uri)
	if err != nil {
		return "", err
	}
	if s.clients.Artifact == nil {
		return "", fmt.Errorf("artifact manager is not configured")
	}

	record, err := s.clients.Artifact.GetArtifact(ctx, campaignID, artifactPath)
	if err != nil {
		return "", fmt.Errorf("campaign artifact get failed: %w", err)
	}
	return marshalIndent(artifactFromRecord(record, true))
}

// --- daggerheart://campaign/{id}/snapshot ---

type snapshotPayload struct {
	GmFear                int                   `json:"gm_fear"`
	ConsecutiveShortRests int                   `json:"consecutive_short_rests"`
	Characters            []characterStateEntry `json:"characters"`
}

type characterStateEntry struct {
	CharacterID    string                `json:"character_id"`
	HP             int                   `json:"hp"`
	Hope           int                   `json:"hope"`
	HopeMax        int                   `json:"hope_max"`
	Stress         int                   `json:"stress"`
	Armor          int                   `json:"armor"`
	LifeState      string                `json:"life_state"`
	Conditions     []conditionEntry      `json:"conditions,omitempty"`
	TemporaryArmor []temporaryArmorEntry `json:"temporary_armor,omitempty"`
	StatModifiers  []statModifierEntry   `json:"stat_modifiers,omitempty"`
}

type conditionEntry struct {
	Label         string   `json:"label"`
	ClearTriggers []string `json:"clear_triggers,omitempty"`
}

type temporaryArmorEntry struct {
	Source string `json:"source"`
	Amount int    `json:"amount"`
}

type statModifierEntry struct {
	Target string `json:"target"`
	Delta  int    `json:"delta"`
	Label  string `json:"label"`
}

func (s *DirectSession) readDaggerheartSnapshot(ctx context.Context, uri string) (string, error) {
	campaignID := strings.TrimPrefix(uri, "daggerheart://campaign/")
	campaignID = strings.TrimSuffix(campaignID, "/snapshot")
	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in snapshot URI")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Snapshot.GetSnapshot(callCtx, &statev1.GetSnapshotRequest{CampaignId: campaignID})
	if err != nil {
		return "", fmt.Errorf("get snapshot failed: %w", err)
	}

	snap := resp.GetSnapshot()
	var payload snapshotPayload
	if dh := snap.GetDaggerheart(); dh != nil {
		payload.GmFear = int(dh.GetGmFear())
		payload.ConsecutiveShortRests = int(dh.GetConsecutiveShortRests())
	}

	payload.Characters = make([]characterStateEntry, 0, len(snap.GetCharacterStates()))
	for _, cs := range snap.GetCharacterStates() {
		dh := cs.GetDaggerheart()
		if dh == nil {
			continue
		}

		entry := characterStateEntry{
			CharacterID: cs.GetCharacterId(),
			HP:          int(dh.GetHp()),
			Hope:        int(dh.GetHope()),
			HopeMax:     int(dh.GetHopeMax()),
			Stress:      int(dh.GetStress()),
			Armor:       int(dh.GetArmor()),
			LifeState:   daggerheartLifeStateToString(dh.GetLifeState()),
		}

		for _, cond := range dh.GetConditionStates() {
			triggers := make([]string, 0, len(cond.GetClearTriggers()))
			for _, t := range cond.GetClearTriggers() {
				triggers = append(triggers, daggerheartConditionClearTriggerToString(t))
			}
			entry.Conditions = append(entry.Conditions, conditionEntry{
				Label:         cond.GetLabel(),
				ClearTriggers: triggers,
			})
		}

		for _, ta := range dh.GetTemporaryArmorBuckets() {
			entry.TemporaryArmor = append(entry.TemporaryArmor, temporaryArmorEntry{
				Source: ta.GetSource(),
				Amount: int(ta.GetAmount()),
			})
		}

		for _, sm := range dh.GetStatModifiers() {
			entry.StatModifiers = append(entry.StatModifiers, statModifierEntry{
				Target: sm.GetTarget(),
				Delta:  int(sm.GetDelta()),
				Label:  sm.GetLabel(),
			})
		}

		payload.Characters = append(payload.Characters, entry)
	}

	return marshalIndent(payload)
}

// --- URI parsers ---

func matchCampaignArtifactURI(uri string) bool {
	return strings.HasPrefix(uri, "campaign://") && strings.Contains(uri, "/artifacts/")
}

func parseArtifactURI(uri string) (string, string, error) {
	if !strings.HasPrefix(uri, "campaign://") {
		return "", "", fmt.Errorf("URI must start with \"campaign://\"")
	}
	rest := strings.TrimPrefix(uri, "campaign://")
	parts := strings.SplitN(rest, "/artifacts/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("URI must match campaign://{campaign_id}/artifacts/{path}")
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("campaign and artifact path are required")
	}
	return parts[0], parts[1], nil
}

func parseCampaignIDFromSuffixURI(uri, suffix string) (string, error) {
	prefix := "campaign://"
	fullSuffix := "/" + suffix
	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("URI must start with %q", prefix)
	}
	if !strings.HasSuffix(uri, fullSuffix) {
		return "", fmt.Errorf("URI must end with %q", fullSuffix)
	}
	campaignID := strings.TrimSuffix(strings.TrimPrefix(uri, prefix), fullSuffix)
	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in URI")
	}
	return campaignID, nil
}

func parseSceneListURI(uri string) (string, string, error) {
	if !strings.HasPrefix(uri, "campaign://") {
		return "", "", fmt.Errorf("URI must start with \"campaign://\"")
	}
	rest := strings.TrimPrefix(uri, "campaign://")
	parts := strings.Split(rest, "/")
	if len(parts) != 4 || parts[1] != "sessions" || parts[3] != "scenes" {
		return "", "", fmt.Errorf("URI must match campaign://{campaign_id}/sessions/{session_id}/scenes")
	}
	if parts[0] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("campaign and session IDs are required in URI")
	}
	return parts[0], parts[2], nil
}

func marshalIndent(v any) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
