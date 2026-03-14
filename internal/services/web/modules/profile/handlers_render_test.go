package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
)

func TestHandleProfileRendersPublicProfileFromServiceResult(t *testing.T) {
	t.Parallel()

	service := &handlerServiceStub{
		profile: profileapp.Profile{
			Username:      "louis",
			UserID:        "user-1",
			Name:          "Louis Branch",
			Pronouns:      "they/them",
			Bio:           "Building Fracturing.Space.",
			AvatarSetID:   "avatar_set_v1",
			AvatarAssetID: "apothecary_journeyman",
		},
	}
	h := newHandlers(service, "https://cdn.example.com/avatars", publichandler.NewBase(publichandler.WithResolveViewerSignedIn(func(*http.Request) bool {
		return true
	})))

	req := httptest.NewRequest(http.MethodGet, "/u/louis", nil)
	rr := httptest.NewRecorder()
	h.handleProfile(rr, req, "louis")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if service.lastUsername != "louis" {
		t.Fatalf("service username = %q, want %q", service.lastUsername, "louis")
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-public-profile-username="louis"`,
		`data-public-profile-field="name">Louis Branch</dd>`,
		`data-public-profile-field="pronouns">they/them</dd>`,
		`Building Fracturing.Space.`,
		`https://cdn.example.com/avatars`,
		`href="/app/dashboard"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
}

func TestHandleProfileWritesServiceError(t *testing.T) {
	t.Parallel()

	h := newHandlers(&handlerServiceStub{
		err: apperrors.E(apperrors.KindUnavailable, "profile service is unavailable"),
	}, "", publichandler.NewBase())

	req := httptest.NewRequest(http.MethodGet, "/u/louis", nil)
	rr := httptest.NewRecorder()
	h.handleProfile(rr, req, "louis")

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

type handlerServiceStub struct {
	profile      profileapp.Profile
	err          error
	lastUsername string
}

func (s *handlerServiceStub) LoadProfile(_ context.Context, username string) (profileapp.Profile, error) {
	s.lastUsername = username
	if s.err != nil {
		return profileapp.Profile{}, s.err
	}
	return s.profile, nil
}
