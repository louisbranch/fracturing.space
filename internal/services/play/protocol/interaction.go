package protocol

import (
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

type InteractionState struct {
	CampaignID               string              `json:"campaign_id"`
	CampaignName             string              `json:"campaign_name,omitempty"`
	Locale                   string              `json:"locale,omitempty"`
	Viewer                   *InteractionViewer  `json:"viewer,omitempty"`
	ActiveSession            *InteractionSession `json:"active_session,omitempty"`
	ActiveScene              *InteractionScene   `json:"active_scene,omitempty"`
	PlayerPhase              *ScenePlayerPhase   `json:"player_phase,omitempty"`
	OOC                      *OOCState           `json:"ooc,omitempty"`
	GMAuthorityParticipantID string              `json:"gm_authority_participant_id,omitempty"`
	AITurn                   *AITurnState        `json:"ai_turn,omitempty"`
}

type InteractionViewer struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
	Role          string `json:"role,omitempty"`
}

type InteractionSession struct {
	SessionID string `json:"session_id"`
	Name      string `json:"name,omitempty"`
}

// InteractionScene represents an active scene in the interaction.
type InteractionScene struct {
	SceneID            string                     `json:"scene_id"`
	Name               string                     `json:"name,omitempty"`
	Description        string                     `json:"description,omitempty"`
	Characters         []InteractionCharacter     `json:"characters"`
	CurrentInteraction *InteractionGMInteraction  `json:"current_interaction,omitempty"`
	InteractionHistory []InteractionGMInteraction `json:"interaction_history,omitempty"`
}

// InteractionCharacter is a character present in a scene.
type InteractionCharacter struct {
	CharacterID        string `json:"character_id"`
	Name               string `json:"name,omitempty"`
	OwnerParticipantID string `json:"owner_participant_id,omitempty"`
}

type InteractionGMInteractionIllustration struct {
	ImageURL string `json:"image_url,omitempty"`
	Alt      string `json:"alt,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

type InteractionGMInteractionBeat struct {
	BeatID string `json:"beat_id"`
	Type   string `json:"type,omitempty"`
	Text   string `json:"text,omitempty"`
}

type InteractionGMInteraction struct {
	InteractionID string                                `json:"interaction_id"`
	SceneID       string                                `json:"scene_id,omitempty"`
	PhaseID       string                                `json:"phase_id,omitempty"`
	ParticipantID string                                `json:"participant_id,omitempty"`
	Title         string                                `json:"title,omitempty"`
	CharacterIDs  []string                              `json:"character_ids"`
	Illustration  *InteractionGMInteractionIllustration `json:"illustration,omitempty"`
	Beats         []InteractionGMInteractionBeat        `json:"beats"`
	CreatedAt     string                                `json:"created_at,omitempty"`
}

func InteractionStateFromGameState(state *gamev1.InteractionState) InteractionState {
	if state == nil {
		return InteractionState{}
	}
	return InteractionState{
		CampaignID:               strings.TrimSpace(state.GetCampaignId()),
		CampaignName:             strings.TrimSpace(state.GetCampaignName()),
		Locale:                   localeString(state.GetLocale()),
		Viewer:                   ViewerFromGameViewer(state.GetViewer()),
		ActiveSession:            SessionFromGameSession(state.GetActiveSession()),
		ActiveScene:              SceneFromGameScene(state.GetActiveScene()),
		PlayerPhase:              PlayerPhaseFromGamePhase(state.GetPlayerPhase()),
		OOC:                      OOCFromGameOOC(state.GetOoc()),
		GMAuthorityParticipantID: strings.TrimSpace(state.GetGmAuthorityParticipantId()),
		AITurn:                   AITurnFromGameAITurn(state.GetAiTurn()),
	}
}

func ViewerFromGameViewer(viewer *gamev1.InteractionViewer) *InteractionViewer {
	if viewer == nil {
		return nil
	}
	value := &InteractionViewer{
		ParticipantID: strings.TrimSpace(viewer.GetParticipantId()),
		Name:          strings.TrimSpace(viewer.GetName()),
		Role:          interactionRoleString(viewer.GetRole()),
	}
	if value.ParticipantID == "" && value.Name == "" && value.Role == "" {
		return nil
	}
	return value
}

func SessionFromGameSession(session *gamev1.InteractionSession) *InteractionSession {
	if session == nil {
		return nil
	}
	value := &InteractionSession{
		SessionID: strings.TrimSpace(session.GetSessionId()),
		Name:      strings.TrimSpace(session.GetName()),
	}
	if value.SessionID == "" && value.Name == "" {
		return nil
	}
	return value
}

// SceneFromGameScene maps a proto InteractionScene to protocol.
func SceneFromGameScene(scene *gamev1.InteractionScene) *InteractionScene {
	if scene == nil {
		return nil
	}
	sceneID := strings.TrimSpace(scene.GetSceneId())
	if sceneID == "" {
		return nil
	}
	characters := make([]InteractionCharacter, 0, len(scene.GetCharacters()))
	for _, c := range scene.GetCharacters() {
		characters = append(characters, InteractionCharacter{
			CharacterID:        strings.TrimSpace(c.GetCharacterId()),
			Name:               strings.TrimSpace(c.GetName()),
			OwnerParticipantID: strings.TrimSpace(c.GetOwnerParticipantId()),
		})
	}
	return &InteractionScene{
		SceneID:            sceneID,
		Name:               strings.TrimSpace(scene.GetName()),
		Description:        strings.TrimSpace(scene.GetDescription()),
		Characters:         characters,
		CurrentInteraction: gmInteractionFromProto(scene.GetCurrentInteraction()),
		InteractionHistory: gmInteractionHistoryFromProto(scene.GetInteractionHistory()),
	}
}

func gmInteractionFromProto(interaction *gamev1.GMInteraction) *InteractionGMInteraction {
	if interaction == nil {
		return nil
	}
	interactionID := strings.TrimSpace(interaction.GetInteractionId())
	if interactionID == "" {
		return nil
	}
	beats := make([]InteractionGMInteractionBeat, 0, len(interaction.GetBeats()))
	for _, beat := range interaction.GetBeats() {
		beats = append(beats, InteractionGMInteractionBeat{
			BeatID: strings.TrimSpace(beat.GetBeatId()),
			Type:   gmInteractionBeatTypeString(beat.GetType()),
			Text:   strings.TrimSpace(beat.GetText()),
		})
	}
	result := &InteractionGMInteraction{
		InteractionID: interactionID,
		SceneID:       strings.TrimSpace(interaction.GetSceneId()),
		PhaseID:       strings.TrimSpace(interaction.GetPhaseId()),
		ParticipantID: strings.TrimSpace(interaction.GetParticipantId()),
		Title:         strings.TrimSpace(interaction.GetTitle()),
		CharacterIDs:  TrimStringSlice(interaction.GetCharacterIds()),
		Beats:         beats,
		CreatedAt:     FormatTimestamp(interaction.GetCreatedAt()),
	}
	if illustration := interaction.GetIllustration(); illustration != nil {
		result.Illustration = &InteractionGMInteractionIllustration{
			ImageURL: strings.TrimSpace(illustration.GetImageUrl()),
			Alt:      strings.TrimSpace(illustration.GetAlt()),
			Caption:  strings.TrimSpace(illustration.GetCaption()),
		}
	}
	return result
}

func gmInteractionHistoryFromProto(items []*gamev1.GMInteraction) []InteractionGMInteraction {
	result := make([]InteractionGMInteraction, 0, len(items))
	for _, item := range items {
		if interaction := gmInteractionFromProto(item); interaction != nil {
			result = append(result, *interaction)
		}
	}
	return result
}

func gmInteractionBeatTypeString(value gamev1.GMInteractionBeatType) string {
	return ProtoEnumToLower(value, gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_UNSPECIFIED, "GM_INTERACTION_BEAT_TYPE_")
}
