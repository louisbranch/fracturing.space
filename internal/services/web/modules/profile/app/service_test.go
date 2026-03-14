package app

import (
	"context"
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestNewServiceFailsClosedWhenGatewayMissing(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)
	_, err := svc.LoadProfile(context.Background(), "louis")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestUnavailableGatewayFailsClosed(t *testing.T) {
	t.Parallel()

	gateway := NewUnavailableGateway()
	if IsGatewayHealthy(nil) {
		t.Fatalf("IsGatewayHealthy(nil) = true, want false")
	}
	if IsGatewayHealthy(gateway) {
		t.Fatalf("IsGatewayHealthy(unavailable) = true, want false")
	}
	if !IsGatewayHealthy(&fakeGateway{}) {
		t.Fatalf("IsGatewayHealthy(stub) = false, want true")
	}

	resp, err := gateway.LookupUserProfile(context.Background(), LookupUserProfileRequest{Username: "louis"})
	if err == nil {
		t.Fatalf("LookupUserProfile() error = nil, want unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("LookupUserProfile() status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if resp != (LookupUserProfileResponse{}) {
		t.Fatalf("LookupUserProfile() resp = %+v, want zero value", resp)
	}
}

func TestLoadProfileRequiresUsername(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeGateway{})
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

	svc := NewService(&fakeGateway{lookupResp: LookupUserProfileResponse{Username: "   "}})
	_, err := svc.LoadProfile(context.Background(), "louis")
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestLoadProfileNormalizesFieldsAndKeepsAvatarIdentity(t *testing.T) {
	t.Parallel()

	gateway := &fakeGateway{lookupResp: LookupUserProfileResponse{
		Username:            "  louis  ",
		UserID:              "   ",
		Name:                "  Louis Branch  ",
		Pronouns:            "  they/them  ",
		Bio:                 "  Explorer  ",
		AvatarSetID:         "  set-v1  ",
		AvatarAssetID:       "  001  ",
		SocialProfileStatus: SocialProfileStatusLoaded,
	}}
	svc := NewService(gateway)

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
	if profile.UserID != "" {
		t.Fatalf("UserID = %q, want empty string", profile.UserID)
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
	if profile.AvatarSetID != "set-v1" {
		t.Fatalf("AvatarSetID = %q, want %q", profile.AvatarSetID, "set-v1")
	}
	if profile.AvatarAssetID != "001" {
		t.Fatalf("AvatarAssetID = %q, want %q", profile.AvatarAssetID, "001")
	}
	if profile.SocialProfileStatus != SocialProfileStatusLoaded {
		t.Fatalf("SocialProfileStatus = %q, want %q", profile.SocialProfileStatus, SocialProfileStatusLoaded)
	}
}

func TestLoadProfileKeepsExplicitSocialFallbackStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status SocialProfileStatus
	}{
		{name: "missing", status: SocialProfileStatusMissing},
		{name: "unavailable", status: SocialProfileStatusUnavailable},
		{name: "unconfigured", status: SocialProfileStatusUnconfigured},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := NewService(&fakeGateway{lookupResp: LookupUserProfileResponse{
				Username:            "louis",
				UserID:              "user-1",
				SocialProfileStatus: tc.status,
			}})

			profile, err := svc.LoadProfile(context.Background(), "louis")
			if err != nil {
				t.Fatalf("LoadProfile() error = %v", err)
			}
			if profile.SocialProfileStatus != tc.status {
				t.Fatalf("SocialProfileStatus = %q, want %q", profile.SocialProfileStatus, tc.status)
			}
		})
	}
}

func TestLoadProfilePropagatesGatewayError(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeGateway{lookupErr: errors.New("boom")})
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
