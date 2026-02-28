package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(newService(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "Campaign"}}}), module.Dependencies{}))
}

func TestRegisterRoutesCampaignsPathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newService(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "Campaign"}}, participants: []CampaignParticipant{{ID: "p-manager", UserID: "user-123", CampaignAccess: "Manager"}}}), module.Dependencies{ResolveUserID: func(*http.Request) string { return "user-123" }}))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
		wantLoc    string
	}{
		{name: "campaigns root", method: http.MethodGet, path: routepath.AppCampaigns, wantStatus: http.StatusOK},
		{name: "campaigns slash root", method: http.MethodGet, path: routepath.CampaignsPrefix, wantStatus: http.StatusOK},
		{name: "campaign new get", method: http.MethodGet, path: routepath.AppCampaignsNew, wantStatus: http.StatusOK},
		{name: "campaign create get", method: http.MethodGet, path: routepath.AppCampaignsCreate, wantStatus: http.StatusOK},
		{name: "campaign overview head", method: http.MethodHead, path: routepath.AppCampaign("c1"), wantStatus: http.StatusOK},
		{name: "campaign overview post rejected", method: http.MethodPost, path: routepath.AppCampaign("c1"), wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodGet + ", HEAD"},
		{name: "campaign session start post", method: http.MethodPost, path: routepath.AppCampaignSessionStart("c1"), wantStatus: http.StatusFound, wantLoc: routepath.AppCampaignSessions("c1")},
		{name: "campaign unknown subpath", method: http.MethodGet, path: routepath.AppCampaign("c1") + "/unknown", wantStatus: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantAllow != "" {
				if got := rr.Header().Get("Allow"); got != tc.wantAllow {
					t.Fatalf("Allow = %q, want %q", got, tc.wantAllow)
				}
			}
			if tc.wantLoc != "" {
				if got := rr.Header().Get("Location"); got != tc.wantLoc {
					t.Fatalf("Location = %q, want %q", got, tc.wantLoc)
				}
			}
		})
	}
}

func TestRegisterStableRoutesExposeWorkflowRoutesAndHideScaffoldedRoutes(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerStableRoutes(mux, newHandlers(newService(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "Campaign"}},
		characterCreationProgress: CampaignCharacterCreationProgress{
			Steps:    []CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
			NextStep: 1,
		},
	}), module.Dependencies{ResolveUserID: func(*http.Request) string { return "user-123" }}))

	for _, path := range []string{
		routepath.AppCampaignSessionStart("c1"),
		routepath.AppCampaignSessionEnd("c1"),
		routepath.AppCampaignParticipantUpdate("c1"),
		routepath.AppCampaignCharacterUpdate("c1"),
		routepath.AppCampaignCharacterControl("c1"),
		routepath.AppCampaignInviteCreate("c1"),
		routepath.AppCampaignInviteRevoke("c1"),
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusNotFound)
		}
	}

	for _, path := range []string{
		routepath.AppCampaign("c1"),
		routepath.AppCampaignParticipants("c1"),
		routepath.AppCampaignCharacters("c1"),
		routepath.AppCampaignCharacter("c1", "char-1"),
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
	}

	for _, path := range []string{
		routepath.AppCampaignSessions("c1"),
		routepath.AppCampaignSession("c1", "sess-1"),
		routepath.AppCampaignInvites("c1"),
		routepath.AppCampaignGame("c1"),
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusNotFound)
		}
	}

	stepReq := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationStep("c1", "char-1"), strings.NewReader("class_id=warrior&subclass_id=guardian"))
	stepReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stepRR := httptest.NewRecorder()
	mux.ServeHTTP(stepRR, stepReq)
	if stepRR.Code != http.StatusFound {
		t.Fatalf("step status = %d, want %d", stepRR.Code, http.StatusFound)
	}
	if got := stepRR.Header().Get("Location"); got != routepath.AppCampaignCharacter("c1", "char-1") {
		t.Fatalf("step location = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-1"))
	}

	resetReq := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationReset("c1", "char-1"), nil)
	resetRR := httptest.NewRecorder()
	mux.ServeHTTP(resetRR, resetReq)
	if resetRR.Code != http.StatusFound {
		t.Fatalf("reset status = %d, want %d", resetRR.Code, http.StatusFound)
	}
	if got := resetRR.Header().Get("Location"); got != routepath.AppCampaignCharacter("c1", "char-1") {
		t.Fatalf("reset location = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-1"))
	}

	createReq := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&kind=pc"))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createRR := httptest.NewRecorder()
	mux.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusFound {
		t.Fatalf("create status = %d, want %d", createRR.Code, http.StatusFound)
	}
	if got := createRR.Header().Get("Location"); got != routepath.AppCampaignCharacter("c1", "char-created") {
		t.Fatalf("create location = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-created"))
	}
}

func TestRegisterRoutesCharacterCreationWorkflowEndpoints(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newService(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "Campaign"}},
		characterCreationProgress: CampaignCharacterCreationProgress{
			Steps:    []CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
			NextStep: 1,
		},
	}), module.Dependencies{ResolveUserID: func(*http.Request) string { return "user-123" }}))

	stepReq := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationStep("c1", "char-1"), strings.NewReader("class_id=warrior&subclass_id=guardian"))
	stepReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stepRR := httptest.NewRecorder()
	mux.ServeHTTP(stepRR, stepReq)
	if stepRR.Code != http.StatusFound {
		t.Fatalf("step status = %d, want %d", stepRR.Code, http.StatusFound)
	}
	if got := stepRR.Header().Get("Location"); got != routepath.AppCampaignCharacter("c1", "char-1") {
		t.Fatalf("step location = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-1"))
	}

	resetReq := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationReset("c1", "char-1"), nil)
	resetRR := httptest.NewRecorder()
	mux.ServeHTTP(resetRR, resetReq)
	if resetRR.Code != http.StatusFound {
		t.Fatalf("reset status = %d, want %d", resetRR.Code, http.StatusFound)
	}
	if got := resetRR.Header().Get("Location"); got != routepath.AppCampaignCharacter("c1", "char-1") {
		t.Fatalf("reset location = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-1"))
	}
}
