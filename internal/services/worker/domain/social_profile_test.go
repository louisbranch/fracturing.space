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

func TestSignupSocialProfileHandler_HandleCreatesMissingProfile(t *testing.T) {
	social := &fakeSocialProfileClient{
		getErr: status.Error(codes.NotFound, "missing profile"),
	}
	handler := NewSignupSocialProfileHandler(social)

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{"user_id":"user-1","signup_method":"passkey"}`,
	})
	if err != nil {
		t.Fatalf("handle signup social profile: %v", err)
	}
	if social.lastSetReq == nil {
		t.Fatal("expected set user profile request")
	}
	if social.lastSetReq.GetUserId() != "user-1" {
		t.Fatalf("set user id = %q, want %q", social.lastSetReq.GetUserId(), "user-1")
	}
}

func TestSignupSocialProfileHandler_HandleNoOpWhenProfileExists(t *testing.T) {
	social := &fakeSocialProfileClient{
		getResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{UserId: "user-1", Username: "existing"}},
	}
	handler := NewSignupSocialProfileHandler(social)

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{"user_id":"user-1"}`,
	})
	if err != nil {
		t.Fatalf("handle signup social profile: %v", err)
	}
	if social.lastSetReq != nil {
		t.Fatalf("expected no set user profile call when profile exists")
	}
}

func TestSignupSocialProfileHandler_MissingUserIDPermanent(t *testing.T) {
	handler := NewSignupSocialProfileHandler(&fakeSocialProfileClient{})

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{}`,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestSignupSocialProfileHandler_InvalidArgumentPermanent(t *testing.T) {
	social := &fakeSocialProfileClient{
		getErr: status.Error(codes.NotFound, "missing profile"),
		setErr: status.Error(codes.InvalidArgument, "bad request"),
	}
	handler := NewSignupSocialProfileHandler(social)

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

func TestSignupSocialProfileHandler_UnavailableRetryable(t *testing.T) {
	social := &fakeSocialProfileClient{
		getErr: status.Error(codes.NotFound, "missing profile"),
		setErr: status.Error(codes.Unavailable, "social unavailable"),
	}
	handler := NewSignupSocialProfileHandler(social)

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{"user_id":"user-1"}`,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if IsPermanent(err) {
		t.Fatalf("expected retryable error, got permanent: %v", err)
	}
}

type fakeSocialProfileClient struct {
	getResp    *socialv1.GetUserProfileResponse
	getErr     error
	setResp    *socialv1.SetUserProfileResponse
	setErr     error
	lastSetReq *socialv1.SetUserProfileRequest
}

func (f *fakeSocialProfileClient) GetUserProfile(_ context.Context, _ *socialv1.GetUserProfileRequest, _ ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &socialv1.GetUserProfileResponse{}, nil
}

func (f *fakeSocialProfileClient) SetUserProfile(_ context.Context, req *socialv1.SetUserProfileRequest, _ ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	f.lastSetReq = req
	if f.setErr != nil {
		return nil, f.setErr
	}
	if f.setResp != nil {
		return f.setResp, nil
	}
	return &socialv1.SetUserProfileResponse{}, nil
}
