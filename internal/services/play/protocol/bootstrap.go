package protocol

// Bootstrap is the initial payload sent to the play UI on page load.
type Bootstrap struct {
	CampaignID                 string                         `json:"campaign_id"`
	AIDebugEnabled             bool                           `json:"ai_debug_enabled,omitempty"`
	Viewer                     *InteractionViewer             `json:"viewer,omitempty"`
	System                     System                         `json:"system"`
	InteractionState           InteractionState               `json:"interaction_state"`
	Participants               []Participant                  `json:"participants"`
	CharacterInspectionCatalog map[string]CharacterInspection `json:"character_inspection_catalog"`
	Chat                       ChatSnapshot                   `json:"chat"`
	Realtime                   RealtimeConfig                 `json:"realtime"`
	TransitionSFX              *TransitionSFX                 `json:"transition_sfx,omitempty"`
}

// TransitionSFX holds resolved CDN URLs for scene and interaction transition
// sound effects delivered to the play UI at bootstrap.
type TransitionSFX struct {
	SceneChangeURL       string `json:"scene_change_url,omitempty"`
	InteractionChangeURL string `json:"interaction_change_url,omitempty"`
}

// System identifies the game system and version backing the campaign.
type System struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

// RealtimeConfig holds the websocket endpoint and protocol metadata for
// establishing a realtime connection.
type RealtimeConfig struct {
	URL             string `json:"url"`
	ProtocolVersion int    `json:"protocol_version"`
	TypingTTLMs     int    `json:"typing_ttl_ms,omitempty"`
}
