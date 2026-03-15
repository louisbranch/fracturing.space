// Package integration defines internal game-to-worker integration contracts.
package integration

const (
	InviteNotificationCreatedOutboxEventType  = "game.invite.created.v1"
	InviteNotificationClaimedOutboxEventType  = "game.invite.claimed.v1"
	InviteNotificationDeclinedOutboxEventType = "game.invite.declined.v1"

	InviteNotificationCreatedMessageType  = "campaign.invite.created.v1"
	InviteNotificationAcceptedMessageType = "campaign.invite.accepted.v1"
	InviteNotificationDeclinedMessageType = "campaign.invite.declined.v1"
)

// InviteNotificationOutboxPayload is the durable worker-facing integration
// payload emitted from invite-domain events.
type InviteNotificationOutboxPayload struct {
	InviteID         string `json:"invite_id"`
	CampaignID       string `json:"campaign_id"`
	RecipientUserID  string `json:"recipient_user_id,omitempty"`
	NotificationKind string `json:"notification_kind"`
}

func InviteCreatedNotificationDedupeKey(inviteID string) string {
	return "invite:" + inviteID + ":created"
}

func InviteAcceptedNotificationDedupeKey(inviteID string) string {
	return "invite:" + inviteID + ":accepted"
}

func InviteDeclinedNotificationDedupeKey(inviteID string) string {
	return "invite:" + inviteID + ":declined"
}
