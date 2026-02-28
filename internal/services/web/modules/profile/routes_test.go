package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(newService(&routeGatewayStub{}, ""), publichandler.Base{}))
}

func TestRegisterRoutesProfileMethodContract(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newService(&routeGatewayStub{
		lookupResp: LookupUserProfileResponse{Username: "adventurer"},
	}, ""), publichandler.Base{}))

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
	lookupResp LookupUserProfileResponse
}

func (s *routeGatewayStub) LookupUserProfile(_ context.Context, _ LookupUserProfileRequest) (LookupUserProfileResponse, error) {
	if s.lookupResp.Username == "" {
		return LookupUserProfileResponse{Username: "adventurer"}, nil
	}
	return s.lookupResp, nil
}
