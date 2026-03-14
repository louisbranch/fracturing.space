package game

import (
	"context"
	"encoding/json"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

const (
	inviteNotificationCreatedMessageType  = "campaign.invite.created.v1"
	inviteNotificationAcceptedMessageType = "campaign.invite.accepted.v1"
	inviteNotificationDeclinedMessageType = "campaign.invite.declined.v1"
)

type inviteNotificationPayload struct {
	InviteID          string `json:"invite_id"`
	CampaignID        string `json:"campaign_id"`
	CampaignName      string `json:"campaign_name"`
	ParticipantID     string `json:"participant_id"`
	ParticipantName   string `json:"participant_name"`
	InviterUsername   string `json:"inviter_username,omitempty"`
	RecipientUsername string `json:"recipient_username,omitempty"`
}

func (a inviteApplication) notifyInviteCreated(ctx context.Context, inv storage.InviteRecord) {
	recipientUserID := strings.TrimSpace(inv.RecipientUserID)
	if recipientUserID == "" {
		return
	}
	campaignRecord, seat, ok := a.loadInviteNotificationContext(ctx, inv)
	if !ok {
		return
	}
	inviterUsername, _ := a.lookupParticipantUsername(ctx, inv.CampaignID, inv.CreatedByParticipantID)
	a.createInviteNotification(
		ctx,
		recipientUserID,
		inviteNotificationCreatedMessageType,
		inviteNotificationPayload{
			InviteID:        inv.ID,
			CampaignID:      inv.CampaignID,
			CampaignName:    campaignRecord.Name,
			ParticipantID:   inv.ParticipantID,
			ParticipantName: seat.Name,
			InviterUsername: inviterUsername,
		},
		"invite:"+inv.ID+":created",
	)
}

func (a inviteApplication) notifyInviteClaimed(ctx context.Context, inv storage.InviteRecord, recipientUserID string) {
	a.notifyInviteRecipientOutcome(ctx, inv, recipientUserID, inviteNotificationAcceptedMessageType, "invite:"+inv.ID+":accepted")
}

func (a inviteApplication) notifyInviteDeclined(ctx context.Context, inv storage.InviteRecord, recipientUserID string) {
	a.notifyInviteRecipientOutcome(ctx, inv, recipientUserID, inviteNotificationDeclinedMessageType, "invite:"+inv.ID+":declined")
}

func (a inviteApplication) notifyInviteRecipientOutcome(ctx context.Context, inv storage.InviteRecord, recipientUserID, messageType, dedupeKey string) {
	creatorUserID, ok := a.lookupParticipantUserID(ctx, inv.CampaignID, inv.CreatedByParticipantID)
	if !ok {
		return
	}
	recipientUserID = strings.TrimSpace(recipientUserID)
	if creatorUserID == "" || recipientUserID == "" || creatorUserID == recipientUserID {
		return
	}
	campaignRecord, seat, ok := a.loadInviteNotificationContext(ctx, inv)
	if !ok {
		return
	}
	recipientUsername, _ := a.lookupUsername(ctx, recipientUserID)
	a.createInviteNotification(
		ctx,
		creatorUserID,
		messageType,
		inviteNotificationPayload{
			InviteID:          inv.ID,
			CampaignID:        inv.CampaignID,
			CampaignName:      campaignRecord.Name,
			ParticipantID:     inv.ParticipantID,
			ParticipantName:   seat.Name,
			RecipientUsername: recipientUsername,
		},
		dedupeKey,
	)
}

func (a inviteApplication) createInviteNotification(ctx context.Context, recipientUserID, messageType string, payload inviteNotificationPayload, dedupeKey string) {
	if a.notificationClient == nil {
		return
	}
	recipientUserID = strings.TrimSpace(recipientUserID)
	messageType = strings.TrimSpace(messageType)
	dedupeKey = strings.TrimSpace(dedupeKey)
	if recipientUserID == "" || messageType == "" || dedupeKey == "" {
		return
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = a.notificationClient.CreateNotificationIntent(ctx, &notificationsv1.CreateNotificationIntentRequest{
		RecipientUserId: recipientUserID,
		MessageType:     messageType,
		PayloadJson:     string(payloadJSON),
		DedupeKey:       dedupeKey,
		Source:          notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
	})
}

func (a inviteApplication) loadInviteNotificationContext(ctx context.Context, inv storage.InviteRecord) (storage.CampaignRecord, storage.ParticipantRecord, bool) {
	campaignRecord, err := a.stores.Campaign.Get(ctx, inv.CampaignID)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, false
	}
	seat, err := a.stores.Participant.GetParticipant(ctx, inv.CampaignID, inv.ParticipantID)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, false
	}
	return campaignRecord, seat, true
}

func (a inviteApplication) lookupParticipantUsername(ctx context.Context, campaignID, participantID string) (string, bool) {
	userID, ok := a.lookupParticipantUserID(ctx, campaignID, participantID)
	if !ok {
		return "", false
	}
	return a.lookupUsername(ctx, userID)
}

func (a inviteApplication) lookupParticipantUserID(ctx context.Context, campaignID, participantID string) (string, bool) {
	campaignID = strings.TrimSpace(campaignID)
	participantID = strings.TrimSpace(participantID)
	if campaignID == "" || participantID == "" {
		return "", false
	}
	participantRecord, err := a.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return "", false
	}
	userID := strings.TrimSpace(participantRecord.UserID)
	if userID == "" {
		return "", false
	}
	return userID, true
}

func (a inviteApplication) lookupUsername(ctx context.Context, userID string) (string, bool) {
	if a.authClient == nil {
		return "", false
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", false
	}
	resp, err := a.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
	if err != nil || resp == nil || resp.GetUser() == nil {
		return "", false
	}
	username := strings.TrimSpace(resp.GetUser().GetUsername())
	if username == "" {
		return "", false
	}
	return username, true
}
