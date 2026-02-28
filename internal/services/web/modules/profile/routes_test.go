package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(newService(&routeGatewayStub{}, ""), module.Dependencies{}))
}

func TestRegisterRoutesProfileMethodContract(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newService(&routeGatewayStub{
		lookupResp: &socialv1.LookupUserProfileResponse{UserProfile: &socialv1.UserProfile{Username: "adventurer"}},
	}, ""), module.Dependencies{}))

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

type routeGatewayStub struct {
	lookupResp *socialv1.LookupUserProfileResponse
}

func (s *routeGatewayStub) LookupUserProfile(_ context.Context, _ *socialv1.LookupUserProfileRequest) (*socialv1.LookupUserProfileResponse, error) {
	if s.lookupResp != nil {
		return s.lookupResp, nil
	}
	return &socialv1.LookupUserProfileResponse{UserProfile: &socialv1.UserProfile{Username: "adventurer"}}, nil
}
