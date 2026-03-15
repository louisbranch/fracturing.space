package notificationpayload

import (
	"encoding/json"
	"strings"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
)

const (
	ActionMethodGet  = "GET"
	ActionMethodPost = "POST"

	ActionStylePrimary   = "primary"
	ActionStyleSecondary = "secondary"

	ActionKindPublicInviteView = "public_invite_view"
	ActionKindAppCampaignOpen  = "app_campaign_open"
)

// InAppPayload is the canonical inbox payload stored in notification payload_json.
type InAppPayload struct {
	Title   platformi18n.CopyRef `json:"title"`
	Body    platformi18n.CopyRef `json:"body"`
	Facts   []PayloadFact        `json:"facts,omitempty"`
	Actions []PayloadAction      `json:"actions,omitempty"`
}

// PayloadFact is one labeled row in the notification detail view.
type PayloadFact struct {
	Label platformi18n.CopyRef `json:"label"`
	Value string               `json:"value"`
}

// PayloadAction is one inbox action rendered from notification payload.
type PayloadAction struct {
	Label    platformi18n.CopyRef `json:"label"`
	Kind     string               `json:"kind"`
	TargetID string               `json:"target_id,omitempty"`
	Method   string               `json:"method,omitempty"`
	Style    string               `json:"style,omitempty"`
}

// ViewInvitationAction returns the canonical inbox action for opening a public invite.
func ViewInvitationAction(inviteID string) PayloadAction {
	return PayloadAction{
		Label:    platformi18n.NewCopyRef("notification.action.view_invitation"),
		Kind:     ActionKindPublicInviteView,
		TargetID: strings.TrimSpace(inviteID),
		Method:   ActionMethodGet,
		Style:    ActionStylePrimary,
	}
}

// OpenCampaignAction returns the canonical inbox action for opening a campaign.
func OpenCampaignAction(campaignID string) PayloadAction {
	return PayloadAction{
		Label:    platformi18n.NewCopyRef("notification.action.open_campaign"),
		Kind:     ActionKindAppCampaignOpen,
		TargetID: strings.TrimSpace(campaignID),
		Method:   ActionMethodGet,
		Style:    ActionStylePrimary,
	}
}

// ParseInAppPayload decodes the canonical inbox payload contract.
func ParseInAppPayload(raw string) (InAppPayload, bool) {
	if strings.TrimSpace(raw) == "" {
		return InAppPayload{}, false
	}
	var payload InAppPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return InAppPayload{}, false
	}
	return normalizeInAppPayload(payload)
}

func normalizeInAppPayload(payload InAppPayload) (InAppPayload, bool) {
	payload.Title, _ = platformi18n.NormalizeCopyRef(payload.Title)
	payload.Body, _ = platformi18n.NormalizeCopyRef(payload.Body)
	payload.Facts = normalizePayloadFacts(payload.Facts)
	payload.Actions = normalizePayloadActions(payload.Actions)
	if payload.Title.Key == "" && payload.Body.Key == "" && len(payload.Facts) == 0 && len(payload.Actions) == 0 {
		return InAppPayload{}, false
	}
	return payload, true
}

func normalizePayloadFacts(facts []PayloadFact) []PayloadFact {
	if len(facts) == 0 {
		return nil
	}
	result := make([]PayloadFact, 0, len(facts))
	for _, fact := range facts {
		fact.Label, _ = platformi18n.NormalizeCopyRef(fact.Label)
		fact.Value = strings.TrimSpace(fact.Value)
		if fact.Label.Key == "" || fact.Value == "" {
			continue
		}
		result = append(result, fact)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizePayloadActions(actions []PayloadAction) []PayloadAction {
	if len(actions) == 0 {
		return nil
	}
	result := make([]PayloadAction, 0, len(actions))
	for _, action := range actions {
		action.Label, _ = platformi18n.NormalizeCopyRef(action.Label)
		action.Kind = strings.ToLower(strings.TrimSpace(action.Kind))
		action.TargetID = strings.TrimSpace(action.TargetID)
		action.Method = strings.ToUpper(strings.TrimSpace(action.Method))
		action.Style = strings.ToLower(strings.TrimSpace(action.Style))
		if action.Label.Key == "" || action.TargetID == "" {
			continue
		}
		if !supportedActionKind(action.Kind) {
			continue
		}
		if action.Method != ActionMethodPost {
			action.Method = ActionMethodGet
		}
		if action.Style != ActionStylePrimary {
			action.Style = ActionStyleSecondary
		}
		result = append(result, action)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func supportedActionKind(kind string) bool {
	switch kind {
	case ActionKindPublicInviteView, ActionKindAppCampaignOpen:
		return true
	default:
		return false
	}
}
