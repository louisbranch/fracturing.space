package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc/metadata"
)

func TestMountCharacterCreateUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := New(Config{Gateway: managerMutationGateway(), Base: managerMutationBase(), ChatFallbackPort: "", Workflows: nil})
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&kind=pc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignCharacter("c1", "char-created") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-created"))
	}
}

func TestMountCharacterCreateRedirectsForNonHTMX(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: managerMutationGateway(), Base: managerMutationBase(), ChatFallbackPort: "", Workflows: nil})
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&kind=pc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacter("c1", "char-created") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-created"))
	}
}

func TestMountCharacterCreateRejectsInvalidKind(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: managerMutationGateway(), Base: managerMutationBase(), ChatFallbackPort: "", Workflows: nil})
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&kind=invalid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestStableMutationRoutesReturnParseErrorLocalizationKeys(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: managerMutationGateway(), Base: managerMutationBase(), ChatFallbackPort: "", Workflows: nil})
	mount, _ := m.Mount()

	tests := []struct {
		name        string
		path        string
		wantMarkerA string
		wantMarkerB string
	}{
		{
			name:        "campaign update parse error",
			path:        routepath.AppCampaignEdit("c1"),
			wantMarkerA: "error.web.message.failed_to_parse_campaign_update_form",
			wantMarkerB: "failed to parse campaign update form",
		},
		{
			name:        "session start parse error",
			path:        routepath.AppCampaignSessionStart("c1"),
			wantMarkerA: "error.web.message.failed_to_parse_session_start_form",
			wantMarkerB: "failed to parse session start form",
		},
		{
			name:        "session end parse error",
			path:        routepath.AppCampaignSessionEnd("c1"),
			wantMarkerA: "error.web.message.failed_to_parse_session_end_form",
			wantMarkerB: "failed to parse session end form",
		},
		{
			name:        "invite create parse error",
			path:        routepath.AppCampaignInviteCreate("c1"),
			wantMarkerA: "error.web.message.failed_to_parse_invite_create_form",
			wantMarkerB: "failed to parse invite create form",
		},
		{
			name:        "invite revoke parse error",
			path:        routepath.AppCampaignInviteRevoke("c1"),
			wantMarkerA: "error.web.message.failed_to_parse_invite_revoke_form",
			wantMarkerB: "failed to parse invite revoke form",
		},
		{
			name:        "participant update parse error",
			path:        routepath.AppCampaignParticipantEdit("c1", "p-manager"),
			wantMarkerA: "error.web.message.failed_to_parse_participant_update_form",
			wantMarkerB: "failed to parse participant update form",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader("bad=%zz"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
			}
			body := rr.Body.String()
			if !strings.Contains(body, tc.wantMarkerA) && !strings.Contains(body, tc.wantMarkerB) {
				t.Fatalf("body missing parse error marker %q or %q: %q", tc.wantMarkerA, tc.wantMarkerB, body)
			}
		})
	}
}

func TestStableMutationRoutesReturnRequiredFieldLocalizationKeys(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: managerMutationGateway(), Base: managerMutationBase(), ChatFallbackPort: "", Workflows: nil})
	mount, _ := m.Mount()

	tests := []struct {
		name        string
		path        string
		body        string
		wantMarkerA string
		wantMarkerB string
	}{
		{
			name:        "session end missing session id",
			path:        routepath.AppCampaignSessionEnd("c1"),
			body:        "session_id=   ",
			wantMarkerA: "error.web.message.session_id_is_required",
			wantMarkerB: "session id is required",
		},
		{
			name:        "invite create missing participant id",
			path:        routepath.AppCampaignInviteCreate("c1"),
			body:        "participant_id=   &recipient_user_id=user-2",
			wantMarkerA: "error.web.message.participant_id_is_required",
			wantMarkerB: "participant id is required",
		},
		{
			name:        "invite revoke missing invite id",
			path:        routepath.AppCampaignInviteRevoke("c1"),
			body:        "invite_id=   ",
			wantMarkerA: "error.web.message.invite_id_is_required",
			wantMarkerB: "invite id is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
			}
			body := rr.Body.String()
			if !strings.Contains(body, tc.wantMarkerA) && !strings.Contains(body, tc.wantMarkerB) {
				t.Fatalf("body missing validation marker %q or %q: %q", tc.wantMarkerA, tc.wantMarkerB, body)
			}
		})
	}
}

func TestStableMutationRoutesRedirectWithHTMXParity(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: managerMutationGateway(), Base: managerMutationBase(), ChatFallbackPort: "", Workflows: nil})
	mount, _ := m.Mount()

	tests := []struct {
		name         string
		path         string
		body         string
		wantLocation string
	}{
		{
			name:         "campaign update",
			path:         routepath.AppCampaignEdit("c1"),
			body:         "name=Campaign+One&theme_prompt=Updated+theme&locale=en-US",
			wantLocation: routepath.AppCampaign("c1"),
		},
		{
			name:         "session start",
			path:         routepath.AppCampaignSessionStart("c1"),
			body:         "name=Session+Two",
			wantLocation: routepath.AppCampaignSessions("c1"),
		},
		{
			name:         "session end",
			path:         routepath.AppCampaignSessionEnd("c1"),
			body:         "session_id=sess-1",
			wantLocation: routepath.AppCampaignSessions("c1"),
		},
		{
			name:         "invite create",
			path:         routepath.AppCampaignInviteCreate("c1"),
			body:         "participant_id=p-1&recipient_user_id=user-123",
			wantLocation: routepath.AppCampaignInvites("c1"),
		},
		{
			name:         "invite revoke",
			path:         routepath.AppCampaignInviteRevoke("c1"),
			body:         "invite_id=inv-1",
			wantLocation: routepath.AppCampaignInvites("c1"),
		},
		{
			name:         "participant update",
			path:         routepath.AppCampaignParticipantEdit("c1", "p-manager"),
			body:         "name=Manager+One&role=player&pronouns=they%2Fthem",
			wantLocation: routepath.AppCampaignParticipants("c1"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name+" browser", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
			}
			if got := rr.Header().Get("Location"); got != tc.wantLocation {
				t.Fatalf("Location = %q, want %q", got, tc.wantLocation)
			}
		})

		t.Run(tc.name+" htmx", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
			}
			if got := rr.Header().Get("HX-Redirect"); got != tc.wantLocation {
				t.Fatalf("HX-Redirect = %q, want %q", got, tc.wantLocation)
			}
		})
	}
}

func TestCampaignAIBindingRouteRedirectsBackToParticipantEdit(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:           true,
			Allowed:             true,
			ActorCampaignAccess: "Owner",
		},
	}, Base: managerMutationBase(), ChatFallbackPort: "", Workflows: nil})
	mount, _ := m.Mount()

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignAIBinding("c1"), strings.NewReader("participant_id=p-ai&ai_agent_id=agent-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignParticipantEdit("c1", "p-ai") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignParticipantEdit("c1", "p-ai"))
	}
}

func TestParticipantUpdateRouteValidatesRoleAndAccess(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: managerMutationGateway(), Base: managerMutationBase(), ChatFallbackPort: "", Workflows: nil})
	mount, _ := m.Mount()

	tests := []struct {
		name        string
		body        string
		wantMarkerA string
		wantMarkerB string
	}{
		{
			name:        "invalid role",
			body:        "name=Manager+One&role=invalid&pronouns=they%2Fthem",
			wantMarkerA: "error.web.message.participant_role_value_is_invalid",
			wantMarkerB: "participant role value is invalid",
		},
		{
			name:        "invalid access",
			body:        "name=Manager+One&role=player&campaign_access=invalid",
			wantMarkerA: "error.web.message.campaign_access_value_is_invalid",
			wantMarkerB: "campaign access value is invalid",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignParticipantEdit("c1", "p-manager"), strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
			}
			body := rr.Body.String()
			if !strings.Contains(body, tc.wantMarkerA) && !strings.Contains(body, tc.wantMarkerB) {
				t.Fatalf("body missing validation marker %q or %q: %q", tc.wantMarkerA, tc.wantMarkerB, body)
			}
		})
	}
}

func TestParticipantUpdateRouteRejectsAIInvariantTampering(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		participant: CampaignParticipant{
			ID:             "p-ai",
			Name:           "Caretaker",
			Role:           "GM",
			CampaignAccess: "Member",
			Controller:     "AI",
			Pronouns:       "it/its",
		},
		authorizationDecision: campaignapp.AuthorizationDecision{Evaluated: true, Allowed: true},
	}, Base: managerMutationBase(), ChatFallbackPort: "", Workflows: nil})
	mount, _ := m.Mount()

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignParticipantEdit("c1", "p-ai"), strings.NewReader("name=Caretaker&role=player&campaign_access=member&pronouns=it%2Fits"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "error.web.message.participant_ai_role_and_access_are_fixed") &&
		!strings.Contains(body, "AI participants must remain GM and Member") {
		t.Fatalf("body missing AI invariant marker: %q", body)
	}
}

func TestCampaignUpdateRouteValidatesLocale(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: managerMutationGateway(), Base: managerMutationBase(), ChatFallbackPort: "", Workflows: nil})
	mount, _ := m.Mount()

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignEdit("c1"), strings.NewReader("name=Campaign+One&theme_prompt=Theme&locale=es-ES"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "error.web.message.campaign_locale_value_is_invalid") && !strings.Contains(body, "campaign locale value is invalid") {
		t.Fatalf("body missing locale validation marker: %q", body)
	}
}

func TestRequestContextWithUserIDBehavior(t *testing.T) {
	t.Parallel()

	h := newHandlers(newService(fakeGateway{}), modulehandler.NewBase(nil, nil, nil), "")
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	ctx, _ := h.RequestContextAndUserID(req)
	if md, ok := metadata.FromOutgoingContext(ctx); ok && len(md.Get(grpcmeta.UserIDHeader)) > 0 {
		t.Fatalf("unexpected user metadata when resolver is nil")
	}

	h = newHandlers(newService(fakeGateway{}), modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), "")
	ctx, _ = h.RequestContextAndUserID(req)
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata")
	}
	if got := md.Get(grpcmeta.UserIDHeader); len(got) != 1 || got[0] != "user-123" {
		t.Fatalf("user metadata = %v, want [user-123]", got)
	}
}

func TestParseAppCharacterKind(t *testing.T) {
	t.Parallel()

	if kind, ok := parseAppCharacterKind("pc"); !ok || kind != CharacterKindPC {
		t.Fatalf("parseAppCharacterKind pc = (%v, %v)", kind, ok)
	}
	if kind, ok := parseAppCharacterKind("npc"); !ok || kind != CharacterKindNPC {
		t.Fatalf("parseAppCharacterKind npc = (%v, %v)", kind, ok)
	}
	if _, ok := parseAppCharacterKind("invalid"); ok {
		t.Fatalf("expected invalid character kind to fail parse")
	}
}

func managerMutationGateway() fakeGateway {
	return fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		participants: []CampaignParticipant{{
			ID:             "p-manager",
			UserID:         "user-123",
			CampaignAccess: "Manager",
		}},
	}
}

func managerMutationBase() modulehandler.Base {
	return modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil)
}
