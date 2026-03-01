package gateway

import (
	"context"
	"net/http"
	"testing"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewGRPCGatewayFailsClosedWhenClientMissing(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCGateway(nil)
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

	client := &socialClientStub{resp: &socialv1.LookupUserProfileResponse{
		UserProfile: &socialv1.UserProfile{
			UserId:        "  user-1  ",
			Username:      "  louis  ",
			Name:          "  Louis Branch  ",
			Pronouns:      pronouns.ToProto("they/them"),
			Bio:           "  Explorer  ",
			AvatarSetId:   "  set-v1  ",
			AvatarAssetId: "  001  ",
		},
	}}
	gateway := GRPCGateway{Client: client}

	resp, err := gateway.LookupUserProfile(context.Background(), profileapp.LookupUserProfileRequest{Username: "  louis  "})
	if err != nil {
		t.Fatalf("LookupUserProfile() error = %v", err)
	}
	if client.lastReq.GetUsername() != "louis" {
		t.Fatalf("request username = %q, want %q", client.lastReq.GetUsername(), "louis")
	}
	if resp.UserID != "user-1" {
		t.Fatalf("UserID = %q, want %q", resp.UserID, "user-1")
	}
	if resp.Username != "louis" {
		t.Fatalf("Username = %q, want %q", resp.Username, "louis")
	}
	if resp.Name != "Louis Branch" {
		t.Fatalf("Name = %q, want %q", resp.Name, "Louis Branch")
	}
	if resp.Pronouns != "they/them" {
		t.Fatalf("Pronouns = %q, want %q", resp.Pronouns, "they/them")
	}
	if resp.Bio != "Explorer" {
		t.Fatalf("Bio = %q, want %q", resp.Bio, "Explorer")
	}
}

func TestGRPCGatewayLookupReturnsEmptyWhenPayloadMissing(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{Client: &socialClientStub{resp: &socialv1.LookupUserProfileResponse{}}}
	resp, err := gateway.LookupUserProfile(context.Background(), profileapp.LookupUserProfileRequest{Username: "louis"})
	if err != nil {
		t.Fatalf("LookupUserProfile() error = %v", err)
	}
	if resp != (profileapp.LookupUserProfileResponse{}) {
		t.Fatalf("response = %#v, want empty response", resp)
	}
}

func TestGRPCGatewayLookupMapsNotFoundError(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{Client: &socialClientStub{err: status.Error(codes.NotFound, "profile not found")}}
	_, err := gateway.LookupUserProfile(context.Background(), profileapp.LookupUserProfileRequest{Username: "missing"})
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

type socialClientStub struct {
	resp    *socialv1.LookupUserProfileResponse
	err     error
	lastReq *socialv1.LookupUserProfileRequest
}

func (s *socialClientStub) LookupUserProfile(_ context.Context, req *socialv1.LookupUserProfileRequest, _ ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	if s.resp != nil {
		return s.resp, nil
	}
	return &socialv1.LookupUserProfileResponse{}, nil
}
