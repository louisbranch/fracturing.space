package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(profileapp.NewService(&routeGatewayStub{}), "", publichandler.Base{}))
}

func TestRegisterRoutesProfileMethodContract(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(profileapp.NewService(&routeGatewayStub{
		lookupResp: profileapp.LookupUserProfileResponse{Username: "adventurer"},
	}), "", publichandler.Base{}))

	getReq := httptest.NewRequest(http.MethodGet, routepath.UserProfile("adventurer"), nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", getRR.Code, http.StatusOK)
	}

	headReq := httptest.NewRequest(http.MethodHead, routepath.UserProfile("adventurer"), nil)
	headRR := httptest.NewRecorder()
	mux.ServeHTTP(headRR, headReq)
	if headRR.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", headRR.Code, http.StatusOK)
	}

	postReq := httptest.NewRequest(http.MethodPost, routepath.UserProfile("adventurer"), nil)
	postRR := httptest.NewRecorder()
	mux.ServeHTTP(postRR, postReq)
	if postRR.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", postRR.Code, http.StatusMethodNotAllowed)
	}
	if got := postRR.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("Allow = %q, want %q", got, "GET, HEAD")
	}

	nestedReq := httptest.NewRequest(http.MethodGet, routepath.UserProfile("adventurer")+"/details", nil)
	nestedRR := httptest.NewRecorder()
	mux.ServeHTTP(nestedRR, nestedReq)
	if nestedRR.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", nestedRR.Code, http.StatusNotFound)
	}
}

func TestWithUsernameReturnsNotFoundForMissingPathValue(t *testing.T) {
	t.Parallel()

	h := newHandlers(profileapp.NewService(&routeGatewayStub{}), "", publichandler.Base{})
	called := false
	handler := h.withUsername(func(http.ResponseWriter, *http.Request, string) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Fatal("expected delegate not to be called")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestWithUsernameDelegatesResolvedValue(t *testing.T) {
	t.Parallel()

	h := newHandlers(profileapp.NewService(&routeGatewayStub{}), "", publichandler.Base{})
	called := false
	var got string
	handler := h.withUsername(func(_ http.ResponseWriter, _ *http.Request, username string) {
		called = true
		got = username
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("username", " adventurer ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected delegate to be called")
	}
	if got != "adventurer" {
		t.Fatalf("username = %q, want %q", got, "adventurer")
	}
}

type routeGatewayStub struct {
	lookupResp profileapp.LookupUserProfileResponse
}

func (s *routeGatewayStub) LookupUserProfile(_ context.Context, _ profileapp.LookupUserProfileRequest) (profileapp.LookupUserProfileResponse, error) {
	if s.lookupResp.Username == "" {
		return profileapp.LookupUserProfileResponse{Username: "adventurer"}, nil
	}
	return s.lookupResp, nil
}
