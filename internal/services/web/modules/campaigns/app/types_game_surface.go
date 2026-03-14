package app

// CampaignGameParticipant stores the current viewer's communication identity.
type CampaignGameParticipant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// CampaignGameStream stores one visible communication stream for the game surface.
type CampaignGameStream struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Scope     string `json:"scope"`
	SessionID string `json:"sessionId"`
	SceneID   string `json:"sceneId"`
	Label     string `json:"label"`
}

// CampaignGamePersona stores one available speaking persona for the viewer.
type CampaignGamePersona struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	ParticipantID string `json:"participantId"`
	CharacterID   string `json:"characterId"`
	DisplayName   string `json:"displayName"`
}

// CampaignGameGate stores the authoritative active session gate state exposed to communication surfaces.
type CampaignGameGate struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Status   string         `json:"status"`
	Reason   string         `json:"reason"`
	Metadata map[string]any `json:"metadata"`
	Progress map[string]any `json:"progress"`
}

// CampaignGameSpotlight stores the authoritative active session spotlight state.
type CampaignGameSpotlight struct {
	Type        string `json:"type"`
	CharacterID string `json:"characterId"`
}

// CampaignGameSurface stores the game-surface communication context consumed by the web route.
type CampaignGameSurface struct {
	Participant            CampaignGameParticipant `json:"participant"`
	SessionID              string                  `json:"sessionId"`
	SessionName            string                  `json:"sessionName"`
	DefaultStreamID        string                  `json:"defaultStreamId"`
	DefaultPersonaID       string                  `json:"defaultPersonaId"`
	ActiveSessionGate      *CampaignGameGate       `json:"activeSessionGate"`
	ActiveSessionSpotlight *CampaignGameSpotlight  `json:"activeSessionSpotlight"`
	Streams                []CampaignGameStream    `json:"streams"`
	Personas               []CampaignGamePersona   `json:"personas"`
}
