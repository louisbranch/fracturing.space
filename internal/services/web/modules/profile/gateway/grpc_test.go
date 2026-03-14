package gateway

import (
	"context"
	"net/http"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewGRPCGatewayFailsClosedWhenAuthClientMissing(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCGateway(nil, nil)
	_, err := gateway.LookupUserProfile(context.Background(), profileapp.LookupUserProfileRequest{Username: "louis"})
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestGRPCGatewayLookupMapsFields(t *testing.T) {
	t.Parallel()

	authClient := &authClientStub{resp: &authv1.LookupUserByUsernameResponse{
		User: &authv1.User{Id: "  user-1  ", Username: "  louis  "},
	}}
	socialClient := &socialClientStub{resp: &socialv1.GetUserProfileResponse{
		UserProfile: &socialv1.UserProfile{
			UserId:        "  user-1  ",
			Name:          "  Louis Branch  ",
			Pronouns:      pronouns.ToProto("they/them"),
			Bio:           "  Explorer  ",
			AvatarSetId:   "  set-v1  ",
			AvatarAssetId: "  001  ",
		},
	}}
	gateway := GRPCGateway{AuthClient: authClient, SocialClient: socialClient}

	resp, err := gateway.LookupUserProfile(context.Background(), profileapp.LookupUserProfileRequest{Username: "  louis  "})
	if err != nil {
		t.Fatalf("LookupUserProfile() error = %v", err)
	}
	if authClient.lastReq.GetUsername() != "louis" {
		t.Fatalf("request username = %q, want %q", authClient.lastReq.GetUsername(), "louis")
	}
	if socialClient.lastReq.GetUserId() != "user-1" {
		t.Fatalf("social request user id = %q, want %q", socialClient.lastReq.GetUserId(), "user-1")
	}
	if resp.UserID != "user-1" || resp.Username != "louis" || resp.Name != "Louis Branch" || resp.Pronouns != "they/them" || resp.Bio != "Explorer" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.SocialProfileStatus != profileapp.SocialProfileStatusLoaded {
		t.Fatalf("SocialProfileStatus = %q, want %q", resp.SocialProfileStatus, profileapp.SocialProfileStatusLoaded)
	}
}

func TestGRPCGatewayLookupReturnsBaselineWhenSocialProfileMissing(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{
		AuthClient: &authClientStub{resp: &authv1.LookupUserByUsernameResponse{
			User: &authv1.User{Id: "user-1", Username: "louis"},
		}},
		SocialClient: &socialClientStub{err: status.Error(codes.NotFound, "profile not found")},
	}
	resp, err := gateway.LookupUserProfile(context.Background(), profileapp.LookupUserProfileRequest{Username: "louis"})
	if err != nil {
		t.Fatalf("LookupUserProfile() error = %v", err)
	}
	if resp.UserID != "user-1" || resp.Username != "louis" {
		t.Fatalf("unexpected baseline response: %+v", resp)
	}
	if resp.SocialProfileStatus != profileapp.SocialProfileStatusMissing {
		t.Fatalf("SocialProfileStatus = %q, want %q", resp.SocialProfileStatus, profileapp.SocialProfileStatusMissing)
	}
}

func TestGRPCGatewayLookupReturnsBaselineWhenSocialServiceUnavailable(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{
		AuthClient: &authClientStub{resp: &authv1.LookupUserByUsernameResponse{
			User: &authv1.User{Id: "user-1", Username: "louis"},
		}},
		SocialClient: &socialClientStub{err: status.Error(codes.Unavailable, "social unavailable")},
	}
	resp, err := gateway.LookupUserProfile(context.Background(), profileapp.LookupUserProfileRequest{Username: "louis"})
	if err != nil {
		t.Fatalf("LookupUserProfile() error = %v", err)
	}
	if resp.UserID != "user-1" || resp.Username != "louis" {
		t.Fatalf("unexpected baseline response: %+v", resp)
	}
	if resp.SocialProfileStatus != profileapp.SocialProfileStatusUnavailable {
		t.Fatalf("SocialProfileStatus = %q, want %q", resp.SocialProfileStatus, profileapp.SocialProfileStatusUnavailable)
	}
}

func TestGRPCGatewayLookupReturnsBaselineWhenSocialClientUnconfigured(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCGateway(&authClientStub{resp: &authv1.LookupUserByUsernameResponse{
		User: &authv1.User{Id: "user-1", Username: "louis"},
	}}, nil)
	resp, err := gateway.LookupUserProfile(context.Background(), profileapp.LookupUserProfileRequest{Username: "louis"})
	if err != nil {
		t.Fatalf("LookupUserProfile() error = %v", err)
	}
	if resp.SocialProfileStatus != profileapp.SocialProfileStatusUnconfigured {
		t.Fatalf("SocialProfileStatus = %q, want %q", resp.SocialProfileStatus, profileapp.SocialProfileStatusUnconfigured)
	}
}

func TestGRPCGatewayLookupMapsNotFoundError(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{AuthClient: &authClientStub{err: status.Error(codes.NotFound, "user not found")}}
	_, err := gateway.LookupUserProfile(context.Background(), profileapp.LookupUserProfileRequest{Username: "missing"})
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

type authClientStub struct {
	resp    *authv1.LookupUserByUsernameResponse
	err     error
	lastReq *authv1.LookupUserByUsernameRequest
}

func (s *authClientStub) LookupUserByUsername(_ context.Context, req *authv1.LookupUserByUsernameRequest, _ ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	if s.resp != nil {
		return s.resp, nil
	}
	return &authv1.LookupUserByUsernameResponse{}, nil
}

type socialClientStub struct {
	resp    *socialv1.GetUserProfileResponse
	err     error
	lastReq *socialv1.GetUserProfileRequest
}

func (s *socialClientStub) GetUserProfile(_ context.Context, req *socialv1.GetUserProfileRequest, _ ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	if s.resp != nil {
		return s.resp, nil
	}
	return &socialv1.GetUserProfileResponse{}, nil
}
