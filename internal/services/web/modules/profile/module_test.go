package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMountServesProfilePage(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &authClientStub{lookupResp: &authv1.LookupUserByUsernameResponse{User: &authv1.User{
		Id:       "user-1",
		Username: "louis",
	}}}, &socialClientStub{getResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{
		UserId:        "user-1",
		Name:          "Louis",
		Pronouns:      sharedpronouns.ToProto("they/them"),
		AvatarSetId:   "avatar_set_v1",
		AvatarAssetId: "apothecary_journeyman",
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

	mount := mountProfileModule(t, &authClientStub{lookupResp: &authv1.LookupUserByUsernameResponse{User: &authv1.User{
		Id:       "user-1",
		Username: "louis",
	}}}, &socialClientStub{}, "", nil)

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

	mount := mountProfileModule(t, &authClientStub{lookupResp: &authv1.LookupUserByUsernameResponse{User: &authv1.User{Username: "louis"}}}, &socialClientStub{}, "", nil)

	req := httptest.NewRequest(http.MethodHead, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMountReturnsNotFoundWhenProfileLookupMisses(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, &authClientStub{lookupErr: status.Error(codes.NotFound, "username not found")}, &socialClientStub{}, "", nil)

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("unknown"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestMountReturnsServiceUnavailableWhenAuthServiceMissing(t *testing.T) {
	t.Parallel()

	mount := mountProfileModule(t, nil, nil, "", nil)

	req := httptest.NewRequest(http.MethodGet, routepath.UserProfile("louis"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestModuleHealthyReflectsGatewayState(t *testing.T) {
	t.Parallel()

	if New(Config{}).Healthy() {
		t.Fatalf("New(Config{}).Healthy() = true, want false")
	}
	if !New(Config{Gateway: profilegateway.NewGRPCGateway(&authClientStub{}, &socialClientStub{})}).Healthy() {
		t.Fatalf("expected configured gateway to be healthy")
	}
}

func mountProfileModule(t *testing.T, authClient AuthClient, socialClient SocialClient, assetBaseURL string, resolveSignedIn module.ResolveSignedIn) module.Mount {
	t.Helper()

	mount, err := New(Config{
		Gateway:         profilegateway.NewGRPCGateway(authClient, socialClient),
		AssetBaseURL:    assetBaseURL,
		ResolveSignedIn: resolveSignedIn,
	}).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	return mount
}

type authClientStub struct {
	lookupResp *authv1.LookupUserByUsernameResponse
	lookupErr  error
}

func (s *authClientStub) LookupUserByUsername(context.Context, *authv1.LookupUserByUsernameRequest, ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error) {
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	if s.lookupResp != nil {
		return s.lookupResp, nil
	}
	return &authv1.LookupUserByUsernameResponse{}, nil
}

type socialClientStub struct {
	getResp *socialv1.GetUserProfileResponse
	getErr  error
}

func (s *socialClientStub) GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getResp != nil {
		return s.getResp, nil
	}
	return &socialv1.GetUserProfileResponse{}, nil
}
