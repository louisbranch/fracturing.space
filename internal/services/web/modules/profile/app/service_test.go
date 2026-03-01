package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestNewServiceFailsClosedWhenGatewayMissing(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, "")
	_, err := svc.LoadProfile(context.Background(), "louis")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestLoadProfileRequiresUsername(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeGateway{}, "")
	_, err := svc.LoadProfile(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestLoadProfileReturnsNotFoundWhenGatewayMisses(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeGateway{lookupResp: LookupUserProfileResponse{Username: "   "}}, "")
	_, err := svc.LoadProfile(context.Background(), "louis")
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestLoadProfileNormalizesFieldsAndBuildsAvatarURL(t *testing.T) {
	t.Parallel()

	gateway := &fakeGateway{lookupResp: LookupUserProfileResponse{
		Username:      "  louis  ",
		UserID:        "   ",
		Name:          "  Louis Branch  ",
		Pronouns:      "  they/them  ",
		Bio:           "  Explorer  ",
		AvatarSetID:   "  set-v1  ",
		AvatarAssetID: "  001  ",
	}}
	svc := NewService(gateway, "https://cdn.example.com")

	profile, err := svc.LoadProfile(context.Background(), "  louis  ")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if gateway.lastReq.Username != "louis" {
		t.Fatalf("gateway username = %q, want %q", gateway.lastReq.Username, "louis")
	}
	if profile.Username != "louis" {
		t.Fatalf("Username = %q, want %q", profile.Username, "louis")
	}
	if profile.Name != "Louis Branch" {
		t.Fatalf("Name = %q, want %q", profile.Name, "Louis Branch")
	}
	if profile.Pronouns != "they/them" {
		t.Fatalf("Pronouns = %q, want %q", profile.Pronouns, "they/them")
	}
	if profile.Bio != "Explorer" {
		t.Fatalf("Bio = %q, want %q", profile.Bio, "Explorer")
	}
	if !strings.HasPrefix(profile.AvatarURL, "https://cdn.example.com/") {
		t.Fatalf("AvatarURL = %q, want asset base URL prefix", profile.AvatarURL)
	}
}

func TestLoadProfilePropagatesGatewayError(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeGateway{lookupErr: errors.New("boom")}, "")
	_, err := svc.LoadProfile(context.Background(), "louis")
	if err == nil {
		t.Fatalf("expected gateway error")
	}
	if err.Error() != "boom" {
		t.Fatalf("err = %q, want %q", err.Error(), "boom")
	}
}

type fakeGateway struct {
	lookupResp LookupUserProfileResponse
	lookupErr  error
	lastReq    LookupUserProfileRequest
}

func (f *fakeGateway) LookupUserProfile(_ context.Context, req LookupUserProfileRequest) (LookupUserProfileResponse, error) {
	f.lastReq = req
	if f.lookupErr != nil {
		return LookupUserProfileResponse{}, f.lookupErr
	}
	return f.lookupResp, nil
}
