package app

// CampaignGameParticipant stores the current viewer's interaction identity.
type CampaignGameParticipant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// CampaignGameCharacter stores one visible scene character on the game surface.
type CampaignGameCharacter struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	OwnerParticipantID string `json:"ownerParticipantId"`
}

// CampaignGameScene stores the active scene snapshot shown on the game surface.
type CampaignGameScene struct {
	ID          string                  `json:"id"`
	SessionID   string                  `json:"sessionId"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Characters  []CampaignGameCharacter `json:"characters"`
}

// CampaignGamePlayerSlot stores one participant-owned slot in the current
// player phase, including GM review state.
type CampaignGamePlayerSlot struct {
	ParticipantID      string   `json:"participantId"`
	SummaryText        string   `json:"summaryText"`
	CharacterIDs       []string `json:"characterIds"`
	UpdatedAtUnix      int64    `json:"updatedAtUnix"`
	Yielded            bool     `json:"yielded"`
	ReviewStatus       string   `json:"reviewStatus"`
	ReviewReason       string   `json:"reviewReason"`
	ReviewCharacterIDs []string `json:"reviewCharacterIds"`
}

// CampaignGamePlayerPhase stores the active scene player-phase state.
type CampaignGamePlayerPhase struct {
	PhaseID              string                   `json:"phaseId"`
	Status               string                   `json:"status"`
	FrameText            string                   `json:"frameText"`
	ActingCharacterIDs   []string                 `json:"actingCharacterIds"`
	ActingParticipantIDs []string                 `json:"actingParticipantIds"`
	Slots                []CampaignGamePlayerSlot `json:"slots"`
}

// CampaignGameOOCPost stores one append-only OOC message on the game surface.
type CampaignGameOOCPost struct {
	PostID        string `json:"postId"`
	ParticipantID string `json:"participantId"`
	Body          string `json:"body"`
	CreatedAtUnix int64  `json:"createdAtUnix"`
}

// CampaignGameOOCState stores the session-level OOC pause state.
type CampaignGameOOCState struct {
	Open                        bool                  `json:"open"`
	Posts                       []CampaignGameOOCPost `json:"posts"`
	ReadyToResumeParticipantIDs []string              `json:"readyToResumeParticipantIds"`
}

// CampaignGameAITurn stores the session-level AI GM turn state on the game surface.
type CampaignGameAITurn struct {
	Status             string `json:"status"`
	TurnToken          string `json:"turnToken"`
	OwnerParticipantID string `json:"ownerParticipantId"`
	SourceEventType    string `json:"sourceEventType"`
	SourceSceneID      string `json:"sourceSceneId"`
	SourcePhaseID      string `json:"sourcePhaseId"`
	LastError          string `json:"lastError"`
}

// CampaignGameSurface stores the game-surface interaction context consumed by
// the web route.
type CampaignGameSurface struct {
	Participant              CampaignGameParticipant  `json:"participant"`
	SessionID                string                   `json:"sessionId"`
	SessionName              string                   `json:"sessionName"`
	ActiveScene              *CampaignGameScene       `json:"activeScene"`
	PlayerPhase              *CampaignGamePlayerPhase `json:"playerPhase"`
	OOC                      CampaignGameOOCState     `json:"ooc"`
	GMAuthorityParticipantID string                   `json:"gmAuthorityParticipantId"`
	AITurn                   CampaignGameAITurn       `json:"aiTurn"`
}
