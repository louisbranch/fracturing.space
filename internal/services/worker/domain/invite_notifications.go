package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	gameintegration "github.com/louisbranch/fracturing.space/internal/services/game/integration"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/notificationpayload"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type workerInviteClient interface {
	GetInvite(ctx context.Context, in *gamev1.GetInviteRequest, opts ...grpc.CallOption) (*gamev1.GetInviteResponse, error)
}

type workerCampaignClient interface {
	GetCampaign(ctx context.Context, in *gamev1.GetCampaignRequest, opts ...grpc.CallOption) (*gamev1.GetCampaignResponse, error)
}

type workerParticipantClient interface {
	GetParticipant(ctx context.Context, in *gamev1.GetParticipantRequest, opts ...grpc.CallOption) (*gamev1.GetParticipantResponse, error)
}

type workerAuthClient interface {
	GetUser(ctx context.Context, in *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error)
}

type workerNotificationClient interface {
	CreateNotificationIntent(ctx context.Context, in *notificationsv1.CreateNotificationIntentRequest, opts ...grpc.CallOption) (*notificationsv1.CreateNotificationIntentResponse, error)
}

// InviteCreatedNotificationHandler sends targeted invite-created notifications.
type InviteCreatedNotificationHandler struct {
	invites       workerInviteClient
	campaigns     workerCampaignClient
	participants  workerParticipantClient
	auth          workerAuthClient
	notifications workerNotificationClient
}

// NewInviteCreatedNotificationHandler creates the targeted invite-created handler.
func NewInviteCreatedNotificationHandler(
	invites workerInviteClient,
	campaigns workerCampaignClient,
	participants workerParticipantClient,
	auth workerAuthClient,
	notifications workerNotificationClient,
) *InviteCreatedNotificationHandler {
	return &InviteCreatedNotificationHandler{
		invites:       invites,
		campaigns:     campaigns,
		participants:  participants,
		auth:          auth,
		notifications: notifications,
	}
}

// Handle loads invite context and creates a recipient notification intent.
func (h *InviteCreatedNotificationHandler) Handle(ctx context.Context, event OutboxEvent) error {
	if h == nil || h.invites == nil || h.campaigns == nil || h.participants == nil || h.notifications == nil {
		return Permanent(fmt.Errorf("invite created notification dependencies are not configured"))
	}

	payload, err := decodeInviteCreatedPayload(event)
	if err != nil {
		return Permanent(err)
	}
	recipientUserID := strings.TrimSpace(payload.RecipientUserID)
	if recipientUserID == "" {
		return nil
	}

	invitation, campaignRecord, participantRecord, err := h.loadInviteContext(ctx, payload.InviteID)
	if err != nil {
		return err
	}
	inviterUsername, _ := h.lookupParticipantUsername(ctx, invitation.GetCampaignId(), invitation.GetCreatedByParticipantId())
	return h.createNotification(
		ctx,
		recipientUserID,
		gameintegration.InviteNotificationCreatedMessageType,
		inviteCreatedMessagePayload(
			invitation.GetId(),
			campaignRecord.GetName(),
			participantRecord.GetName(),
			inviterUsername,
		),
		gameintegration.InviteCreatedNotificationDedupeKey(invitation.GetId()),
	)
}

type InviteOutcomeNotificationHandler struct {
	invites       workerInviteClient
	campaigns     workerCampaignClient
	participants  workerParticipantClient
	auth          workerAuthClient
	notifications workerNotificationClient
	messageType   string
	dedupeSuffix  string
}

// NewInviteAcceptedNotificationHandler creates the invite-accepted handler.
func NewInviteAcceptedNotificationHandler(
	invites workerInviteClient,
	campaigns workerCampaignClient,
	participants workerParticipantClient,
	auth workerAuthClient,
	notifications workerNotificationClient,
) *InviteOutcomeNotificationHandler {
	return newInviteOutcomeNotificationHandler(invites, campaigns, participants, auth, notifications, gameintegration.InviteNotificationAcceptedMessageType, "accepted")
}

// NewInviteDeclinedNotificationHandler creates the invite-declined handler.
func NewInviteDeclinedNotificationHandler(
	invites workerInviteClient,
	campaigns workerCampaignClient,
	participants workerParticipantClient,
	auth workerAuthClient,
	notifications workerNotificationClient,
) *InviteOutcomeNotificationHandler {
	return newInviteOutcomeNotificationHandler(invites, campaigns, participants, auth, notifications, gameintegration.InviteNotificationDeclinedMessageType, "declined")
}

func newInviteOutcomeNotificationHandler(
	invites workerInviteClient,
	campaigns workerCampaignClient,
	participants workerParticipantClient,
	auth workerAuthClient,
	notifications workerNotificationClient,
	messageType string,
	dedupeSuffix string,
) *InviteOutcomeNotificationHandler {
	return &InviteOutcomeNotificationHandler{
		invites:       invites,
		campaigns:     campaigns,
		participants:  participants,
		auth:          auth,
		notifications: notifications,
		messageType:   messageType,
		dedupeSuffix:  dedupeSuffix,
	}
}

// Handle loads invite context and notifies the inviter about invite consumption.
func (h *InviteOutcomeNotificationHandler) Handle(ctx context.Context, event OutboxEvent) error {
	if h == nil || h.invites == nil || h.campaigns == nil || h.participants == nil || h.notifications == nil {
		return Permanent(fmt.Errorf("invite outcome notification dependencies are not configured"))
	}

	payload, err := decodeInviteOutcomePayload(event)
	if err != nil {
		return Permanent(err)
	}
	recipientUserID := strings.TrimSpace(payload.RecipientUserID)
	if recipientUserID == "" {
		return nil
	}

	invitation, campaignRecord, participantRecord, err := h.loadInviteContext(ctx, payload.InviteID)
	if err != nil {
		return err
	}
	creatorUserID, err := loadParticipantUserID(ctx, h.participants, invitation.GetCampaignId(), invitation.GetCreatedByParticipantId())
	if err != nil {
		return err
	}
	if creatorUserID == "" {
		return nil
	}
	if creatorUserID == recipientUserID {
		return nil
	}
	recipientUsername, _ := h.lookupUsername(ctx, recipientUserID)
	dedupeKey := gameintegration.InviteAcceptedNotificationDedupeKey(invitation.GetId())
	if h.dedupeSuffix == "declined" {
		dedupeKey = gameintegration.InviteDeclinedNotificationDedupeKey(invitation.GetId())
	}
	return h.createNotification(
		ctx,
		creatorUserID,
		h.messageType,
		inviteOutcomeMessagePayload(h.messageType, invitation.GetCampaignId(), campaignRecord.GetName(), participantRecord.GetName(), recipientUsername),
		dedupeKey,
	)
}

func (h *InviteCreatedNotificationHandler) loadInviteContext(ctx context.Context, inviteID string) (*gamev1.Invite, *gamev1.Campaign, *gamev1.Participant, error) {
	return loadInviteNotificationContext(ctx, h.invites, h.campaigns, h.participants, inviteID)
}

func (h *InviteOutcomeNotificationHandler) loadInviteContext(ctx context.Context, inviteID string) (*gamev1.Invite, *gamev1.Campaign, *gamev1.Participant, error) {
	return loadInviteNotificationContext(ctx, h.invites, h.campaigns, h.participants, inviteID)
}

func loadInviteNotificationContext(
	ctx context.Context,
	invites workerInviteClient,
	campaigns workerCampaignClient,
	participants workerParticipantClient,
	inviteID string,
) (*gamev1.Invite, *gamev1.Campaign, *gamev1.Participant, error) {
	callCtx := workerGameReadContext(ctx)
	inviteResp, err := invites.GetInvite(callCtx, &gamev1.GetInviteRequest{InviteId: inviteID})
	if err != nil {
		return nil, nil, nil, classifyGameReadError("load invite", err)
	}
	if inviteResp == nil || inviteResp.GetInvite() == nil {
		return nil, nil, nil, fmt.Errorf("load invite: empty response")
	}
	invitation := inviteResp.GetInvite()

	campaignResp, err := campaigns.GetCampaign(callCtx, &gamev1.GetCampaignRequest{CampaignId: invitation.GetCampaignId()})
	if err != nil {
		return nil, nil, nil, classifyGameReadError("load campaign", err)
	}
	if campaignResp == nil || campaignResp.GetCampaign() == nil {
		return nil, nil, nil, fmt.Errorf("load campaign: empty response")
	}

	participantResp, err := participants.GetParticipant(callCtx, &gamev1.GetParticipantRequest{
		CampaignId:    invitation.GetCampaignId(),
		ParticipantId: invitation.GetParticipantId(),
	})
	if err != nil {
		return nil, nil, nil, classifyGameReadError("load participant", err)
	}
	if participantResp == nil || participantResp.GetParticipant() == nil {
		return nil, nil, nil, fmt.Errorf("load participant: empty response")
	}
	return invitation, campaignResp.GetCampaign(), participantResp.GetParticipant(), nil
}

func (h *InviteCreatedNotificationHandler) lookupParticipantUsername(ctx context.Context, campaignID, participantID string) (string, bool) {
	return lookupParticipantUsername(ctx, h.participants, h.auth, campaignID, participantID)
}

func (h *InviteOutcomeNotificationHandler) lookupUsername(ctx context.Context, userID string) (string, bool) {
	return lookupUsername(ctx, h.auth, userID)
}

func lookupParticipantUsername(ctx context.Context, participants workerParticipantClient, auth workerAuthClient, campaignID, participantID string) (string, bool) {
	userID, err := loadParticipantUserID(ctx, participants, campaignID, participantID)
	if err != nil || userID == "" {
		return "", false
	}
	return lookupUsername(ctx, auth, userID)
}

func loadParticipantUserID(ctx context.Context, participants workerParticipantClient, campaignID, participantID string) (string, error) {
	campaignID = strings.TrimSpace(campaignID)
	participantID = strings.TrimSpace(participantID)
	if campaignID == "" || participantID == "" || participants == nil {
		return "", nil
	}
	resp, err := participants.GetParticipant(workerGameReadContext(ctx), &gamev1.GetParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		return "", classifyGameReadError("load participant user binding", err)
	}
	if resp == nil || resp.GetParticipant() == nil {
		return "", fmt.Errorf("load participant user binding: empty response")
	}
	userID := strings.TrimSpace(resp.GetParticipant().GetUserId())
	if userID == "" {
		return "", nil
	}
	return userID, nil
}

func lookupUsername(ctx context.Context, auth workerAuthClient, userID string) (string, bool) {
	if auth == nil {
		return "", false
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", false
	}
	resp, err := auth.GetUser(serviceContext(ctx), &authv1.GetUserRequest{UserId: userID})
	if err != nil || resp == nil || resp.GetUser() == nil {
		return "", false
	}
	username := strings.TrimSpace(resp.GetUser().GetUsername())
	if username == "" {
		return "", false
	}
	return username, true
}

func (h *InviteCreatedNotificationHandler) createNotification(ctx context.Context, recipientUserID, messageType string, payload notificationpayload.InAppPayload, dedupeKey string) error {
	return createNotification(ctx, h.notifications, recipientUserID, messageType, payload, dedupeKey)
}

func (h *InviteOutcomeNotificationHandler) createNotification(ctx context.Context, recipientUserID, messageType string, payload notificationpayload.InAppPayload, dedupeKey string) error {
	return createNotification(ctx, h.notifications, recipientUserID, messageType, payload, dedupeKey)
}

func createNotification(ctx context.Context, notifications workerNotificationClient, recipientUserID, messageType string, payload notificationpayload.InAppPayload, dedupeKey string) error {
	recipientUserID = strings.TrimSpace(recipientUserID)
	messageType = strings.TrimSpace(messageType)
	dedupeKey = strings.TrimSpace(dedupeKey)
	if notifications == nil {
		return Permanent(fmt.Errorf("notification client is not configured"))
	}
	if recipientUserID == "" || messageType == "" || dedupeKey == "" {
		return nil
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return Permanent(fmt.Errorf("marshal notification payload: %w", err))
	}
	_, err = notifications.CreateNotificationIntent(serviceContext(ctx), &notificationsv1.CreateNotificationIntentRequest{
		RecipientUserId: recipientUserID,
		MessageType:     messageType,
		PayloadJson:     string(payloadJSON),
		DedupeKey:       dedupeKey,
		Source:          notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
	})
	if err == nil {
		return nil
	}
	if isPermanentNotificationError(err) {
		return Permanent(err)
	}
	return err
}

func inviteCreatedMessagePayload(inviteID, campaignName, participantName, inviterUsername string) notificationpayload.InAppPayload {
	body := "Review this invitation to respond."
	if inviterUsername = strings.TrimSpace(inviterUsername); inviterUsername != "" && strings.TrimSpace(campaignName) != "" {
		body = "@" + inviterUsername + " invited you to join " + strings.TrimSpace(campaignName) + "."
	}
	return notificationpayload.InAppPayload{
		Title:   "Campaign invitation",
		Body:    body,
		Facts:   inviteFacts(campaignName, participantName, inviterUsername),
		Actions: []notificationpayload.PayloadAction{notificationpayload.ViewInvitationAction(inviteID)},
	}
}

func inviteOutcomeMessagePayload(messageType, campaignID, campaignName, participantName, recipientUsername string) notificationpayload.InAppPayload {
	recipient := strings.TrimSpace(recipientUsername)
	body := "An invitation was updated."
	title := "Invitation update"
	switch strings.TrimSpace(messageType) {
	case gameintegration.InviteNotificationAcceptedMessageType:
		title = "Invitation accepted"
		if recipient != "" && participantName != "" && campaignName != "" {
			body = "@" + recipient + " accepted " + participantName + " in " + campaignName + "."
		} else {
			body = "A campaign invitation was accepted."
		}
	case gameintegration.InviteNotificationDeclinedMessageType:
		title = "Invitation declined"
		if recipient != "" && participantName != "" && campaignName != "" {
			body = "@" + recipient + " declined to participate as " + participantName + " in " + campaignName + "."
		} else {
			body = "A campaign invitation was declined."
		}
	}
	return notificationpayload.InAppPayload{
		Title:   title,
		Body:    body,
		Facts:   inviteFacts(campaignName, participantName, ""),
		Actions: []notificationpayload.PayloadAction{notificationpayload.OpenCampaignAction(campaignID)},
	}
}

func inviteFacts(campaignName, participantName, inviterUsername string) []notificationpayload.PayloadFact {
	facts := make([]notificationpayload.PayloadFact, 0, 3)
	if campaignName = strings.TrimSpace(campaignName); campaignName != "" {
		facts = append(facts, notificationpayload.PayloadFact{Label: "Campaign", Value: campaignName})
	}
	if participantName = strings.TrimSpace(participantName); participantName != "" {
		facts = append(facts, notificationpayload.PayloadFact{Label: "Seat", Value: participantName})
	}
	if inviterUsername = strings.TrimSpace(inviterUsername); inviterUsername != "" {
		facts = append(facts, notificationpayload.PayloadFact{Label: "Invited by", Value: "@" + inviterUsername})
	}
	return facts
}

func decodeInviteCreatedPayload(event OutboxEvent) (gameintegration.InviteNotificationOutboxPayload, error) {
	if event == nil {
		return gameintegration.InviteNotificationOutboxPayload{}, fmt.Errorf("event is required")
	}
	var payload gameintegration.InviteNotificationOutboxPayload
	if err := json.Unmarshal([]byte(event.GetPayloadJson()), &payload); err != nil {
		return gameintegration.InviteNotificationOutboxPayload{}, fmt.Errorf("decode invite created payload: %w", err)
	}
	payload.InviteID = strings.TrimSpace(payload.InviteID)
	payload.CampaignID = strings.TrimSpace(payload.CampaignID)
	payload.RecipientUserID = strings.TrimSpace(payload.RecipientUserID)
	if payload.InviteID == "" {
		return gameintegration.InviteNotificationOutboxPayload{}, fmt.Errorf("invite_id is required in invite created payload")
	}
	if payload.CampaignID == "" {
		return gameintegration.InviteNotificationOutboxPayload{}, fmt.Errorf("campaign_id is required in invite created payload")
	}
	return payload, nil
}

func decodeInviteOutcomePayload(event OutboxEvent) (gameintegration.InviteNotificationOutboxPayload, error) {
	if event == nil {
		return gameintegration.InviteNotificationOutboxPayload{}, fmt.Errorf("event is required")
	}
	var payload gameintegration.InviteNotificationOutboxPayload
	if err := json.Unmarshal([]byte(event.GetPayloadJson()), &payload); err != nil {
		return gameintegration.InviteNotificationOutboxPayload{}, fmt.Errorf("decode invite outcome payload: %w", err)
	}
	payload.InviteID = strings.TrimSpace(payload.InviteID)
	payload.CampaignID = strings.TrimSpace(payload.CampaignID)
	payload.RecipientUserID = strings.TrimSpace(payload.RecipientUserID)
	if payload.InviteID == "" {
		return gameintegration.InviteNotificationOutboxPayload{}, fmt.Errorf("invite_id is required in invite outcome payload")
	}
	if payload.CampaignID == "" {
		return gameintegration.InviteNotificationOutboxPayload{}, fmt.Errorf("campaign_id is required in invite outcome payload")
	}
	return payload, nil
}

func workerGameReadContext(ctx context.Context) context.Context {
	return grpcauthctx.WithAdminOverride(serviceContext(ctx), "worker invite notifications")
}

func serviceContext(ctx context.Context) context.Context {
	return grpcauthctx.WithServiceID(ctx, serviceaddr.ServiceWorker)
}

func classifyGameReadError(operation string, err error) error {
	if isPermanentGameReadError(err) {
		return Permanent(fmt.Errorf("%s: %w", operation, err))
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func isPermanentGameReadError(err error) bool {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition:
		return true
	default:
		return false
	}
}

func isPermanentNotificationError(err error) bool {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition:
		return true
	default:
		return false
	}
}
