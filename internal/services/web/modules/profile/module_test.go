package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestMountServesProfilePage(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &moduleGatewayStub{lookupResp: profileapp.LookupUserProfileResponse{
		UserID:        "user-1",
		Username:      "louis",
		Name:          "Louis",
		Pronouns:      "they/them",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "apothecary_journeyman",
		Bio:           "Building Fracturing.Space.",
	}}, "https://cdn.example.com/avatars", func(*http.Request) bool {
		return true
	})

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-public-profile-username="louis"`,
		`data-public-profile-field="name">Louis</dd>`,
		`data-public-profile-field="pronouns">they/them</dd>`,
		`Building Fracturing.Space.`,
		`href="/app/dashboard"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
}

func TestMountServesHomeActionWhenViewerAnonymous(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &moduleGatewayStub{lookupResp: profileapp.LookupUserProfileResponse{
		UserID:   "user-1",
		Username: "louis",
	}}, "", nil)

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `href="/"`) || !strings.Contains(body, `Back to home`) {
		t.Fatalf("body missing home action: %q", body)
	}
}

func TestMountServesProfileHead(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &moduleGatewayStub{lookupResp: profileapp.LookupUserProfileResponse{Username: "louis"}}, "", nil)

	req := httptest.NewRequest(http.MethodHead, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMountReturnsNotFoundWhenProfileLookupMisses(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &moduleGatewayStub{}, "", nil)

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("unknown"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestMountReturnsServiceUnavailableWhenAuthServiceMissing(t *testing.T) {
	t.Parallel()

	mount, err := New(Config{}).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func mountProfileModule(
	t *testing.T,
	gateway profileapp.Gateway,
	assetBaseURL string,
	resolveSignedIn func(*http.Request) bool,
) module.Mount {
	t.Helper()

	mount, err := New(Config{
		Service:      profileapp.NewService(gateway),
		AssetBaseURL: assetBaseURL,
		Principal: principal.NewPrincipal(
			nil,
			resolveSignedIn,
			nil,
			nil,
			nil,
		),
	}).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	return mount
}

type moduleGatewayStub struct {
	lookupResp profileapp.LookupUserProfileResponse
	lookupErr  error
}

func (s *moduleGatewayStub) LookupUserProfile(_ context.Context, _ profileapp.LookupUserProfileRequest) (profileapp.LookupUserProfileResponse, error) {
	if s.lookupErr != nil {
		return profileapp.LookupUserProfileResponse{}, s.lookupErr
	}
	return s.lookupResp, nil
}
