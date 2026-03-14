package domain

import (
	"context"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSignupSocialDirectoryHandler_HandleSyncsDirectoryUser(t *testing.T) {
	social := &fakeSocialDirectoryClient{}
	handler := NewSignupSocialDirectoryHandler(social)

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{"user_id":"user-1","username":"alice","signup_method":"passkey"}`,
	})
	if err != nil {
		t.Fatalf("handle signup social directory: %v", err)
	}
	if social.lastReq == nil {
		t.Fatal("expected sync directory request")
	}
	if social.lastReq.GetUserId() != "user-1" || social.lastReq.GetUsername() != "alice" {
		t.Fatalf("sync request = %+v, want user-1/alice", social.lastReq)
	}
}

func TestSignupSocialDirectoryHandler_MissingUsernamePermanent(t *testing.T) {
	handler := NewSignupSocialDirectoryHandler(&fakeSocialDirectoryClient{})

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{"user_id":"user-1"}`,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestSignupSocialDirectoryHandler_InvalidArgumentPermanent(t *testing.T) {
	social := &fakeSocialDirectoryClient{
		err: status.Error(codes.InvalidArgument, "bad request"),
	}
	handler := NewSignupSocialDirectoryHandler(social)

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{"user_id":"user-1","username":"alice"}`,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

type fakeSocialDirectoryClient struct {
	lastReq *socialv1.SyncDirectoryUserRequest
	err     error
}

func (f *fakeSocialDirectoryClient) SyncDirectoryUser(_ context.Context, req *socialv1.SyncDirectoryUserRequest, _ ...grpc.CallOption) (*socialv1.SyncDirectoryUserResponse, error) {
	f.lastReq = req
	if f.err != nil {
		return nil, f.err
	}
	return &socialv1.SyncDirectoryUserResponse{}, nil
}
