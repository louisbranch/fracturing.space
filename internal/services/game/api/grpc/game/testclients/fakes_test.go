package testclients

import (
	"context"
	"errors"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
)

func TestFakeAuthClientGetUserReturnsConfiguredUserAndTracksRequest(t *testing.T) {
	client := &FakeAuthClient{
		User: &authv1.User{Id: "user-1", Username: "alice"},
	}
	req := &authv1.GetUserRequest{UserId: "user-1"}

	resp, err := client.GetUser(context.Background(), req)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if client.LastGetUserRequest != req {
		t.Fatal("expected FakeAuthClient to retain last request pointer")
	}
	if resp.GetUser().GetId() != "user-1" {
		t.Fatalf("GetUser().User.Id = %q, want %q", resp.GetUser().GetId(), "user-1")
	}
	if resp.GetUser().GetUsername() != "alice" {
		t.Fatalf("GetUser().User.Username = %q, want %q", resp.GetUser().GetUsername(), "alice")
	}
}

func TestFakeAuthClientGetUserReturnsConfiguredError(t *testing.T) {
	wantErr := errors.New("auth unavailable")
	client := &FakeAuthClient{GetUserErr: wantErr}

	resp, err := client.GetUser(context.Background(), &authv1.GetUserRequest{UserId: "user-1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetUser() error = %v, want %v", err, wantErr)
	}
	if resp != nil {
		t.Fatalf("GetUser() response = %v, want nil on error", resp)
	}
}

func TestFakeSocialClientGetUserProfileReturnsConfiguredProfileAndTracksRequest(t *testing.T) {
	client := &FakeSocialClient{
		Profile: &socialv1.UserProfile{Name: "Alice"},
	}
	req := &socialv1.GetUserProfileRequest{UserId: "user-1"}

	resp, err := client.GetUserProfile(context.Background(), req)
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}
	if client.GetUserProfileCalls != 1 {
		t.Fatalf("GetUserProfileCalls = %d, want 1", client.GetUserProfileCalls)
	}
	if client.LastGetUserProfileReq != req {
		t.Fatal("expected FakeSocialClient to retain last request pointer")
	}
	if resp.GetUserProfile().GetName() != "Alice" {
		t.Fatalf("GetUserProfile().UserProfile.Name = %q, want %q", resp.GetUserProfile().GetName(), "Alice")
	}
}

func TestFakeSocialClientGetUserProfileReturnsEmptyResponseWithoutProfile(t *testing.T) {
	client := &FakeSocialClient{}

	resp, err := client.GetUserProfile(context.Background(), &socialv1.GetUserProfileRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}
	if client.GetUserProfileCalls != 1 {
		t.Fatalf("GetUserProfileCalls = %d, want 1", client.GetUserProfileCalls)
	}
	if resp.GetUserProfile() != nil {
		t.Fatalf("GetUserProfile().UserProfile = %v, want nil when no profile configured", resp.GetUserProfile())
	}
}

func TestFakeSocialClientGetUserProfileReturnsConfiguredError(t *testing.T) {
	wantErr := errors.New("social unavailable")
	client := &FakeSocialClient{GetUserProfileErr: wantErr}

	resp, err := client.GetUserProfile(context.Background(), &socialv1.GetUserProfileRequest{UserId: "user-1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetUserProfile() error = %v, want %v", err, wantErr)
	}
	if resp != nil {
		t.Fatalf("GetUserProfile() response = %v, want nil on error", resp)
	}
	if client.GetUserProfileCalls != 1 {
		t.Fatalf("GetUserProfileCalls = %d, want 1", client.GetUserProfileCalls)
	}
}
