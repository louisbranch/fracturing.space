package campaign

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

// CreatePayload captures the payload for campaign.create commands and campaign.created events.
type CreatePayload struct {
	Name         string `json:"name"`
	Locale       string `json:"locale"`
	GameSystem   string `json:"game_system"`
	GmMode       string `json:"gm_mode"`
	Intent       string `json:"intent,omitempty"`
	AccessPolicy string `json:"access_policy,omitempty"`
	ThemePrompt  string `json:"theme_prompt,omitempty"`
	CoverAssetID string `json:"cover_asset_id,omitempty"`
	CoverSetID   string `json:"cover_set_id,omitempty"`
}

// CreateWithParticipantsPayload captures campaign bootstrap workflow input.
// It emits one campaign.created event and one participant.joined event per participant.
type CreateWithParticipantsPayload struct {
	Campaign     CreatePayload             `json:"campaign"`
	Participants []participant.JoinPayload `json:"participants"`
}

// UpdatePayload captures the payload for campaign.update commands and campaign.updated events.
type UpdatePayload struct {
	Fields map[string]string `json:"fields"`
}

// AIBindPayload captures the payload for campaign.ai_bind commands/events.
type AIBindPayload struct {
	AIAgentID string `json:"ai_agent_id"`
}

// AIUnbindPayload captures the payload for campaign.ai_unbind commands/events.
type AIUnbindPayload struct{}

// AIAuthRotatePayload captures the payload for campaign.ai_auth_rotate commands/events.
type AIAuthRotatePayload struct {
	EpochAfter uint64 `json:"epoch_after"`
	Reason     string `json:"reason"`
}

// ForkPayload captures the payload for campaign.fork commands and campaign.forked events.
type ForkPayload struct {
	ParentCampaignID ids.CampaignID `json:"parent_campaign_id"`
	ForkEventSeq     uint64         `json:"fork_event_seq"`
	OriginCampaignID ids.CampaignID `json:"origin_campaign_id"`
	CopyParticipants bool           `json:"copy_participants"`
}
