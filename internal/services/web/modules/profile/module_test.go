package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMountServesProfilePage(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &socialClientStub{lookupResp: &socialv1.LookupUserProfileResponse{UserProfile: &socialv1.UserProfile{
		UserId:        "user-1",
		Username:      "louis",
		Name:          "Louis",
		Pronouns:      "they/them",
		AvatarSetId:   "avatar_set_v1",
		AvatarAssetId: "001",
		Bio:           "Building Fracturing.Space.",
	}}}, "https://cdn.example.com/avatars", func(*http.Request) bool {
		return true
	})

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<title>louis | Fracturing.Space</title>`,
		`id="public-profile-page"`,
		`data-public-profile-username="louis"`,
		`data-public-profile-field="username">louis</dd>`,
		`data-public-profile-field="name">Louis</dd>`,
		`data-public-profile-field="pronouns">they/them</dd>`,
		`data-public-profile-field="bio"`,
		`Building Fracturing.Space.`,
		`data-public-profile-card="true"`,
		`lg:order-2`,
		`src="https://cdn.example.com/avatars/`,
		`href="/app/dashboard"`,
		`Back to dashboard`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
}

func TestMountServesHomeActionWhenViewerAnonymous(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &socialClientStub{lookupResp: &socialv1.LookupUserProfileResponse{UserProfile: &socialv1.UserProfile{
		UserId:   "user-1",
		Username: "louis",
	}}}, "", nil)

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `href="/"`) {
		t.Fatalf("body missing home action href: %q", body)
	}
	if !strings.Contains(body, `Back to home`) {
		t.Fatalf("body missing home action label: %q", body)
	}
	if strings.Contains(body, `href="/app/dashboard"`) {
		t.Fatalf("body unexpectedly contains dashboard action: %q", body)
	}
}

func TestMountServesProfileHead(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &socialClientStub{lookupResp: &socialv1.LookupUserProfileResponse{UserProfile: &socialv1.UserProfile{Username: "louis"}}}, "", nil)

	req := httptest.NewRequest(http.MethodHead, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMountReturnsNotFoundWhenUsernameMissing(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &socialClientStub{}, "", nil)

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfilePrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if !strings.Contains(rr.Body.String(), `id="app-error-state"`) {
		t.Fatalf("body missing app error marker: %q", rr.Body.String())
	}
}

func TestMountReturnsNotFoundWhenProfileLookupMisses(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &socialClientStub{lookupErr: status.Error(codes.NotFound, "username not found")}, "", nil)

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("unknown"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestMountReturnsNotFoundForNestedPath(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &socialClientStub{}, "", nil)

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("louis")+"/extra", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestMountReturnsServiceUnavailableWhenSocialServiceMissing(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, nil, "", nil)

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rr.Body.String(), `id="app-error-state"`) {
		t.Fatalf("body missing app error marker: %q", rr.Body.String())
	}
}

func TestMountRejectsProfileNonGet(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &socialClientStub{}, "", nil)
	req := httptest.NewRequest(http.MethodDelete, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestModuleIDReturnsProfile(t *testing.T) {
	t.Parallel()

	if got := New(nil, "", nil).ID(); got != "profile" {
		t.Fatalf("ID() = %q, want %q", got, "profile")
	}
}

func mountProfileModule(t *testing.T, socialClient SocialClient, assetBaseURL string, resolveSignedIn module.ResolveSignedIn) module.Mount {
	t.Helper()

	mount, err := New(socialClient, assetBaseURL, resolveSignedIn).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	return mount
}

type socialClientStub struct {
	lookupResp *socialv1.LookupUserProfileResponse
	lookupErr  error
}

func (s *socialClientStub) LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest, ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error) {
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	if s.lookupResp != nil {
		return s.lookupResp, nil
	}
	return &socialv1.LookupUserProfileResponse{}, nil
}

func (*socialClientStub) GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	return &socialv1.GetUserProfileResponse{}, nil
}

func (*socialClientStub) SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	return &socialv1.SetUserProfileResponse{}, nil
}
