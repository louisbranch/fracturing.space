package domain

import (
	"context"
	"encoding/json"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	gameintegration "github.com/louisbranch/fracturing.space/internal/services/game/integration"
	"github.com/louisbranch/fracturing.space/internal/services/shared/notificationpayload"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestInviteCreatedNotificationHandler_HandleCreatesRecipientIntent(t *testing.T) {
	notifications := &fakeNotificationClient{}
	handler := NewInviteCreatedNotificationHandler(
		&fakeInviteReader{invite: &gamev1.Invite{Id: "invite-1", CampaignId: "campaign-1", ParticipantId: "seat-1", RecipientUserId: "user-2", CreatedByParticipantId: "owner-1"}},
		&fakeCampaignReader{campaign: &gamev1.Campaign{Id: "campaign-1", Name: "Campaign Name"}},
		&fakeParticipantReader{participants: map[string]*gamev1.Participant{
			"campaign-1/seat-1":  {Id: "seat-1", CampaignId: "campaign-1", Name: "Seat Name"},
			"campaign-1/owner-1": {Id: "owner-1", CampaignId: "campaign-1", UserId: "user-1"},
		}},
		&fakeAuthReader{users: map[string]*authv1.User{"user-1": {Id: "user-1", Username: "owner"}}},
		notifications,
	)

	err := handler.Handle(context.Background(), outboxEventStub{payloadJSON: `{"invite_id":"invite-1","campaign_id":"campaign-1","recipient_user_id":"user-2"}`})
	if err != nil {
		t.Fatalf("handle created notification: %v", err)
	}
	if len(notifications.requests) != 1 {
		t.Fatalf("notification requests len = %d, want 1", len(notifications.requests))
	}
	req := notifications.requests[0]
	if req.GetRecipientUserId() != "user-2" {
		t.Fatalf("recipient user id = %q, want %q", req.GetRecipientUserId(), "user-2")
	}
	if req.GetMessageType() != gameintegration.InviteNotificationCreatedMessageType {
		t.Fatalf("message type = %q, want %q", req.GetMessageType(), gameintegration.InviteNotificationCreatedMessageType)
	}
	var payload notificationpayload.InAppPayload
	if err := json.Unmarshal([]byte(req.GetPayloadJson()), &payload); err != nil {
		t.Fatalf("unmarshal payload json: %v", err)
	}
	if payload.Title.Key != "notification.campaign_invite.created.title" {
		t.Fatalf("title key = %q, want %q", payload.Title.Key, "notification.campaign_invite.created.title")
	}
	if payload.Body.Key != "notification.campaign_invite.created.body_summary" {
		t.Fatalf("body key = %q, want inviter campaign summary key", payload.Body.Key)
	}
	if len(payload.Body.Args) != 2 || payload.Body.Args[0] != "owner" || payload.Body.Args[1] != "Campaign Name" {
		t.Fatalf("body args = %+v, want inviter+campaign args", payload.Body.Args)
	}
	if len(payload.Facts) != 3 {
		t.Fatalf("facts = %+v, want campaign, seat, inviter", payload.Facts)
	}
	if len(payload.Actions) != 1 {
		t.Fatalf("actions = %+v, want single view action", payload.Actions)
	}
	if payload.Actions[0].Label.Key != "notification.action.view_invitation" {
		t.Fatalf("action label key = %q, want %q", payload.Actions[0].Label.Key, "notification.action.view_invitation")
	}
	if payload.Actions[0].Kind != notificationpayload.ActionKindPublicInviteView || payload.Actions[0].TargetID != "invite-1" {
		t.Fatalf("action = %+v, want invite view action", payload.Actions[0])
	}
}

func TestInviteAcceptedNotificationHandler_HandleCreatesInviterIntent(t *testing.T) {
	notifications := &fakeNotificationClient{}
	handler := NewInviteAcceptedNotificationHandler(
		&fakeInviteReader{invite: &gamev1.Invite{Id: "invite-1", CampaignId: "campaign-1", ParticipantId: "seat-1", CreatedByParticipantId: "owner-1"}},
		&fakeCampaignReader{campaign: &gamev1.Campaign{Id: "campaign-1", Name: "Campaign Name"}},
		&fakeParticipantReader{participants: map[string]*gamev1.Participant{
			"campaign-1/seat-1":  {Id: "seat-1", CampaignId: "campaign-1", Name: "Seat Name"},
			"campaign-1/owner-1": {Id: "owner-1", CampaignId: "campaign-1", UserId: "user-1"},
		}},
		&fakeAuthReader{users: map[string]*authv1.User{"user-2": {Id: "user-2", Username: "recipient"}}},
		notifications,
	)

	err := handler.Handle(context.Background(), outboxEventStub{payloadJSON: `{"invite_id":"invite-1","campaign_id":"campaign-1","recipient_user_id":"user-2"}`})
	if err != nil {
		t.Fatalf("handle accepted notification: %v", err)
	}
	if len(notifications.requests) != 1 {
		t.Fatalf("notification requests len = %d, want 1", len(notifications.requests))
	}
	req := notifications.requests[0]
	if req.GetRecipientUserId() != "user-1" {
		t.Fatalf("recipient user id = %q, want %q", req.GetRecipientUserId(), "user-1")
	}
	if req.GetMessageType() != gameintegration.InviteNotificationAcceptedMessageType {
		t.Fatalf("message type = %q, want %q", req.GetMessageType(), gameintegration.InviteNotificationAcceptedMessageType)
	}
	var payload notificationpayload.InAppPayload
	if err := json.Unmarshal([]byte(req.GetPayloadJson()), &payload); err != nil {
		t.Fatalf("unmarshal payload json: %v", err)
	}
	if payload.Title.Key != "notification.campaign_invite.accepted.title" {
		t.Fatalf("title key = %q, want %q", payload.Title.Key, "notification.campaign_invite.accepted.title")
	}
	if len(payload.Actions) != 1 || payload.Actions[0].Kind != notificationpayload.ActionKindAppCampaignOpen || payload.Actions[0].TargetID != "campaign-1" {
		t.Fatalf("actions = %+v, want open campaign action", payload.Actions)
	}
}

func TestInviteDeclinedNotificationHandler_HandleCreatesInviterIntent(t *testing.T) {
	notifications := &fakeNotificationClient{}
	handler := NewInviteDeclinedNotificationHandler(
		&fakeInviteReader{invite: &gamev1.Invite{Id: "invite-1", CampaignId: "campaign-1", ParticipantId: "seat-1", CreatedByParticipantId: "owner-1"}},
		&fakeCampaignReader{campaign: &gamev1.Campaign{Id: "campaign-1", Name: "Campaign Name"}},
		&fakeParticipantReader{participants: map[string]*gamev1.Participant{
			"campaign-1/seat-1":  {Id: "seat-1", CampaignId: "campaign-1", Name: "Seat Name"},
			"campaign-1/owner-1": {Id: "owner-1", CampaignId: "campaign-1", UserId: "user-1"},
		}},
		&fakeAuthReader{users: map[string]*authv1.User{"user-2": {Id: "user-2", Username: "recipient"}}},
		notifications,
	)

	err := handler.Handle(context.Background(), outboxEventStub{payloadJSON: `{"invite_id":"invite-1","campaign_id":"campaign-1","recipient_user_id":"user-2"}`})
	if err != nil {
		t.Fatalf("handle declined notification: %v", err)
	}
	if len(notifications.requests) != 1 {
		t.Fatalf("notification requests len = %d, want 1", len(notifications.requests))
	}
	req := notifications.requests[0]
	if req.GetRecipientUserId() != "user-1" {
		t.Fatalf("recipient user id = %q, want %q", req.GetRecipientUserId(), "user-1")
	}
	if req.GetMessageType() != gameintegration.InviteNotificationDeclinedMessageType {
		t.Fatalf("message type = %q, want %q", req.GetMessageType(), gameintegration.InviteNotificationDeclinedMessageType)
	}
	var payload notificationpayload.InAppPayload
	if err := json.Unmarshal([]byte(req.GetPayloadJson()), &payload); err != nil {
		t.Fatalf("unmarshal payload json: %v", err)
	}
	if payload.Title.Key != "notification.campaign_invite.declined.title" {
		t.Fatalf("title key = %q, want %q", payload.Title.Key, "notification.campaign_invite.declined.title")
	}
	if payload.Body.Key != "notification.campaign_invite.declined.body_summary" {
		t.Fatalf("body key = %q, want %q", payload.Body.Key, "notification.campaign_invite.declined.body_summary")
	}
	if len(payload.Body.Args) != 3 || payload.Body.Args[0] != "recipient" || payload.Body.Args[1] != "Seat Name" || payload.Body.Args[2] != "Campaign Name" {
		t.Fatalf("body args = %+v, want recipient+seat+campaign args", payload.Body.Args)
	}
	if len(payload.Actions) != 1 || payload.Actions[0].Kind != notificationpayload.ActionKindAppCampaignOpen || payload.Actions[0].TargetID != "campaign-1" {
		t.Fatalf("actions = %+v, want open campaign action", payload.Actions)
	}
}

func TestInviteAcceptedNotificationHandler_HandleRetriesOnCreatorLookupFailure(t *testing.T) {
	notifications := &fakeNotificationClient{}
	handler := NewInviteAcceptedNotificationHandler(
		&fakeInviteReader{invite: &gamev1.Invite{Id: "invite-1", CampaignId: "campaign-1", ParticipantId: "seat-1", CreatedByParticipantId: "owner-1"}},
		&fakeCampaignReader{campaign: &gamev1.Campaign{Id: "campaign-1", Name: "Campaign Name"}},
		&fakeParticipantReader{
			participants: map[string]*gamev1.Participant{
				"campaign-1/seat-1": {Id: "seat-1", CampaignId: "campaign-1", Name: "Seat Name"},
			},
			errByKey: map[string]error{
				"campaign-1/owner-1": status.Error(codes.Unavailable, "projection lag"),
			},
		},
		&fakeAuthReader{users: map[string]*authv1.User{"user-2": {Id: "user-2", Username: "recipient"}}},
		notifications,
	)

	err := handler.Handle(context.Background(), outboxEventStub{payloadJSON: `{"invite_id":"invite-1","campaign_id":"campaign-1","recipient_user_id":"user-2"}`})
	if err == nil {
		t.Fatal("expected transient creator lookup error")
	}
	if len(notifications.requests) != 0 {
		t.Fatalf("notification requests len = %d, want 0", len(notifications.requests))
	}
}

type outboxEventStub struct {
	payloadJSON string
}

func (e outboxEventStub) GetId() string          { return "evt-1" }
func (e outboxEventStub) GetEventType() string   { return "invite.test" }
func (e outboxEventStub) GetPayloadJson() string { return e.payloadJSON }
func (e outboxEventStub) GetAttemptCount() int32 { return 0 }

type fakeInviteReader struct {
	invite *gamev1.Invite
}

func (f *fakeInviteReader) GetInvite(context.Context, *gamev1.GetInviteRequest, ...grpc.CallOption) (*gamev1.GetInviteResponse, error) {
	return &gamev1.GetInviteResponse{Invite: f.invite}, nil
}

type fakeCampaignReader struct {
	campaign *gamev1.Campaign
}

func (f *fakeCampaignReader) GetCampaign(context.Context, *gamev1.GetCampaignRequest, ...grpc.CallOption) (*gamev1.GetCampaignResponse, error) {
	return &gamev1.GetCampaignResponse{Campaign: f.campaign}, nil
}

type fakeParticipantReader struct {
	participants map[string]*gamev1.Participant
	errByKey     map[string]error
}

func (f *fakeParticipantReader) GetParticipant(_ context.Context, in *gamev1.GetParticipantRequest, _ ...grpc.CallOption) (*gamev1.GetParticipantResponse, error) {
	key := in.GetCampaignId() + "/" + in.GetParticipantId()
	if err := f.errByKey[key]; err != nil {
		return nil, err
	}
	return &gamev1.GetParticipantResponse{Participant: f.participants[key]}, nil
}

type fakeAuthReader struct {
	users map[string]*authv1.User
}

func (f *fakeAuthReader) GetUser(_ context.Context, in *authv1.GetUserRequest, _ ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	return &authv1.GetUserResponse{User: f.users[in.GetUserId()]}, nil
}

type fakeNotificationClient struct {
	requests []*notificationsv1.CreateNotificationIntentRequest
}

func (f *fakeNotificationClient) CreateNotificationIntent(_ context.Context, in *notificationsv1.CreateNotificationIntentRequest, _ ...grpc.CallOption) (*notificationsv1.CreateNotificationIntentResponse, error) {
	f.requests = append(f.requests, in)
	return &notificationsv1.CreateNotificationIntentResponse{}, nil
}
