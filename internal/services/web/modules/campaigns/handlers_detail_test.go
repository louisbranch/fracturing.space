package campaigns

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestMountServesCampaignDetailRoutes(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	paths := map[string]string{
		routepath.AppCampaign("c1"):                      "campaign-overview",
		routepath.AppCampaignEdit("c1"):                  "campaign-edit",
		routepath.AppCampaignParticipants("c1"):          "campaign-participants",
		routepath.AppCampaignParticipantCreate("c1"):     "campaign-participant-create",
		routepath.AppCampaignParticipantEdit("c1", "p1"): "campaign-participant-edit",
		routepath.AppCampaignCharacters("c1"):            "campaign-characters",
		routepath.AppCampaignCharacter("c1", "pc1"):      "campaign-character-detail",
		routepath.AppCampaignCharacterCreate("c1"):       "campaign-character-create",
		routepath.AppCampaignCharacterEdit("c1", "pc1"):  "campaign-character-edit",
	}
	for path, marker := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
		if body := rr.Body.String(); !strings.Contains(body, marker) {
			t.Fatalf("path %q body = %q, want marker %q", path, body, marker)
		}
	}
}

func TestMountStableCampaignMutationDetailRoutes(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:    []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		sessions: []campaignapp.CampaignSession{{ID: "s1", Name: "Session 1", Status: "Active"}},
	}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	paths := map[string]string{
		routepath.AppCampaignSessions("c1"):      "campaign-sessions",
		routepath.AppCampaignSession("c1", "s1"): "campaign-session-detail",
		routepath.AppCampaignInvites("c1"):       "campaign-invites",
	}
	for path, marker := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
		if body := rr.Body.String(); !strings.Contains(body, marker) {
			t.Fatalf("path %q body = %q, want marker %q", path, body, marker)
		}
	}
}

func TestMountCampaignSessionsRouteRendersSessionCards(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		sessions: []campaignapp.CampaignSession{{
			ID:     "s1",
			Name:   "First Light",
			Status: "Active",
		}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignSessions("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-session-card-id="s1"`,
		`data-campaign-session-name="First Light"`,
		`data-campaign-session-status="Active"`,
		`href="/app/campaigns/c1/sessions/s1"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing sessions marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignWorkspaceMenuRendersSessionsSectionAcrossPages(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{
			ID:               "c1",
			Name:             "The Guildhouse",
			ParticipantCount: "2",
			CharacterCount:   "2",
		}},
		sessions: []campaignapp.CampaignSession{
			{
				ID:        "s1",
				Name:      "First Light",
				Status:    "Ended",
				StartedAt: "2026-02-01 20:00 UTC",
				EndedAt:   "2026-02-01 22:00 UTC",
			},
			{
				ID:        "s2",
				Name:      "Second Light",
				Status:    "Active",
				StartedAt: "2026-02-02 20:00 UTC",
			},
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	paths := []string{
		routepath.AppCampaign("c1"),
		routepath.AppCampaignParticipants("c1"),
		routepath.AppCampaignCharacters("c1"),
		routepath.AppCampaignSessions("c1"),
		routepath.AppCampaignSession("c1", "s2"),
		routepath.AppCampaignInvites("c1"),
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mount.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
		body := rr.Body.String()
		for _, marker := range []string{
			`href="/app/campaigns/c1/invites"`,
			`href="/app/campaigns/c1/sessions"`,
			`class="badge badge-sm badge-soft badge-primary">2</div>`,
			// Only the active session (s2) appears as a sub-item.
			`data-app-side-menu-subitem="/app/campaigns/c1/sessions/s2"`,
		} {
			if !strings.Contains(body, marker) {
				t.Fatalf("path %q body missing sessions menu marker %q: %q", path, marker, body)
			}
		}
		// Ended session s1 should not appear as a sub-item.
		if strings.Contains(body, `data-app-side-menu-subitem="/app/campaigns/c1/sessions/s1"`) {
			t.Fatalf("path %q body should not contain ended session sub-item for s1", path)
		}
	}
}

func TestMountCampaignWorkspaceSessionsMenuHighlightsEntireActiveRow(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		sessions: []campaignapp.CampaignSession{
			{
				ID:        "s3",
				Name:      "Third Light",
				Status:    "Ended",
				StartedAt: "2026-02-03 20:00 UTC",
				EndedAt:   "2026-02-03 22:00 UTC",
			},
			{
				ID:        "s1",
				Name:      "Session 1",
				Status:    "Active",
				StartedAt: "2026-02-01 20:00 UTC",
			},
			{
				ID:        "s2",
				Name:      "Second Light",
				Status:    "Ended",
				StartedAt: "2026-02-02 20:00 UTC",
				EndedAt:   "2026-02-02 22:00 UTC",
			},
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipants("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()

	// Only the active session (s1) should appear as a sub-item in the menu.
	s1Idx := strings.Index(body, `data-app-side-menu-subitem="/app/campaigns/c1/sessions/s1"`)
	if s1Idx == -1 {
		t.Fatalf("expected active session s1 subitem in side menu: %q", body)
	}
	// Ended sessions should not appear.
	if strings.Contains(body, `data-app-side-menu-subitem="/app/campaigns/c1/sessions/s2"`) {
		t.Fatalf("ended session s2 should not appear as sub-item")
	}
	if strings.Contains(body, `data-app-side-menu-subitem="/app/campaigns/c1/sessions/s3"`) {
		t.Fatalf("ended session s3 should not appear as sub-item")
	}

	activeRowMarker := `data-app-side-menu-subitem="/app/campaigns/c1/sessions/s1" data-app-side-menu-subitem-active-session="true"`
	activeRowIdx := strings.Index(body, activeRowMarker)
	if activeRowIdx == -1 {
		t.Fatalf("expected active session row marker %q in output: %q", activeRowMarker, body)
	}
	activeRowBody := body[activeRowIdx:]
	if !strings.Contains(activeRowBody, `class="block rounded-md border px-3 py-2 leading-tight transition-colors border-success bg-base-200"`) {
		t.Fatalf("expected full active-session row highlight class in output: %q", activeRowBody)
	}
	for _, marker := range []string{
		`data-app-side-menu-subitem-start="Start: 2026-02-01 20:00 UTC"`,
		`>Session 1</a>`,
		`Join Game</a>`,
	} {
		if !strings.Contains(activeRowBody, marker) {
			t.Fatalf("expected active session detail marker %q in output: %q", marker, activeRowBody)
		}
	}
}

func TestMountCampaignSessionsRouteRendersReadinessBlockers(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		sessionReadiness: campaignapp.CampaignSessionReadiness{
			Ready: false,
			Blockers: []campaignapp.CampaignSessionReadinessBlocker{
				{
					Code:    "SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED",
					Message: "Campaign readiness requires at least one AI-controlled GM participant for AI GM mode",
				},
			},
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignSessions("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-session-readiness-blocked="true"`,
		`data-campaign-session-readiness-blockers="true"`,
		`data-campaign-session-readiness-blocker-code="SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED"`,
		`Campaign readiness requires at least one AI-controlled GM participant for AI GM mode`,
		`data-campaign-session-start-disabled="true"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing readiness marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignSessionDetailRouteRendersSelectedSession(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		sessions: []campaignapp.CampaignSession{{
			ID:     "s1",
			Name:   "First Light",
			Status: "Active",
		}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignSession("c1", "s1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-session-detail-id="s1"`,
		`data-campaign-session-detail-name="First Light"`,
		`data-campaign-session-detail-status="Active"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing session detail marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignInvitesRouteRendersInviteCards(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		participants: []campaignapp.CampaignParticipant{
			{ID: "p-eligible", Name: "Aria", Controller: "Human"},
			{ID: "p-bound", Name: "Bound Seat", Controller: "Human", UserID: "user-9"},
			{ID: "p-ai", Name: "Oracle", Controller: "AI"},
			{ID: "p1", Name: "Pending Seat", Controller: "Human"},
		},
		invites: []campaignapp.CampaignInvite{{
			ID:                "inv-1",
			ParticipantID:     "p1",
			ParticipantName:   "Pending Seat",
			RecipientUserID:   "user-2",
			RecipientUsername: "river",
			HasRecipient:      true,
			Status:            "Pending",
		}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignInvites("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-invites-header="true"`,
		`<h2 class="card-title">Invites</h2>`,
		`data-campaign-invite-card-id="inv-1"`,
		`data-campaign-invite-participant="p1"`,
		`data-campaign-invite-recipient="river"`,
		`@river`,
		`data-campaign-invite-status="Pending"`,
		`data-campaign-invite-public-url="true"`,
		`value="http://example.com/invite/inv-1"`,
		`data-campaign-invite-create-form="true"`,
		`data-campaign-invite-create-submit="true"`,
		`data-campaign-invite-create-participant-select="true"`,
		`data-campaign-invite-create-option-id="p-eligible"`,
		`>Aria</option>`,
		`data-campaign-invite-revoke-form="true"`,
		`data-campaign-invite-revoke-submit="true"`,
		`<script defer src="/static/username-input.js"></script>`,
		`class="menu-active" href="/app/campaigns/c1/invites"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing invite marker %q: %q", marker, body)
		}
	}
	// Invariant: the selector must exclude bound, AI, and already-pending seats.
	for _, marker := range []string{
		`data-campaign-invite-create-option-id="p-bound"`,
		`data-campaign-invite-create-option-id="p-ai"`,
		`data-campaign-invite-create-option-id="p1"`,
		`name="participant_id" required placeholder=`,
		`>user-2<`,
	} {
		if strings.Contains(body, marker) {
			t.Fatalf("body should not render ineligible invite selector marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignInvitesRouteHidesManageControlsWithoutInvitePermission(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		invites: []campaignapp.CampaignInvite{{
			ID:              "inv-1",
			ParticipantID:   "p1",
			RecipientUserID: "user-2",
			Status:          "Pending",
		}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:  true,
			Allowed:    false,
			ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED",
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignInvites("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-invite-card-id="inv-1"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing invite marker %q: %q", marker, body)
		}
	}
	// Invariant: invite-manage navigation must not be exposed without permission.
	if strings.Contains(body, `data-app-side-menu-item="/app/campaigns/c1/invites"`) {
		t.Fatalf("body should hide invites menu item without permission: %q", body)
	}
	for _, marker := range []string{
		`data-campaign-invite-create-entry="true"`,
		`data-campaign-invite-create-form="true"`,
		`data-campaign-invite-revoke-form="true"`,
		`data-campaign-invite-create-disabled="true"`,
		`data-campaign-invite-revoke-disabled="true"`,
	} {
		if strings.Contains(body, marker) {
			t.Fatalf("body should not render invite manage control marker %q: %q", marker, body)
		}
	}
	// Invariant: without invite-manage permission, the fragment must not load the client-side mutation UX script.
	if strings.Contains(body, `<script defer src="/static/username-input.js"></script>`) {
		t.Fatalf("body should not render invite mutation script without permission: %q", body)
	}
}

func TestMountCampaignInvitesRouteHidesPublicURLForNonPendingStatuses(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		invites: []campaignapp.CampaignInvite{
			{ID: "inv-pending", ParticipantID: "p1", ParticipantName: "Pending Seat", Status: "Pending"},
			{ID: "inv-claimed", ParticipantID: "p2", ParticipantName: "Accepted Seat", Status: "Claimed"},
			{ID: "inv-declined", ParticipantID: "p3", ParticipantName: "Rejected Seat", Status: "Declined"},
			{ID: "inv-revoked", ParticipantID: "p4", ParticipantName: "Revoked Seat", Status: "Revoked"},
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignInvites("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Count(body, `data-campaign-invite-public-url="true"`) != 1 {
		t.Fatalf("public url inputs = %d, want 1 in body %q", strings.Count(body, `data-campaign-invite-public-url="true"`), body)
	}
	if !strings.Contains(body, `value="http://example.com/invite/inv-pending"`) {
		t.Fatalf("body missing pending invite public url: %q", body)
	}
	for _, marker := range []string{
		`value="http://example.com/invite/inv-claimed"`,
		`value="http://example.com/invite/inv-declined"`,
		`value="http://example.com/invite/inv-revoked"`,
	} {
		if strings.Contains(body, marker) {
			t.Fatalf("body should not include non-pending public url %q: %q", marker, body)
		}
	}
}

func TestMountCampaignInvitesRouteDisablesManageControlsWhileActionsLocked(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		participants: []campaignapp.CampaignParticipant{{
			ID:         "p-open",
			Name:       "Aria",
			Controller: "Human",
		}},
		invites: []campaignapp.CampaignInvite{{
			ID:              "inv-1",
			ParticipantID:   "p1",
			RecipientUserID: "user-2",
			Status:          "Pending",
		}},
		sessions: []campaignapp.CampaignSession{{
			ID:        "s1",
			Name:      "First Light",
			Status:    "Active",
			StartedAt: "2026-02-01 20:00 UTC",
		}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:  true,
			Allowed:    true,
			ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL",
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignInvites("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-invite-create-entry="true"`,
		`data-campaign-invite-create-participant-select="true"`,
		`data-campaign-invite-create-disabled="true"`,
		`data-campaign-invite-revoke-disabled="true"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing locked invite marker %q: %q", marker, body)
		}
	}
	for _, marker := range []string{
		`data-campaign-invite-create-form="true"`,
		`data-campaign-invite-revoke-form="true"`,
	} {
		if strings.Contains(body, marker) {
			t.Fatalf("body should not render live invite form marker %q while locked: %q", marker, body)
		}
	}
	// Invariant: locked invite actions must not activate the client-side mutation UX script.
	if strings.Contains(body, `<script defer src="/static/username-input.js"></script>`) {
		t.Fatalf("body should not render invite mutation script while locked: %q", body)
	}
}

func TestMountCampaignInvitesRouteDisablesCreateWhenNoEligibleSeats(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		participants: []campaignapp.CampaignParticipant{
			{ID: "p-bound", Name: "Bound Seat", Controller: "Human", UserID: "user-1"},
			{ID: "p-ai", Name: "Oracle", Controller: "AI"},
			{ID: "p-pending", Name: "Pending Seat", Controller: "Human"},
		},
		invites: []campaignapp.CampaignInvite{{
			ID:            "inv-1",
			ParticipantID: "p-pending",
			Status:        "Pending",
		}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:  true,
			Allowed:    true,
			ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL",
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignInvites("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-invite-create-form="true"`,
		`data-campaign-invite-create-participant-disabled="true"`,
		`data-campaign-invite-create-empty="true"`,
		`data-campaign-invite-create-disabled="true"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing no-eligible-seat marker %q: %q", marker, body)
		}
	}
	// Invariant: no eligible seat means no selectable invite target options.
	if strings.Contains(body, `data-campaign-invite-create-option-id=`) {
		t.Fatalf("body should not render selectable invite target options when none are eligible: %q", body)
	}
}

func TestMountCampaignCharacterDetailRouteRendersSelectedCharacter(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			AvatarURL:  "/static/avatars/aria.png",
		}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacter("c1", "char-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-detail-id="char-1"`,
		`data-campaign-character-detail-pronouns=`,
		`data-campaign-character-detail-kind="PC"`,
		`data-campaign-character-control-current-controller="Ariadne"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing character detail marker %q: %q", marker, body)
		}
	}
	for _, marker := range []string{
		`data-campaign-character-detail-name=`,
		`data-campaign-character-detail-controller=`,
	} {
		if strings.Contains(body, marker) {
			t.Fatalf("body should not render redundant character detail marker %q: %q", marker, body)
		}
	}
	if !strings.Contains(body, `data-campaign-character-detail-id="char-1">Aria</h2>`) {
		t.Fatalf("character detail heading should render character name, got %q", body)
	}
	if strings.Contains(body, `data-campaign-character-detail-id="char-1">char-1</h2>`) {
		t.Fatalf("character detail heading should not render character id, got %q", body)
	}
}

func TestMountCampaignCharacterDetailShowsClaimAndManagerControlsForUnassignedCharacter(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		participants: []campaignapp.CampaignParticipant{
			{ID: "p-1", UserID: "user-123", Name: "Ariadne", CampaignAccess: "Manager"},
			{ID: "p-2", UserID: "user-456", Name: "Moss", CampaignAccess: "Member"},
		},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Unassigned",
			AvatarURL:  "/static/avatars/aria.png",
		}},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{
			{CheckID: "char-1", Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
		},
		authorizationDecision: campaignapp.AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
	}, modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacter("c1", "char-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-control-card="true"`,
		`data-campaign-character-control-manager-card="true"`,
		`data-campaign-character-claim-form="true"`,
		`data-campaign-character-controller-form="true"`,
		`data-campaign-character-delete-form="true"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing character control marker %q: %q", marker, body)
		}
	}
	if strings.Contains(body, "Signed in as Ariadne.") {
		t.Fatalf("character detail should not render signed-in participant copy: %q", body)
	}
	if strings.Contains(body, `data-campaign-character-release-form="true"`) {
		t.Fatalf("character detail should not show release action for unassigned character: %q", body)
	}
}

func TestMountCampaignCharacterDetailShowsReleaseForCurrentController(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		participants: []campaignapp.CampaignParticipant{
			{ID: "p-1", UserID: "user-123", Name: "Ariadne", CampaignAccess: "Member"},
		},
		characters: []campaignapp.CampaignCharacter{{
			ID:                      "char-1",
			Name:                    "Aria",
			Kind:                    "PC",
			Controller:              "Ariadne",
			ControllerParticipantID: "p-1",
		}},
	}, modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacter("c1", "char-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-campaign-character-release-form="true"`) {
		t.Fatalf("character detail should show release action for current controller: %q", body)
	}
	if strings.Contains(body, `data-campaign-character-claim-form="true"`) {
		t.Fatalf("character detail should not show claim action for current controller: %q", body)
	}
}

func TestMountCampaignCharacterDetailHidesSelfServiceControlsWhenAnotherParticipantControlsCharacter(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		participants: []campaignapp.CampaignParticipant{
			{ID: "p-1", UserID: "user-123", Name: "Ariadne", CampaignAccess: "Member"},
			{ID: "p-2", UserID: "user-456", Name: "Moss", CampaignAccess: "Member"},
		},
		characters: []campaignapp.CampaignCharacter{{
			ID:                      "char-1",
			Name:                    "Aria",
			Kind:                    "PC",
			Controller:              "Moss",
			ControllerParticipantID: "p-2",
		}},
		authorizationDecision: campaignapp.AuthorizationDecision{Evaluated: true, Allowed: false, ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED"},
	}, modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacter("c1", "char-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-claim-form="true"`,
		`data-campaign-character-release-form="true"`,
		`data-campaign-character-controller-form="true"`,
	} {
		if strings.Contains(body, marker) {
			t.Fatalf("character detail unexpectedly rendered %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharacterDetailDisablesControlAndDeleteActionsDuringActiveSession(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		participants: []campaignapp.CampaignParticipant{
			{ID: "p-1", UserID: "user-123", Name: "Ariadne", CampaignAccess: "Manager"},
		},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Unassigned",
		}},
		sessions: []campaignapp.CampaignSession{{ID: "sess-1", Name: "Session One", Status: "Active"}},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{
			{CheckID: "char-1", Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
		},
		authorizationDecision: campaignapp.AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
	}, modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacter("c1", "char-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-claim-disabled="true"`,
		`data-campaign-character-controller-submit-disabled="true"`,
		`data-campaign-character-delete-disabled="true"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing active-session disabled marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharacterDetailBreadcrumbUsesCharacterName(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
		}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacter("c1", "char-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, marker := range []string{
		`class="breadcrumbs text-sm"`,
		`href="/app/campaigns/c1/characters"`,
		`>Aria</li>`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing breadcrumb marker %q: %q", marker, body)
		}
	}
	if strings.Contains(body, `<li>char-1</li>`) {
		t.Fatalf("character detail breadcrumb should not render character id, got %q", body)
	}
}

func TestMountCampaignCharacterDetailRendersCreationLinkCard(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
		}},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{
			{CheckID: "char-1", Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
		},
		characterCreationProgress: campaignapp.CampaignCharacterCreationProgress{
			Steps:    []campaignapp.CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
			NextStep: 1,
		},
		characterCreationCatalog: campaignapp.CampaignCharacterCreationCatalog{
			Classes:    []campaignapp.CatalogClass{{ID: "warrior", Name: "Warrior"}},
			Subclasses: []campaignapp.CatalogSubclass{{ID: "guardian", Name: "Guardian", ClassID: "warrior"}},
		},
	}, modulehandler.NewTestBase(), defaultTestWorkflows()))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacter("c1", "char-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-character-creation-workflow="true"`,
		`Daggerheart Character Sheet`,
		`data-character-creation-link="true"`,
		`/app/campaigns/c1/characters/char-1/creation`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing workflow marker %q: %q", marker, body)
		}
	}
	if strings.Contains(body, `data-campaign-character-update-form="true"`) {
		t.Fatalf("character detail should no longer render inline update form: %q", body)
	}
}

func TestMountCampaignCharacterDetailHidesWorkflowForNonDaggerheartCampaigns(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:           []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		workspaceSystem: "Pathfinder",
		characters: []campaignapp.CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
		}},
		characterCreationProgressErr: errors.New("workflow should not be loaded for non-daggerheart systems"),
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacter("c1", "char-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, `data-character-creation-workflow="true"`) {
		t.Fatalf("body unexpectedly contains character creation workflow card: %q", body)
	}
}

func TestMountCampaignCharacterCreatePageRendersDedicatedForm(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacterCreate("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-create-page="true"`,
		`data-campaign-character-editor-submit="true"`,
		`action="/app/campaigns/c1/characters/create"`,
		`name="name"`,
		`name="pronouns"`,
		`list="campaign-character-pronouns-presets"`,
		`<option value="they/them"></option>`,
		`<option value="he/him"></option>`,
		`<option value="she/her"></option>`,
		`<option value="it/its"></option>`,
		`type="hidden" name="kind" value="pc"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing character create marker %q: %q", marker, body)
		}
	}
	if strings.Contains(body, `>Controller</dt>`) || strings.Contains(body, `>Unassigned</dd>`) {
		t.Fatalf("character create page should not render controller summary: %q", body)
	}
}

func TestMountCampaignCharacterEditPageRendersDedicatedForm(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			Pronouns:   "she/her",
			CanEdit:    true,
		}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacterEdit("c1", "char-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-edit-page="true"`,
		`data-campaign-character-editor-submit="true"`,
		`action="/app/campaigns/c1/characters/char-1/edit"`,
		`list="campaign-character-pronouns-presets"`,
		`value="Aria"`,
		`value="she/her"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing character edit marker %q: %q", marker, body)
		}
	}
	if strings.Contains(body, `>Controller</dt>`) || strings.Contains(body, `>Ariadne</dd>`) {
		t.Fatalf("character edit page should not render controller summary: %q", body)
	}
}

func TestMountCampaignCharactersDisableMutationsDuringActiveSession(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
		}},
		sessions: []campaignapp.CampaignSession{{ID: "sess-1", Name: "Live", Status: "Active"}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-campaign-character-create-disabled="true"`) {
		t.Fatalf("characters page should disable add-character during active session: %q", body)
	}
	if strings.Contains(body, `data-campaign-character-create-link="true"`) {
		t.Fatalf("characters page should not render active add-character link during active session: %q", body)
	}
}

func TestMountCampaignCharacterDetailDisablesActionsDuringActiveSession(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			CanEdit:    true,
		}},
		sessions: []campaignapp.CampaignSession{{ID: "sess-1", Name: "Live", Status: "Active"}},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{
			{CheckID: "char-1", Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
		},
		characterCreationProgress: campaignapp.CampaignCharacterCreationProgress{
			Steps:    []campaignapp.CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
			NextStep: 1,
		},
		characterCreationCatalog: campaignapp.CampaignCharacterCreationCatalog{
			Classes:    []campaignapp.CatalogClass{{ID: "warrior", Name: "Warrior"}},
			Subclasses: []campaignapp.CatalogSubclass{{ID: "guardian", Name: "Guardian", ClassID: "warrior"}},
		},
	}, modulehandler.NewTestBase(), defaultTestWorkflows()))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacter("c1", "char-1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-edit-disabled="true"`,
		`data-character-creation-disabled="true"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing active-session disabled marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignOverviewRendersWorkspaceDetailsAndMenu(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{
		ID:            "c1",
		Name:          "The Guildhouse",
		Theme:         "Stormbound intrigue",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`class="menu bg-base-200 rounded-box w-full"`,
		`href="/app/campaigns/c1"`,
		`hx-get="/app/campaigns/c1"`,
		`>Overview`,
		`data-campaign-overview-name="The Guildhouse"`,
		`data-campaign-overview-campaign-id="c1"`,
		`data-campaign-overview-theme="Stormbound intrigue"`,
		`data-campaign-overview-system="Daggerheart"`,
		`data-campaign-overview-gm-mode="Human"`,
		`data-campaign-overview-ai-binding-status="Not required"`,
		`data-campaign-overview-status="Active"`,
		`data-campaign-overview-locale="English (US)"`,
		`data-campaign-overview-intent="Standard"`,
		`data-campaign-overview-access-policy="Public"`,
		`data-campaign-overview-edit-link="true"`,
		`href="/app/campaigns/c1/edit"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing campaign workspace marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignOverviewRendersPendingAIBindingStatusAndManageLinkForOwner(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:           []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		workspaceGMMode: "AI",
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:           true,
			Allowed:             true,
			ReasonCode:          "AUTHZ_ALLOW_ACCESS_LEVEL",
			ActorCampaignAccess: "Owner",
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-overview-ai-binding-status="Pending"`,
		`data-campaign-overview-ai-binding-link="true"`,
		`href="/app/campaigns/c1/ai-binding"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing overview AI binding marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignOverviewAllowsHead(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{
		ID:            "c1",
		Name:          "The Guildhouse",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, routepath.AppCampaign("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMountCampaignEditRequiresManagerOrOwnerAccess(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:  true,
			Allowed:    false,
			ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED",
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignEdit("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestMountCampaignParticipantsMenuAndPortraitGallery(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{
			ID:               "c1",
			Name:             "The Guildhouse",
			ParticipantCount: "2",
			CoverImageURL:    "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		participants: []campaignapp.CampaignParticipant{
			{
				ID:             "p-z",
				Name:           "Zara",
				Role:           "Player",
				CampaignAccess: "Member",
				Controller:     "Human",
				AvatarURL:      "/static/avatars/zara.png",
			},
			{
				ID:             "p-a",
				Name:           "Aria",
				Role:           "GM",
				CampaignAccess: "Owner",
				Controller:     "AI",
				AvatarURL:      "/static/avatars/aria.png",
			},
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipants("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`href="/app/campaigns/c1/participants"`,
		`data-campaign-participants-header="true"`,
		`class="grid grid-cols-1 md:grid-cols-2 gap-4"`,
		`data-campaign-participant-card-id="p-a"`,
		`data-campaign-participant-name="Aria"`,
		`data-campaign-participant-role="GM"`,
		`data-campaign-participant-access="Owner"`,
		`data-campaign-participant-controller="AI"`,
		`src="/static/avatars/aria.png"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing participants gallery marker %q: %q", marker, body)
		}
	}
	sideMenuParticipants := strings.Index(body, `href="/app/campaigns/c1/participants"`)
	if sideMenuParticipants == -1 {
		t.Fatalf("expected participants side-menu item in output: %q", body)
	}
	if !strings.Contains(body[sideMenuParticipants:], `class="badge badge-sm badge-soft badge-primary">2</div>`) {
		t.Fatalf("expected participants count badge in side-menu: %q", body)
	}
	ariaIdx := strings.Index(body, `data-campaign-participant-card-id="p-a"`)
	zaraIdx := strings.Index(body, `data-campaign-participant-card-id="p-z"`)
	if ariaIdx == -1 || zaraIdx == -1 {
		t.Fatalf("expected both participant cards in output")
	}
	if ariaIdx > zaraIdx {
		t.Fatalf("expected participant cards sorted by name: %q", body)
	}
	if count := strings.Count(body, `class="menu-active"`); count != 1 {
		t.Fatalf("menu-active count = %d, want 1", count)
	}
	if !strings.Contains(body, `class="menu-active" href="/app/campaigns/c1/participants"`) {
		t.Fatalf("expected participants menu item active: %q", body)
	}
	if count := strings.Count(body, `href="#lucide-book-open"`); count < 2 {
		t.Fatalf("book-open icon count = %d, want at least 2", count)
	}
	if !strings.Contains(body, `href="#lucide-users"`) {
		t.Fatalf("expected participants side-menu icon in output: %q", body)
	}
	if !strings.Contains(body, `href="#lucide-square-user"`) {
		t.Fatalf("expected characters side-menu icon in output: %q", body)
	}
}

func TestMountCampaignParticipantsShowsEditLinkForEditableParticipants(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		participants: []campaignapp.CampaignParticipant{
			{ID: "p-a", Name: "Aria", Role: "GM", CampaignAccess: "Owner", Controller: "Human", AvatarURL: "/static/avatars/aria.png"},
			{ID: "p-b", Name: "Bram", Role: "Player", CampaignAccess: "Member", Controller: "Human", AvatarURL: "/static/avatars/bram.png"},
		},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{
			{CheckID: "p-a", Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
			{CheckID: "p-b", Evaluated: true, Allowed: false, ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED"},
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipants("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `href="/app/campaigns/c1/participants/p-a/edit"`) {
		t.Fatalf("expected editable participant link in output: %q", body)
	}
	if strings.Contains(body, `href="/app/campaigns/c1/participants/p-b/edit"`) {
		t.Fatalf("unexpected edit link for read-only participant: %q", body)
	}
}

func TestMountCampaignParticipantsShowsEditLinkForSelfOwnedParticipant(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		participants: []campaignapp.CampaignParticipant{
			{ID: "p-a", UserID: "user-1", Name: "Aria", Role: "Player", CampaignAccess: "Member", Controller: "Human", AvatarURL: "/static/avatars/aria.png"},
		},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{
			{CheckID: "p-a", Evaluated: true, Allowed: false, ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED"},
		},
	}, modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipants("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `href="/app/campaigns/c1/participants/p-a/edit"`) {
		t.Fatalf("expected self-owned participant link in output: %q", body)
	}
}

func TestMountCampaignParticipantsHighlightsViewerParticipantCard(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		participants: []campaignapp.CampaignParticipant{
			{ID: "p-a", UserID: "user-1", Name: "Aria", Role: "Player", CampaignAccess: "Member", Controller: "Human", AvatarURL: "/static/avatars/aria.png"},
			{ID: "p-b", UserID: "user-2", Name: "Bram", Role: "Player", CampaignAccess: "Member", Controller: "Human", AvatarURL: "/static/avatars/bram.png"},
		},
	}, modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipants("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `class="card bg-base-100 border border-primary shadow-sm md:card-side" data-campaign-participant-card-id="p-a" data-campaign-participant-current-user="true"`) {
		t.Fatalf("expected viewer participant card to use primary border: %q", body)
	}
	if !strings.Contains(body, `class="card bg-base-100 border border-base-300 shadow-sm md:card-side" data-campaign-participant-card-id="p-b" data-campaign-participant-current-user="false"`) {
		t.Fatalf("expected non-viewer participant card to keep base border: %q", body)
	}
}

func TestMountCampaignParticipantsShowsCreateLinkWhenManageAllowed(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		participants: []campaignapp.CampaignParticipant{{
			ID:             "p-manager",
			UserID:         "user-123",
			CampaignAccess: "Manager",
		}},
	}, modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipants("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-campaign-participants-add-link="true"`) {
		t.Fatalf("expected create link marker in output: %q", body)
	}
	if !strings.Contains(body, `href="/app/campaigns/c1/participants/create"`) {
		t.Fatalf("expected create participant href in output: %q", body)
	}
}

func TestMountCampaignParticipantCreateRendersForm(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:        []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		participants: []campaignapp.CampaignParticipant{{ID: "p-owner", UserID: "user-123", CampaignAccess: "Owner"}},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{
			{CheckID: "member", Evaluated: true, Allowed: true},
			{CheckID: "manager", Evaluated: true, Allowed: true},
			{CheckID: "owner", Evaluated: true, Allowed: true},
		},
	}, modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipantCreate("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`campaign-participant-create`,
		`data-campaign-participant-create-page="true"`,
		`data-campaign-participant-create-form="true"`,
		`action="/app/campaigns/c1/participants/create"`,
		`name="name"`,
		`name="role"`,
		`name="campaign_access"`,
		`value="gm"`,
		`value="player"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing participant create marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignParticipantCreateOmitsGMRoleForAIGMCampaigns(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:           []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		workspaceGMMode: "AI",
		participants:    []campaignapp.CampaignParticipant{{ID: "p-owner", UserID: "user-123", CampaignAccess: "Owner"}},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{
			{CheckID: "member", Evaluated: true, Allowed: true},
			{CheckID: "manager", Evaluated: true, Allowed: true},
			{CheckID: "owner", Evaluated: true, Allowed: true},
		},
	}, modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipantCreate("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, `value="gm"`) {
		t.Fatalf("unexpected GM option for AI GM campaign: %q", body)
	}
	if !strings.Contains(body, `value="player"`) {
		t.Fatalf("expected player option in output: %q", body)
	}
}

func TestMountCampaignParticipantEditRendersForm(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:           []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		workspaceGMMode: "Human",
		participant: campaignapp.CampaignParticipant{
			ID:             "p-a",
			Name:           "Aria",
			Role:           "GM",
			CampaignAccess: "Owner",
			Pronouns:       "she/her",
		},
		authorizationDecision: campaignapp.AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{
			{CheckID: "member", Evaluated: true, Allowed: true},
			{CheckID: "manager", Evaluated: true, Allowed: true},
			{CheckID: "owner", Evaluated: true, Allowed: true},
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipantEdit("c1", "p-a"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`campaign-participant-edit`,
		`data-campaign-participant-edit-page="true"`,
		`data-campaign-participant-edit-submit="true"`,
		`action="/app/campaigns/c1/participants/p-a/edit"`,
		`name="role"`,
		`name="pronouns"`,
		`name="campaign_access"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing participant edit marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignParticipantEditAllowsSelfOwnedParticipant(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:           []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		workspaceGMMode: "Human",
		participant: campaignapp.CampaignParticipant{
			ID:             "p-a",
			UserID:         "user-1",
			Name:           "Aria",
			Role:           "Player",
			CampaignAccess: "Member",
			Pronouns:       "she/her",
		},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:  true,
			Allowed:    false,
			ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED",
		},
	}, modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipantEdit("c1", "p-a"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-participant-edit-page="true"`,
		`data-campaign-participant-edit-submit="true"`,
		`data-campaign-participant-role-readonly="true"`,
		`data-campaign-participant-access-readonly="true"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing self-edit marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignParticipantEditOmitsGMRoleForHumanSeatsInAIGMCampaigns(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:           []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		workspaceGMMode: "AI",
		participant: campaignapp.CampaignParticipant{
			ID:             "p-a",
			Name:           "Aria",
			Role:           "GM",
			CampaignAccess: "Member",
			Controller:     "Human",
			Pronouns:       "she/her",
		},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated: true,
			Allowed:   true,
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipantEdit("c1", "p-a"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, `value="gm"`) {
		t.Fatalf("unexpected GM option for AI GM campaign human participant: %q", body)
	}
	if !strings.Contains(body, `value="player"`) {
		t.Fatalf("expected player option in output: %q", body)
	}
}

func TestMountCampaignParticipantEditOmitsAIBindingControlsForAIGMSeats(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:              []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		workspaceAIAgentID: "agent-current",
		participant: campaignapp.CampaignParticipant{
			ID:             "p-ai",
			Name:           "Caretaker",
			Role:           "GM",
			CampaignAccess: "Member",
			Controller:     "AI",
			Pronouns:       "it/its",
		},
		campaignAIAgents: []campaignapp.CampaignAIAgentOption{{ID: "agent-current", Label: "current", Enabled: true}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:           true,
			Allowed:             true,
			ReasonCode:          "AUTHZ_ALLOW_ACCESS_LEVEL",
			ActorCampaignAccess: "Owner",
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipantEdit("c1", "p-ai"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-participant-role-readonly="true"`,
		`data-campaign-participant-access-readonly="true"`,
		`type="hidden" name="role" value="gm"`,
		`type="hidden" name="campaign_access" value="member"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing AI participant edit marker %q: %q", marker, body)
		}
	}
	for _, marker := range []string{
		`data-campaign-ai-binding-form="true"`,
		`data-campaign-participant-edit-layout="ai"`,
		`name="ai_agent_id"`,
		`action="/app/campaigns/c1/ai-binding"`,
	} {
		if strings.Contains(body, marker) {
			t.Fatalf("body unexpectedly contains removed AI binding marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignAIBindingPageRendersForOwner(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:              []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		workspaceAIAgentID: "agent-current",
		campaignAIAgents:   []campaignapp.CampaignAIAgentOption{{ID: "agent-current", Label: "current", Enabled: true}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:           true,
			Allowed:             true,
			ReasonCode:          "AUTHZ_ALLOW_ACCESS_LEVEL",
			ActorCampaignAccess: "Owner",
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignAIBinding("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-ai-binding-page="true"`,
		`action="/app/campaigns/c1/ai-binding"`,
		`name="ai_agent_id"`,
		`>current</option>`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing campaign AI binding marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignAIBindingPageRequiresOwnerAccess(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:           true,
			Allowed:             true,
			ReasonCode:          "AUTHZ_ALLOW_ACCESS_LEVEL",
			ActorCampaignAccess: "Member",
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignAIBinding("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestMountCampaignParticipantsFailsWhenGatewayReturnsError(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{
			ID:             "c1",
			Name:           "The Guildhouse",
			CharacterCount: "2",
			CoverImageURL:  "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		participantsErr: apperrors.E(apperrors.KindUnavailable, "participants unavailable"),
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipants("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestMountCampaignParticipantsFailsClosedWhenParticipantClientMissing(t *testing.T) {
	t.Parallel()

	deps := campaigngateway.GRPCGatewayDeps{Catalog: campaigngateway.CatalogGatewayDeps{Read: campaigngateway.CatalogReadDeps{Campaign: fakeCampaignClient{}}}}
	m := New(configWithGRPCDeps(deps, modulehandler.NewTestBase(), nil))
	_, err := m.Mount()
	if err == nil {
		t.Fatalf("expected Mount() validation error")
	}
	if !strings.Contains(err.Error(), "participant-reads") {
		t.Fatalf("Mount() error = %v, want participant validation failure", err)
	}
}

func TestMountCampaignCharactersMenuAndPortraitGallery(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{
			ID:             "c1",
			Name:           "The Guildhouse",
			CharacterCount: "2",
			CoverImageURL:  "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		characters: []campaignapp.CampaignCharacter{
			{
				ID:         "ch-z",
				Name:       "Zara",
				Kind:       "NPC",
				Controller: "Moss",
				AvatarURL:  "/static/avatars/zara.png",
			},
			{
				ID:         "ch-a",
				Name:       "Aria",
				Kind:       "PC",
				Controller: "Ariadne",
				AvatarURL:  "/static/avatars/aria.png",
			},
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`href="/app/campaigns/c1/characters"`,
		`data-campaign-character-create-entry="true"`,
		`data-campaign-character-create-link="true"`,
		`Add Character`,
		`class="grid grid-cols-1 md:grid-cols-2 gap-4"`,
		`data-campaign-character-card-id="ch-a"`,
		`data-campaign-character-name="Aria"`,
		`href="/app/campaigns/c1/characters/ch-a"`,
		`data-campaign-character-detail-link="true"`,
		`data-campaign-character-view-link="true"`,
		`data-campaign-character-kind="PC"`,
		`data-campaign-character-controller="Ariadne"`,
		`src="/static/avatars/aria.png"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing characters gallery marker %q: %q", marker, body)
		}
	}
	sideMenuCharacters := strings.Index(body, `href="/app/campaigns/c1/characters"`)
	if sideMenuCharacters == -1 {
		t.Fatalf("expected characters side-menu item in output: %q", body)
	}
	if !strings.Contains(body[sideMenuCharacters:], `class="badge badge-sm badge-soft badge-primary">2</div>`) {
		t.Fatalf("expected characters count badge in side-menu: %q", body)
	}
	ariaIdx := strings.Index(body, `data-campaign-character-card-id="ch-a"`)
	zaraIdx := strings.Index(body, `data-campaign-character-card-id="ch-z"`)
	if ariaIdx == -1 || zaraIdx == -1 {
		t.Fatalf("expected both character cards in output")
	}
	if ariaIdx > zaraIdx {
		t.Fatalf("expected character cards sorted by name: %q", body)
	}
	if count := strings.Count(body, `class="menu-active"`); count != 1 {
		t.Fatalf("menu-active count = %d, want 1", count)
	}
	if !strings.Contains(body, `class="menu-active" href="/app/campaigns/c1/characters"`) {
		t.Fatalf("expected characters menu item active: %q", body)
	}
}

func TestMountCampaignCharactersEmptyStateStillShowsCreateEntry(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}}}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`No characters yet.`,
		`data-campaign-character-create-entry="true"`,
		`data-campaign-character-create-link="true"`,
		`Add Character`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing empty-state create marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharactersRendersDaggerheartSummaryRows(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "ch-a",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			Daggerheart: &campaignapp.CampaignCharacterDaggerheartSummary{
				Level:         2,
				ClassName:     "Warrior",
				SubclassName:  "Guardian",
				AncestryName:  "Drakona",
				CommunityName: "Wanderborne",
			},
		}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-daggerheart-level="2"`,
		`data-campaign-character-daggerheart-class-summary="Warrior / Guardian"`,
		`data-campaign-character-daggerheart-heritage-summary="Drakona / Wanderborne"`,
		`L 2`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing daggerheart summary marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharactersHidesIncompleteDaggerheartSummaryRows(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "ch-a",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			Daggerheart: &campaignapp.CampaignCharacterDaggerheartSummary{
				Level:         2,
				ClassName:     "Warrior",
				SubclassName:  "Guardian",
				AncestryName:  "Drakona",
				CommunityName: "",
			},
		}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-daggerheart-level=`,
		`data-campaign-character-daggerheart-class-summary=`,
		`data-campaign-character-daggerheart-heritage-summary=`,
	} {
		if strings.Contains(body, marker) {
			t.Fatalf("body should hide incomplete daggerheart summary marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharactersUsesViewCharacterCTAForEditableCharacters(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{
			ID:            "c1",
			Name:          "The Guildhouse",
			CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "ch-a",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
		}},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{{CheckID: "ch-a", Evaluated: true, Allowed: true}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`href="/app/campaigns/c1/characters/ch-a"`,
		`data-campaign-character-view-link="true"`,
		`View Character`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing view-character marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharactersHighlightsViewerOwnedCharacterCard(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		characters: []campaignapp.CampaignCharacter{
			{ID: "ch-a", Name: "Aria", Kind: "PC", Controller: "Ariadne", OwnedByViewer: true},
			{ID: "ch-b", Name: "Bramble", Kind: "PC", Controller: "Scout", OwnedByViewer: false},
		},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `class="card bg-base-100 border border-primary shadow-sm md:card-side" data-campaign-character-card-id="ch-a" data-campaign-character-owned-by-viewer="true"`) {
		t.Fatalf("expected viewer-owned character card to use primary border: %q", body)
	}
	if !strings.Contains(body, `class="card bg-base-100 border border-base-300 shadow-sm md:card-side" data-campaign-character-card-id="ch-b" data-campaign-character-owned-by-viewer="false"`) {
		t.Fatalf("expected other character card to keep base border: %q", body)
	}
}

func TestMountCampaignCharactersUsesViewCharacterCTAForReadOnlyCharacters(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		characters: []campaignapp.CampaignCharacter{{
			ID:         "ch-a",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			CanEdit:    false,
		}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-view-link="true"`,
		`View Character`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing read-only view marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharactersUsesViewCharacterCTAForNonDaggerheartCampaigns(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:           []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		workspaceSystem: "Pathfinder",
		characters: []campaignapp.CampaignCharacter{{
			ID:         "ch-a",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			CanEdit:    true,
		}},
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-campaign-character-view-link="true"`) {
		t.Fatalf("body missing character view CTA for non-daggerheart campaign: %q", body)
	}
}

func TestMountCampaignCharactersFailsWhenGatewayReturnsError(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{
			ID:            "c1",
			Name:          "The Guildhouse",
			CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		charactersErr: apperrors.E(apperrors.KindUnavailable, "characters unavailable"),
	}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestMountCampaignCharactersFailsClosedWhenCharacterClientMissing(t *testing.T) {
	t.Parallel()

	deps := campaigngateway.GRPCGatewayDeps{Catalog: campaigngateway.CatalogGatewayDeps{Read: campaigngateway.CatalogReadDeps{Campaign: fakeCampaignClient{}}}}
	m := New(configWithGRPCDeps(deps, modulehandler.NewTestBase(), nil))
	_, err := m.Mount()
	if err == nil {
		t.Fatalf("expected Mount() validation error")
	}
	if !strings.Contains(err.Error(), "character-reads") {
		t.Fatalf("Mount() error = %v, want character validation failure", err)
	}
}

func TestMountCampaignRoutesRenderWorkspaceOverviewMenu(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First", CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png"}}}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	paths := []string{
		routepath.AppCampaign("c1"),
		routepath.AppCampaignParticipants("c1"),
		routepath.AppCampaignCharacters("c1"),
		routepath.AppCampaignCharacter("c1", "pc1"),
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
			}
			body := rr.Body.String()
			for _, marker := range []string{
				`class="menu bg-base-200 rounded-box w-full"`,
				`href="/app/campaigns/c1"`,
				`hx-get="/app/campaigns/c1"`,
				`>Overview </a>`,
			} {
				if !strings.Contains(body, marker) {
					t.Fatalf("path %q body missing campaign menu marker %q: %q", path, marker, body)
				}
			}
			participantsIdx := strings.Index(body, `data-app-side-menu-item="/app/campaigns/c1/participants"`)
			invitesIdx := strings.Index(body, `data-app-side-menu-item="/app/campaigns/c1/invites"`)
			charactersIdx := strings.Index(body, `data-app-side-menu-item="/app/campaigns/c1/characters"`)
			sessionsIdx := strings.Index(body, `data-app-side-menu-item="/app/campaigns/c1/sessions"`)
			if participantsIdx == -1 || invitesIdx == -1 || charactersIdx == -1 || sessionsIdx == -1 {
				t.Fatalf("path %q missing campaign workspace menu items: %q", path, body)
			}
			if participantsIdx > invitesIdx {
				t.Fatalf("path %q expected participants menu item before invites menu item: %q", path, body)
			}
			if charactersIdx > sessionsIdx {
				t.Fatalf("path %q expected characters menu item before sessions menu item: %q", path, body)
			}
			if sessionsIdx > invitesIdx {
				t.Fatalf("path %q expected sessions menu item before invites menu item: %q", path, body)
			}
		})
	}
}

func TestMountCampaignOverviewHidesInvitesMenuWithoutPermission(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:  true,
			Allowed:    false,
			ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED",
		},
	}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-app-side-menu-item="/app/campaigns/c1/participants"`) {
		t.Fatalf("body missing participants menu item: %q", body)
	}
	// Invariant: invite-manage navigation must not be exposed without permission.
	if strings.Contains(body, `data-app-side-menu-item="/app/campaigns/c1/invites"`) {
		t.Fatalf("body should hide invites menu item without permission: %q", body)
	}
}

func TestMountCampaignWorkspaceCoverStyleRendersForFullAndHTMX(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{
		ID:            "c1",
		Name:          "First",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}}, modulehandler.NewTestBase(), nil))

	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	nonHTMXReq := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("c1"), nil)
	nonHTMXRR := httptest.NewRecorder()
	mount.Handler.ServeHTTP(nonHTMXRR, nonHTMXReq)
	if nonHTMXRR.Code != http.StatusOK {
		t.Fatalf("non-htmx status = %d, want %d", nonHTMXRR.Code, http.StatusOK)
	}
	body := nonHTMXRR.Body.String()
	if !strings.Contains(body, `style="background-image: url(`) {
		t.Fatalf("non-htmx body = %q, want campaign cover main style", body)
	}
	if !strings.Contains(body, `data-app-route-area="campaign-workspace"`) {
		t.Fatalf("non-htmx body = %q, want campaign workspace route metadata", body)
	}
	if !strings.Contains(body, `campaign-cover-header`) {
		t.Fatalf("non-htmx body = %q, want cover header class", body)
	}
	if strings.Contains(body, `linear-gradient(to bottom`) {
		t.Fatalf("non-htmx body unexpectedly contains overlay gradient: %q", body)
	}

	htmxReq := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("c1"), nil)
	htmxReq.Header.Set("HX-Request", "true")
	htmxRR := httptest.NewRecorder()
	mount.Handler.ServeHTTP(htmxRR, htmxReq)
	if htmxRR.Code != http.StatusOK {
		t.Fatalf("htmx status = %d, want %d", htmxRR.Code, http.StatusOK)
	}
	body = htmxRR.Body.String()
	if !strings.Contains(body, `data-app-main-background-preview="`) {
		t.Fatalf("htmx body = %q, want campaign preview background metadata", body)
	}
	if !strings.Contains(body, `data-app-main-background-full="`) {
		t.Fatalf("htmx body = %q, want campaign full background metadata", body)
	}
	if strings.Contains(body, `data-app-main-style="background-image: url(`) {
		t.Fatalf("htmx body = %q, want base style metadata without inline background image", body)
	}
	if !strings.Contains(body, `data-app-route-area="campaign-workspace"`) {
		t.Fatalf("htmx body = %q, want campaign workspace route metadata", body)
	}
	if !strings.Contains(body, `campaign-cover-header`) {
		t.Fatalf("htmx body = %q, want cover header class", body)
	}
	if strings.Contains(body, `linear-gradient(to bottom`) {
		t.Fatalf("htmx body unexpectedly contains overlay gradient: %q", body)
	}
}

func TestMountUsesWebLayoutForNonHTMX(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="main"`) {
		t.Fatalf("body = %q, want app templ main marker", body)
	}
}

func TestMountCampaignSessionDetailRendersBreadcrumbs(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:    []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		sessions: []campaignapp.CampaignSession{{ID: "s1", Name: "First Light", Status: "Active"}},
	}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignSession("c1", "s1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`class="breadcrumbs text-sm"`,
		`href="/app/campaigns"`,
		`href="/app/campaigns/c1">The Guildhouse</a>`,
		`href="/app/campaigns/c1/sessions"`,
		`>First Light</li>`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing breadcrumb marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignSessionDetailTruncatesLongBreadcrumbLabels(t *testing.T) {
	t.Parallel()

	longCampaignName := "Campaign-" + strings.Repeat("x", 64)
	longSessionID := "session-" + strings.Repeat("y", 64)
	longSessionName := "Session-" + strings.Repeat("z", 64)
	m := New(configWithGateway(fakeGateway{
		items:    []campaignapp.CampaignSummary{{ID: "c1", Name: longCampaignName}},
		sessions: []campaignapp.CampaignSession{{ID: longSessionID, Name: longSessionName, Status: "Active"}},
	}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignSession("c1", longSessionID), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `...`) {
		t.Fatalf("expected truncated breadcrumb labels with ellipsis, got %q", body)
	}
	// Invariant: breadcrumb labels must truncate long values to keep layout stable.
	if strings.Contains(body, `>`+longCampaignName+`</a>`) {
		t.Fatalf("campaign breadcrumb should be truncated, got %q", body)
	}
	// Invariant: breadcrumb labels must truncate long values to keep layout stable.
	if strings.Contains(body, `<li>`+longSessionName+`</li>`) {
		t.Fatalf("session breadcrumb should be truncated, got %q", body)
	}
}

func TestMountCampaignSessionDetailReturnsNotFoundForUnknownSession(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:    []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		sessions: []campaignapp.CampaignSession{{ID: "s1", Name: "First Light", Status: "Active"}},
	}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignSession("c1", "missing-session"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}
