package protocol

import (
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const RealtimeProtocolVersion = 1

type Bootstrap struct {
	CampaignID                 string                         `json:"campaign_id"`
	Viewer                     *InteractionViewer             `json:"viewer,omitempty"`
	System                     System                         `json:"system"`
	InteractionState           InteractionState               `json:"interaction_state"`
	Participants               []Participant                  `json:"participants"`
	CharacterInspectionCatalog map[string]CharacterInspection `json:"character_inspection_catalog"`
	Chat                       ChatSnapshot                   `json:"chat"`
	Realtime                   RealtimeConfig                 `json:"realtime"`
}

type System struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

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

type RealtimeConfig struct {
	URL             string `json:"url"`
	ProtocolVersion int    `json:"protocol_version"`
	TypingTTLMs     int    `json:"typing_ttl_ms,omitempty"`
}

type ChatSnapshot struct {
	SessionID        string        `json:"session_id"`
	LatestSequenceID int64         `json:"latest_sequence_id"`
	Messages         []ChatMessage `json:"messages"`
	HistoryURL       string        `json:"history_url"`
}

type ChatMessage struct {
	MessageID       string    `json:"message_id"`
	CampaignID      string    `json:"campaign_id"`
	SessionID       string    `json:"session_id"`
	SequenceID      int64     `json:"sequence_id"`
	SentAt          string    `json:"sent_at"`
	Actor           ChatActor `json:"actor"`
	Body            string    `json:"body"`
	ClientMessageID string    `json:"client_message_id,omitempty"`
}

type ChatActor struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
}

type HistoryResponse struct {
	SessionID string        `json:"session_id"`
	Messages  []ChatMessage `json:"messages"`
}

type RoomSnapshot struct {
	InteractionState           InteractionState               `json:"interaction_state"`
	Participants               []Participant                  `json:"participants"`
	CharacterInspectionCatalog map[string]CharacterInspection `json:"character_inspection_catalog"`
	Chat                       ChatSnapshot                   `json:"chat"`
	LatestGameSeq              uint64                         `json:"latest_game_sequence"`
}

type WSFrame struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   any    `json:"payload,omitempty"`
}

type WSRawFrame struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   []byte `json:"payload,omitempty"`
}

type ErrorEnvelope struct {
	Error ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

type ChatMessageEnvelope struct {
	Message ChatMessage `json:"message"`
}

type TypingEvent struct {
	SessionID     string `json:"session_id,omitempty"`
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
	Active        bool   `json:"active"`
}

type ConnectRequest struct {
	CampaignID  string `json:"campaign_id"`
	LastGameSeq uint64 `json:"last_game_seq,omitempty"`
	LastChatSeq int64  `json:"last_chat_seq,omitempty"`
}

type ChatSendRequest struct {
	ClientMessageID string `json:"client_message_id"`
	Body            string `json:"body"`
}

type Pong struct {
	Timestamp string `json:"timestamp,omitempty"`
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

func TranscriptMessage(message transcript.Message) ChatMessage {
	return ChatMessage{
		MessageID:       strings.TrimSpace(message.MessageID),
		CampaignID:      strings.TrimSpace(message.CampaignID),
		SessionID:       strings.TrimSpace(message.SessionID),
		SequenceID:      message.SequenceID,
		SentAt:          strings.TrimSpace(message.SentAt),
		Actor:           ChatActor{ParticipantID: strings.TrimSpace(message.Actor.ParticipantID), Name: strings.TrimSpace(message.Actor.Name)},
		Body:            strings.TrimSpace(message.Body),
		ClientMessageID: strings.TrimSpace(message.ClientMessageID),
	}
}

func TranscriptMessages(messages []transcript.Message) []ChatMessage {
	values := make([]ChatMessage, 0, len(messages))
	for _, message := range messages {
		values = append(values, TranscriptMessage(message))
	}
	return values
}

func interactionRoleString(value gamev1.ParticipantRole) string {
	name := strings.TrimSpace(value.String())
	if name == "" || name == gamev1.ParticipantRole_ROLE_UNSPECIFIED.String() {
		return ""
	}
	name = strings.TrimPrefix(name, "PARTICIPANT_ROLE_")
	return strings.ToLower(name)
}

// --- Scene types ---

// InteractionScene represents an active scene in the interaction.
type InteractionScene struct {
	SceneID     string                 `json:"scene_id"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Characters  []InteractionCharacter `json:"characters"`
	GMOutput    *InteractionGMOutput   `json:"gm_output,omitempty"`
}

// InteractionCharacter is a character present in a scene.
type InteractionCharacter struct {
	CharacterID        string `json:"character_id"`
	Name               string `json:"name,omitempty"`
	OwnerParticipantID string `json:"owner_participant_id,omitempty"`
}

// InteractionGMOutput holds the latest GM narrative output for a scene.
type InteractionGMOutput struct {
	Text          string `json:"text,omitempty"`
	ParticipantID string `json:"participant_id,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
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
		SceneID:     sceneID,
		Name:        strings.TrimSpace(scene.GetName()),
		Description: strings.TrimSpace(scene.GetDescription()),
		Characters:  characters,
		GMOutput:    gmOutputFromProto(scene.GetGmOutput()),
	}
}

func gmOutputFromProto(output *gamev1.InteractionGMOutput) *InteractionGMOutput {
	if output == nil {
		return nil
	}
	text := strings.TrimSpace(output.GetText())
	pid := strings.TrimSpace(output.GetParticipantId())
	if text == "" && pid == "" {
		return nil
	}
	return &InteractionGMOutput{
		Text:          text,
		ParticipantID: pid,
		UpdatedAt:     formatTimestamp(output.GetUpdatedAt()),
	}
}

// --- Player Phase types ---

// ScenePlayerPhase represents the current player phase in a scene.
type ScenePlayerPhase struct {
	PhaseID              string            `json:"phase_id"`
	Status               string            `json:"status,omitempty"`
	FrameText            string            `json:"frame_text,omitempty"`
	ActingCharacterIDs   []string          `json:"acting_character_ids"`
	ActingParticipantIDs []string          `json:"acting_participant_ids"`
	Slots                []ScenePlayerSlot `json:"slots"`
}

// ScenePlayerSlot represents one player's submission slot.
type ScenePlayerSlot struct {
	ParticipantID      string   `json:"participant_id"`
	SummaryText        string   `json:"summary_text,omitempty"`
	CharacterIDs       []string `json:"character_ids"`
	UpdatedAt          string   `json:"updated_at,omitempty"`
	Yielded            bool     `json:"yielded"`
	ReviewStatus       string   `json:"review_status,omitempty"`
	ReviewReason       string   `json:"review_reason,omitempty"`
	ReviewCharacterIDs []string `json:"review_character_ids"`
}

// PlayerPhaseFromGamePhase maps a proto ScenePlayerPhase to protocol.
func PlayerPhaseFromGamePhase(phase *gamev1.ScenePlayerPhase) *ScenePlayerPhase {
	if phase == nil {
		return nil
	}
	phaseID := strings.TrimSpace(phase.GetPhaseId())
	if phaseID == "" {
		return nil
	}
	slots := make([]ScenePlayerSlot, 0, len(phase.GetSlots()))
	for _, s := range phase.GetSlots() {
		slots = append(slots, ScenePlayerSlot{
			ParticipantID:      strings.TrimSpace(s.GetParticipantId()),
			SummaryText:        strings.TrimSpace(s.GetSummaryText()),
			CharacterIDs:       trimStringSlice(s.GetCharacterIds()),
			UpdatedAt:          formatTimestamp(s.GetUpdatedAt()),
			Yielded:            s.GetYielded(),
			ReviewStatus:       slotReviewStatusString(s.GetReviewStatus()),
			ReviewReason:       strings.TrimSpace(s.GetReviewReason()),
			ReviewCharacterIDs: trimStringSlice(s.GetReviewCharacterIds()),
		})
	}
	return &ScenePlayerPhase{
		PhaseID:              phaseID,
		Status:               scenePhaseStatusString(phase.GetStatus()),
		FrameText:            strings.TrimSpace(phase.GetFrameText()),
		ActingCharacterIDs:   trimStringSlice(phase.GetActingCharacterIds()),
		ActingParticipantIDs: trimStringSlice(phase.GetActingParticipantIds()),
		Slots:                slots,
	}
}

func scenePhaseStatusString(value gamev1.ScenePhaseStatus) string {
	name := strings.TrimSpace(value.String())
	if name == "" || name == gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_UNSPECIFIED.String() {
		return ""
	}
	name = strings.TrimPrefix(name, "SCENE_PHASE_STATUS_")
	return strings.ToLower(name)
}

func slotReviewStatusString(value gamev1.ScenePlayerSlotReviewStatus) string {
	name := strings.TrimSpace(value.String())
	if name == "" || name == gamev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNSPECIFIED.String() {
		return ""
	}
	name = strings.TrimPrefix(name, "SCENE_PLAYER_SLOT_REVIEW_STATUS_")
	return strings.ToLower(name)
}

// --- OOC types ---

// OOCState represents the out-of-character session pause state.
type OOCState struct {
	Open                        bool      `json:"open"`
	Posts                       []OOCPost `json:"posts"`
	ReadyToResumeParticipantIDs []string  `json:"ready_to_resume_participant_ids"`
}

// OOCPost is a single out-of-character message.
type OOCPost struct {
	PostID        string `json:"post_id"`
	ParticipantID string `json:"participant_id"`
	Body          string `json:"body"`
	CreatedAt     string `json:"created_at,omitempty"`
}

// OOCFromGameOOC maps a proto OOCState to protocol.
func OOCFromGameOOC(ooc *gamev1.OOCState) *OOCState {
	if ooc == nil {
		return nil
	}
	posts := make([]OOCPost, 0, len(ooc.GetPosts()))
	for _, p := range ooc.GetPosts() {
		posts = append(posts, OOCPost{
			PostID:        strings.TrimSpace(p.GetPostId()),
			ParticipantID: strings.TrimSpace(p.GetParticipantId()),
			Body:          strings.TrimSpace(p.GetBody()),
			CreatedAt:     formatTimestamp(p.GetCreatedAt()),
		})
	}
	return &OOCState{
		Open:                        ooc.GetOpen(),
		Posts:                       posts,
		ReadyToResumeParticipantIDs: trimStringSlice(ooc.GetReadyToResumeParticipantIds()),
	}
}

// --- AI Turn types ---

// AITurnState represents the current AI GM turn state.
type AITurnState struct {
	Status             string `json:"status,omitempty"`
	OwnerParticipantID string `json:"owner_participant_id,omitempty"`
	LastError          string `json:"last_error,omitempty"`
}

// AITurnFromGameAITurn maps a proto AITurnState to protocol.
func AITurnFromGameAITurn(aiTurn *gamev1.AITurnState) *AITurnState {
	if aiTurn == nil {
		return nil
	}
	status := aiTurnStatusString(aiTurn.GetStatus())
	owner := strings.TrimSpace(aiTurn.GetOwnerParticipantId())
	lastErr := strings.TrimSpace(aiTurn.GetLastError())
	if status == "" && owner == "" && lastErr == "" {
		return nil
	}
	return &AITurnState{
		Status:             status,
		OwnerParticipantID: owner,
		LastError:          lastErr,
	}
}

func aiTurnStatusString(value gamev1.AITurnStatus) string {
	name := strings.TrimSpace(value.String())
	if name == "" || name == gamev1.AITurnStatus_AI_TURN_STATUS_UNSPECIFIED.String() {
		return ""
	}
	name = strings.TrimPrefix(name, "AI_TURN_STATUS_")
	return strings.ToLower(name)
}

// --- Participant types ---

// Participant represents an enriched campaign participant for the browser.
type Participant struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Role         string   `json:"role,omitempty"`
	AvatarURL    string   `json:"avatar_url,omitempty"`
	CharacterIDs []string `json:"character_ids,omitempty"`
}

// ParticipantFromGameParticipant maps a proto Participant to protocol.
func ParticipantFromGameParticipant(assetBaseURL string, p *gamev1.Participant) Participant {
	if p == nil {
		return Participant{}
	}
	avatarEntityID := strings.TrimSpace(p.GetId())
	if avatarEntityID == "" {
		avatarEntityID = strings.TrimSpace(p.GetUserId())
	}
	if avatarEntityID == "" {
		avatarEntityID = strings.TrimSpace(p.GetCampaignId())
	}
	return Participant{
		ID:   strings.TrimSpace(p.GetId()),
		Name: strings.TrimSpace(p.GetName()),
		Role: interactionRoleString(p.GetRole()),
		AvatarURL: websupport.AvatarImageURL(
			assetBaseURL,
			catalog.AvatarRoleParticipant,
			avatarEntityID,
			strings.TrimSpace(p.GetAvatarSetId()),
			strings.TrimSpace(p.GetAvatarAssetId()),
			playAvatarDeliveryWidthPX,
		),
	}
}

// --- Character inspection types ---

// CharacterInspection holds both card and sheet data for a character.
type CharacterInspection struct {
	System string `json:"system"`
	Card   any    `json:"card"`
	Sheet  any    `json:"sheet"`
}

// --- Locale helper ---

func localeString(value commonv1.Locale) string {
	name := strings.TrimSpace(value.String())
	if name == "" || name == commonv1.Locale_LOCALE_UNSPECIFIED.String() {
		return ""
	}
	name = strings.TrimPrefix(name, "LOCALE_")
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}

// --- Shared helpers ---

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil || (ts.GetSeconds() == 0 && ts.GetNanos() == 0) {
		return ""
	}
	return ts.AsTime().Format(time.RFC3339)
}

func trimStringSlice(values []string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			result = append(result, s)
		}
	}
	return result
}

// --- Pronouns helper ---

func pronounsString(p *commonv1.Pronouns) string {
	if p == nil {
		return ""
	}
	switch v := p.GetValue().(type) {
	case *commonv1.Pronouns_Kind:
		name := strings.TrimSpace(v.Kind.String())
		if name == "" || name == commonv1.Pronoun_PRONOUN_UNSPECIFIED.String() {
			return ""
		}
		name = strings.TrimPrefix(name, "PRONOUN_")
		return strings.ToLower(strings.ReplaceAll(name, "_", "/"))
	case *commonv1.Pronouns_Custom:
		return strings.TrimSpace(v.Custom)
	default:
		return ""
	}
}
