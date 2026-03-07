package campaigns

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestMountServesCampaignDetailRoutes(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	paths := map[string]string{
		routepath.AppCampaign("c1"):                      "campaign-overview",
		routepath.AppCampaignEdit("c1"):                  "campaign-edit",
		routepath.AppCampaignParticipants("c1"):          "campaign-participants",
		routepath.AppCampaignParticipantEdit("c1", "p1"): "campaign-participant-edit",
		routepath.AppCampaignCharacters("c1"):            "campaign-characters",
		routepath.AppCampaignCharacter("c1", "pc1"):      "campaign-character-detail",
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

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), "", nil)
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

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		sessions: []CampaignSession{{
			ID:     "s1",
			Name:   "First Light",
			Status: "Active",
		}},
	}, modulehandler.NewTestBase(), "", nil)
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

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{
			ID:               "c1",
			Name:             "The Guildhouse",
			ParticipantCount: "2",
			CharacterCount:   "2",
		}},
		sessions: []CampaignSession{
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
	}, modulehandler.NewTestBase(), "", nil)
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
			`href="/app/campaigns/c1/sessions"`,
			`class="badge badge-sm badge-soft badge-primary">2</div>`,
			`data-app-side-menu-subitem="/app/campaigns/c1/sessions/s1"`,
			`data-app-side-menu-subitem="/app/campaigns/c1/sessions/s2"`,
		} {
			if !strings.Contains(body, marker) {
				t.Fatalf("path %q body missing sessions menu marker %q: %q", path, marker, body)
			}
		}
	}
}

func TestMountCampaignWorkspaceSessionsMenuHighlightsEntireActiveRow(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		sessions: []CampaignSession{
			{
				ID:        "s3",
				Name:      "Third Light",
				Status:    "Ended",
				StartedAt: "2026-02-03 20:00 UTC",
				EndedAt:   "2026-02-03 22:00 UTC",
			},
			{
				ID:        "s1",
				Name:      "",
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
	}, modulehandler.NewTestBase(), "", nil)
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

	s1Idx := strings.Index(body, `data-app-side-menu-subitem="/app/campaigns/c1/sessions/s1"`)
	s2Idx := strings.Index(body, `data-app-side-menu-subitem="/app/campaigns/c1/sessions/s2"`)
	s3Idx := strings.Index(body, `data-app-side-menu-subitem="/app/campaigns/c1/sessions/s3"`)
	if s1Idx == -1 || s2Idx == -1 || s3Idx == -1 {
		t.Fatalf("expected all session subitems in side menu: %q", body)
	}
	if !(s1Idx < s2Idx && s2Idx < s3Idx) {
		t.Fatalf("expected session subitems oldest-to-newest order; indexes = (%d, %d, %d)", s1Idx, s2Idx, s3Idx)
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
		`data-app-side-menu-subitem-end="End: In progress"`,
		`>Unnamed session</span>`,
	} {
		if !strings.Contains(activeRowBody, marker) {
			t.Fatalf("expected active session detail marker %q in output: %q", marker, activeRowBody)
		}
	}
}

func TestMountCampaignSessionsRouteRendersReadinessBlockers(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		sessionReadiness: CampaignSessionReadiness{
			Ready: false,
			Blockers: []CampaignSessionReadinessBlocker{
				{
					Code:    "SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED",
					Message: "Campaign readiness requires at least one AI-controlled GM participant for AI GM mode",
				},
			},
		},
	}, modulehandler.NewTestBase(), "", nil)
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

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		sessions: []CampaignSession{{
			ID:     "s1",
			Name:   "First Light",
			Status: "Active",
		}},
	}, modulehandler.NewTestBase(), "", nil)
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

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		invites: []CampaignInvite{{
			ID:              "inv-1",
			ParticipantID:   "p1",
			RecipientUserID: "user-2",
			Status:          "Pending",
		}},
	}, modulehandler.NewTestBase(), "", nil)
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
		`data-campaign-invite-participant="p1"`,
		`data-campaign-invite-recipient="user-2"`,
		`data-campaign-invite-status="Pending"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing invite marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharacterDetailRouteRendersSelectedCharacter(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		characters: []CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			AvatarURL:  "/static/avatars/aria.png",
		}},
	}, modulehandler.NewTestBase(), "", nil)
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
		`data-campaign-character-detail-name="Aria"`,
		`data-campaign-character-detail-kind="PC"`,
		`data-campaign-character-detail-controller="Ariadne"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing character detail marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharacterDetailRendersCreationLinkCard(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		characters: []CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
		}},
		characterCreationProgress: CampaignCharacterCreationProgress{
			Steps:    []CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
			NextStep: 1,
		},
		characterCreationCatalog: CampaignCharacterCreationCatalog{
			Classes:    []CatalogClass{{ID: "warrior", Name: "Warrior"}},
			Subclasses: []CatalogSubclass{{ID: "guardian", Name: "Guardian", ClassID: "warrior"}},
		},
	}, modulehandler.NewTestBase(), "", defaultTestWorkflows())
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
		`data-character-creation-link="true"`,
		`/app/campaigns/c1/characters/char-1/creation`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing workflow marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharacterDetailHidesWorkflowForNonDaggerheartCampaigns(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items:           []CampaignSummary{{ID: "c1", Name: "First"}},
		workspaceSystem: "Pathfinder",
		characters: []CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
		}},
		characterCreationProgressErr: errors.New("workflow should not be loaded for non-daggerheart systems"),
	}, modulehandler.NewTestBase(), "", nil)
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

func TestMountCampaignOverviewRendersWorkspaceDetailsAndMenu(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{
		ID:            "c1",
		Name:          "The Guildhouse",
		Theme:         "Stormbound intrigue",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}}, modulehandler.NewTestBase(), "", nil)
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

func TestMountCampaignOverviewAllowsHead(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{
		ID:            "c1",
		Name:          "The Guildhouse",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}}, modulehandler.NewTestBase(), "", nil)
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

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:  true,
			Allowed:    false,
			ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED",
		},
	}, modulehandler.NewTestBase(), "", nil)
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

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{
			ID:               "c1",
			Name:             "The Guildhouse",
			ParticipantCount: "2",
			CoverImageURL:    "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		participants: []CampaignParticipant{
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
	}, modulehandler.NewTestBase(), "", nil)
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
		`class="grid grid-cols-1 md:grid-cols-2 gap-4"`,
		`data-campaign-participant-card-id="p-a"`,
		`data-campaign-participant-name="Aria"`,
		`data-campaign-participant-role="GM"`,
		`data-campaign-participant-access="Owner"`,
		`data-campaign-participant-controller="AI"`,
		`data-image-frame="true"`,
		`data-image-skeleton="true"`,
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

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		participants: []CampaignParticipant{
			{ID: "p-a", Name: "Aria", Role: "GM", CampaignAccess: "Owner", Controller: "Human", AvatarURL: "/static/avatars/aria.png"},
			{ID: "p-b", Name: "Bram", Role: "Player", CampaignAccess: "Member", Controller: "Human", AvatarURL: "/static/avatars/bram.png"},
		},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{
			{CheckID: "p-a", Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
			{CheckID: "p-b", Evaluated: true, Allowed: false, ReasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED"},
		},
	}, modulehandler.NewTestBase(), "", nil)
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

func TestMountCampaignParticipantEditRendersForm(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		participant: CampaignParticipant{
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
	}, modulehandler.NewTestBase(), "", nil)
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
		`data-campaign-participant-edit-form="true"`,
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

func TestMountCampaignParticipantsFailsWhenGatewayReturnsError(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{
			ID:             "c1",
			Name:           "The Guildhouse",
			CharacterCount: "2",
			CoverImageURL:  "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		participantsErr: apperrors.E(apperrors.KindUnavailable, "participants unavailable"),
	}, modulehandler.NewTestBase(), "", nil)
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

	m := New()
	deps := GRPCGatewayDeps{CampaignClient: fakeCampaignClient{}}
	m = NewStableWithGateway(NewGRPCGateway(deps), modulehandler.NewTestBase(), "", nil)
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

func TestMountCampaignCharactersMenuAndPortraitGallery(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{
			ID:             "c1",
			Name:           "The Guildhouse",
			CharacterCount: "2",
			CoverImageURL:  "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		characters: []CampaignCharacter{
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
	}, modulehandler.NewTestBase(), "", nil)
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
		`data-campaign-character-create-form="true"`,
		`action="/app/campaigns/c1/characters/create"`,
		`data-campaign-character-create-name="true"`,
		`Add Character`,
		`class="grid grid-cols-1 md:grid-cols-2 gap-4"`,
		`data-campaign-character-card-id="ch-a"`,
		`data-campaign-character-name="Aria"`,
		`href="/app/campaigns/c1/characters/ch-a"`,
		`data-campaign-character-detail-link="true"`,
		`data-campaign-character-creation-entry="false"`,
		`data-campaign-character-kind="PC"`,
		`data-campaign-character-controller="Ariadne"`,
		`data-image-frame="true"`,
		`data-image-skeleton="true"`,
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

func TestMountCampaignCharactersEmptyStateStillShowsCreateForm(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "The Guildhouse"}}}, modulehandler.NewTestBase(), "", nil)
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
		`data-campaign-character-create-form="true"`,
		`action="/app/campaigns/c1/characters/create"`,
		`Add Character`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing empty-state create marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharactersShowsCreationEntryForEditableDaggerheartCharacters(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{
			ID:            "c1",
			Name:          "The Guildhouse",
			CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		characters: []CampaignCharacter{{
			ID:         "ch-a",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
		}},
		batchAuthorizationDecisions: []campaignapp.AuthorizationDecision{{CheckID: "ch-a", Evaluated: true, Allowed: true}},
	}, modulehandler.NewTestBase(), "", nil)
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
		`data-campaign-character-creation-entry="true"`,
		`Open creation workflow`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing creation-entry marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharactersHidesCreationEntryForReadOnlyCharacters(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		characters: []CampaignCharacter{{
			ID:         "ch-a",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			CanEdit:    false,
		}},
	}, modulehandler.NewTestBase(), "", nil)
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
	if strings.Contains(body, `data-campaign-character-creation-entry="true"`) {
		t.Fatalf("body unexpectedly contains editable creation entry: %q", body)
	}
	for _, marker := range []string{
		`data-campaign-character-creation-entry="false"`,
		`View details`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing read-only entry marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCharactersHidesCreationEntryForNonDaggerheartCampaigns(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items:           []CampaignSummary{{ID: "c1", Name: "The Guildhouse"}},
		workspaceSystem: "Pathfinder",
		characters: []CampaignCharacter{{
			ID:         "ch-a",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			CanEdit:    true,
		}},
	}, modulehandler.NewTestBase(), "", nil)
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
	if strings.Contains(body, `data-campaign-character-creation-entry="true"`) {
		t.Fatalf("body unexpectedly contains non-daggerheart creation entry: %q", body)
	}
	if !strings.Contains(body, `data-campaign-character-creation-entry="false"`) {
		t.Fatalf("body missing fallback detail entry for non-daggerheart campaign: %q", body)
	}
}

func TestMountCampaignCharactersFailsWhenGatewayReturnsError(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{
			ID:            "c1",
			Name:          "The Guildhouse",
			CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		charactersErr: apperrors.E(apperrors.KindUnavailable, "characters unavailable"),
	}, modulehandler.NewTestBase(), "", nil)
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

	m := New()
	deps := GRPCGatewayDeps{CampaignClient: fakeCampaignClient{}}
	m = NewStableWithGateway(NewGRPCGateway(deps), modulehandler.NewTestBase(), "", nil)
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

func TestMountCampaignRoutesRenderWorkspaceOverviewMenu(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First", CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png"}}}, modulehandler.NewTestBase(), "", nil)
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
		})
	}
}

func TestMountCampaignWorkspaceCoverStyleRendersForFullAndHTMX(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{
		ID:            "c1",
		Name:          "First",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}}, modulehandler.NewTestBase(), "", nil)
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
	if !strings.Contains(body, `data-app-main-style="background-image: url(`) {
		t.Fatalf("htmx body = %q, want campaign main style metadata", body)
	}
	if !strings.Contains(body, `data-app-route-area="campaign-workspace"`) {
		t.Fatalf("htmx body = %q, want campaign workspace route metadata", body)
	}
	if strings.Contains(body, `linear-gradient(to bottom`) {
		t.Fatalf("htmx body unexpectedly contains overlay gradient: %q", body)
	}
}

func TestMountUsesWebLayoutForNonHTMX(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
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

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "The Guildhouse"}}}, modulehandler.NewTestBase(), "", nil)
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
		`<a href="/app/campaigns/c1">The Guildhouse</a>`,
		`href="/app/campaigns/c1/sessions"`,
		`<li>s1</li>`,
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
	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: longCampaignName}}}, modulehandler.NewTestBase(), "", nil)
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
	if strings.Contains(body, `<li>`+longSessionID+`</li>`) {
		t.Fatalf("session breadcrumb should be truncated, got %q", body)
	}
}
