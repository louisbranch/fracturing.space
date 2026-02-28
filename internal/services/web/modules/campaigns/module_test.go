package campaigns

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/icons"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/webctx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestModuleIDReturnsCampaigns(t *testing.T) {
	t.Parallel()

	if got := New().ID(); got != "campaigns" {
		t.Fatalf("ID() = %q, want %q", got, "campaigns")
	}
}

func TestMountServesCampaignsGet(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}, {ID: "c2", Name: "Second"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if mount.Prefix != routepath.CampaignsPrefix {
		t.Fatalf("prefix = %q, want %q", mount.Prefix, routepath.CampaignsPrefix)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
	body := rr.Body.String()
	if !strings.Contains(body, "First") || !strings.Contains(body, "Second") || !strings.Contains(body, `data-campaign-id="c1"`) {
		t.Fatalf("body = %q, want campaign list html", body)
	}
}

func TestMountReturnsServiceUnavailableWhenGatewayNotConfigured(t *testing.T) {
	t.Parallel()

	m := New()
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: default module wiring must fail closed when campaigns backend is absent.
	if strings.Contains(body, `data-campaign-id="starter"`) {
		t.Fatalf("body unexpectedly rendered static campaign list without backend: %q", body)
	}
}

func TestMountRejectsCampaignsNonGet(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodPost, routepath.CampaignsPrefix+"123", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestMountMapsCampaignGatewayErrorToHTTPStatus(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{err: apperrors.E(apperrors.KindUnauthorized, "missing session")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMountCampaignsGRPCNotFoundRendersAppErrorPage(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{err: status.Error(codes.NotFound, "campaign not found")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: backend transport errors must never leak raw gRPC strings to user-facing pages.
	if strings.Contains(body, "rpc error:") {
		t.Fatalf("body leaked raw grpc error: %q", body)
	}
}

func TestMountCampaignsInternalErrorRendersServerErrorPage(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{err: errors.New("boom")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
}

func TestMountCampaignsGRPCNotFoundHTMXRendersErrorFragment(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{err: status.Error(codes.NotFound, "campaign not found")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: HTMX failures must swap a fragment and not a full document.
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx error fragment without document wrapper")
	}
}

func TestMountCampaignDetailMissingCampaignReturnsNotFound(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("c999"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: missing campaign detail routes should render the shared not-found page, not fallback pseudo-content.
	if strings.Contains(body, `data-campaign-overview-name="c999"`) {
		t.Fatalf("body unexpectedly rendered fallback campaign workspace: %q", body)
	}
}

func TestMountCampaignsUnknownSubpathRendersNotFoundPage(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix+"c1/unknown", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: unknown app routes should use the shared not-found page, not net/http plain text.
	if strings.Contains(body, "404 page not found") {
		t.Fatalf("body unexpectedly rendered plain 404 text: %q", body)
	}
}

func TestMountCampaignsLegacyChatSubpathRendersNotFoundPage(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/c1/chat", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
}

func TestMountUsesDependenciesCampaignClientWhenGatewayNotProvided(t *testing.T) {
	t.Parallel()

	deps := module.Dependencies{
		CampaignClient: fakeCampaignClient{
			response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "remote-1", Name: "Remote Campaign"}}},
		},
	}
	m := NewWithGateway(NewGRPCGateway(deps))
	mount, err := m.Mount(deps)
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if body := rr.Body.String(); !strings.Contains(body, "Remote Campaign") {
		t.Fatalf("body = %q, want remote campaign", body)
	}
}

func TestMountCampaignsPageRendersCardGridWithCover(t *testing.T) {
	t.Parallel()

	deps := module.Dependencies{
		CampaignClient: fakeCampaignClient{
			response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{
				{
					Id:               "camp-old",
					Name:             "Older Campaign",
					ParticipantCount: 4,
					CharacterCount:   1,
					CreatedAt:        timestamppb.New(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)),
				},
				{
					Id:               "camp-new",
					Name:             "Newer Campaign",
					CoverAssetId:     "abandoned_castle_courtyard",
					ParticipantCount: 12,
					CharacterCount:   7,
					CreatedAt:        timestamppb.New(time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC)),
				},
			}},
		},
	}
	m := NewWithGateway(NewGRPCGateway(deps))

	mount, err := m.Mount(deps)
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`class="grid grid-cols-1 md:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4"`,
		`<a href="/app/campaigns/camp-new" class="group block"><img`,
		`/static/campaign-covers/abandoned_castle_courtyard.png`,
		`Participants: 12`,
		`Characters: 7`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
	newerIdx := strings.Index(body, `href="/app/campaigns/camp-new"`)
	olderIdx := strings.Index(body, `href="/app/campaigns/camp-old"`)
	if newerIdx == -1 || olderIdx == -1 {
		t.Fatalf("expected both campaigns in output")
	}
	if newerIdx > olderIdx {
		t.Fatalf("expected newer campaign to render before older campaign")
	}
}

func TestMountCampaignsPageEscapesCampaignIDsInCardLinks(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{
		ID:               "camp/1",
		Name:             "Escaped Campaign",
		ParticipantCount: "1",
		CharacterCount:   "1",
	}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `href="/app/campaigns/camp%2F1"`) {
		t.Fatalf("body missing escaped campaign route: %q", body)
	}
}

func TestMountCampaignsPageRendersCardIconsFromCatalog(t *testing.T) {
	t.Parallel()

	deps := module.Dependencies{
		CampaignClient: fakeCampaignClient{
			response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{
				{
					Id:               "camp-new",
					Name:             "Newer Campaign",
					ParticipantCount: 12,
					CharacterCount:   7,
					CreatedAt:        timestamppb.New(time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC)),
				},
			}},
		},
	}
	m := NewWithGateway(NewGRPCGateway(deps))

	mount, err := m.Mount(deps)
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	participantHref := `href="#` + icons.LucideSymbolID(icons.LucideNameOrDefault(commonv1.IconId_ICON_ID_PARTICIPANT)) + `"`
	characterHref := `href="#` + icons.LucideSymbolID(icons.LucideNameOrDefault(commonv1.IconId_ICON_ID_CHARACTER)) + `"`
	for _, marker := range []string{
		`Participants: 12`,
		`Characters: 7`,
		participantHref,
		characterHref,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
}

func TestMountServesCampaignsGetWithEmptyList(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<h1 class="mb-0">Campaigns</h1>`,
		`href="/app/campaigns/new"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
}

func TestMountServesCampaignDetailRoutes(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	paths := map[string]string{
		routepath.AppCampaign("c1"):                 "campaign-overview",
		routepath.AppCampaignSessions("c1"):         "campaign-sessions",
		routepath.AppCampaignParticipants("c1"):     "campaign-participants",
		routepath.AppCampaignCharacters("c1"):       "campaign-characters",
		routepath.AppCampaignInvites("c1"):          "campaign-invites",
		routepath.AppCampaignCharacter("c1", "pc1"): "campaign-character-detail",
		routepath.AppCampaignSession("c1", "s1"):    "campaign-session-detail",
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

	m := NewWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		sessions: []CampaignSession{{
			ID:     "s1",
			Name:   "First Light",
			Status: "Active",
		}},
	})
	mount, err := m.Mount(module.Dependencies{})
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

func TestMountCampaignSessionDetailRouteRendersSelectedSession(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		sessions: []CampaignSession{{
			ID:     "s1",
			Name:   "First Light",
			Status: "Active",
		}},
	})
	mount, err := m.Mount(module.Dependencies{})
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

	m := NewWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		invites: []CampaignInvite{{
			ID:              "inv-1",
			ParticipantID:   "p1",
			RecipientUserID: "user-2",
			Status:          "Pending",
		}},
	})
	mount, err := m.Mount(module.Dependencies{})
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

	m := NewWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		characters: []CampaignCharacter{{
			ID:         "char-1",
			Name:       "Aria",
			Kind:       "PC",
			Controller: "Ariadne",
			AvatarURL:  "/static/avatars/aria.png",
		}},
	})
	mount, err := m.Mount(module.Dependencies{})
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

func TestMountCampaignGameRouteRendersDedicatedDrawerChrome(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{
		ID:            "c1",
		Name:          "The Guildhouse",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}})
	mount, err := m.Mount(module.Dependencies{ChatFallbackPort: "8086"})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignGame("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-chat-page="true"`,
		`class="drawer lg:drawer-open min-h-[100dvh] campaign-chat-drawer"`,
		`class="drawer-side is-drawer-close:overflow-visible"`,
		`class="chat-drawer-shell flex min-h-full flex-col items-start border-e border-base-300 bg-base-200"`,
		`class="drawer-overlay chat-drawer-overlay lg:hidden"`,
		`data-campaign-chat-title="The Guildhouse Game"`,
		`class="px-2 text-lg font-bold"`,
		`href="/app/campaigns/c1"`,
		`data-chat-fallback-port="8086"`,
		`id="chat-messages"`,
		`src="/static/campaign-chat.js"`,
		`class="chat-drawer-icon-open size-5"`,
		`class="chat-drawer-icon-close size-5"`,
		`class="chat-drawer-link-label"`,
		`panel-left-open`,
		`panel-right-close`,
		`<symbol id="lucide-panel-left-open"`,
		`<symbol id="lucide-panel-right-close"`,
		`class="grid grid-cols-1 gap-4 lg:grid-cols-2"`,
		`class="card border border-base-300 bg-base-100 shadow-xl"`,
		`<span class="chat-drawer-link-label">Campaign</span>`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing chat page marker %q: %q", marker, body)
		}
	}
	// Invariant: chat title should stay near the left toggle, not centered.
	if strings.Contains(body, `navbar-center`) {
		t.Fatalf("chat route unexpectedly centers navbar title: %q", body)
	}
	// Invariant: dedicated chat route must not render default app chrome shell wrappers.
	if strings.Contains(body, `id="main"`) || strings.Contains(body, `data-nav-item="true"`) {
		t.Fatalf("chat route unexpectedly rendered app chrome: %q", body)
	}
}

func TestMountCampaignGameRouteHTMXRedirectsToFullPage(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "The Guildhouse"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignGame("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignGame("c1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignGame("c1"))
	}
}

func TestMountCampaignOverviewRendersWorkspaceDetailsAndMenu(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{
		ID:            "c1",
		Name:          "The Guildhouse",
		Theme:         "Stormbound intrigue",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}})
	mount, err := m.Mount(module.Dependencies{})
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
		`>Overview</a>`,
		`data-campaign-overview-name="The Guildhouse"`,
		`data-campaign-overview-theme="Stormbound intrigue"`,
		`data-campaign-overview-system=`,
		`data-campaign-overview-gm-mode=`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing campaign workspace marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignOverviewAllowsHead(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{
		ID:            "c1",
		Name:          "The Guildhouse",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}})
	mount, err := m.Mount(module.Dependencies{})
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

func TestMountCampaignParticipantsMenuAndPortraitGallery(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{
		items: []CampaignSummary{{
			ID:            "c1",
			Name:          "The Guildhouse",
			CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
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
	})
	mount, err := m.Mount(module.Dependencies{})
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
		`src="/static/avatars/aria.png"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing participants gallery marker %q: %q", marker, body)
		}
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

func TestMountCampaignParticipantsFailsWhenGatewayReturnsError(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{
		items: []CampaignSummary{{
			ID:            "c1",
			Name:          "The Guildhouse",
			CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		participantsErr: apperrors.E(apperrors.KindUnavailable, "participants unavailable"),
	})
	mount, err := m.Mount(module.Dependencies{})
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
	deps := module.Dependencies{CampaignClient: fakeCampaignClient{}}
	m = NewWithGateway(NewGRPCGateway(deps))
	mount, err := m.Mount(deps)
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

	m := NewWithGateway(fakeGateway{
		items: []CampaignSummary{{
			ID:            "c1",
			Name:          "The Guildhouse",
			CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
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
	})
	mount, err := m.Mount(module.Dependencies{})
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
		`class="grid grid-cols-1 md:grid-cols-2 gap-4"`,
		`data-campaign-character-card-id="ch-a"`,
		`data-campaign-character-name="Aria"`,
		`data-campaign-character-kind="PC"`,
		`data-campaign-character-controller="Ariadne"`,
		`src="/static/avatars/aria.png"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing characters gallery marker %q: %q", marker, body)
		}
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

func TestMountCampaignCharactersFailsWhenGatewayReturnsError(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{
		items: []CampaignSummary{{
			ID:            "c1",
			Name:          "The Guildhouse",
			CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
		}},
		charactersErr: apperrors.E(apperrors.KindUnavailable, "characters unavailable"),
	})
	mount, err := m.Mount(module.Dependencies{})
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
	deps := module.Dependencies{CampaignClient: fakeCampaignClient{}}
	m = NewWithGateway(NewGRPCGateway(deps))
	mount, err := m.Mount(deps)
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

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First", CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	paths := []string{
		routepath.AppCampaign("c1"),
		routepath.AppCampaignSessions("c1"),
		routepath.AppCampaignSession("c1", "s1"),
		routepath.AppCampaignParticipants("c1"),
		routepath.AppCampaignCharacters("c1"),
		routepath.AppCampaignCharacter("c1", "pc1"),
		routepath.AppCampaignInvites("c1"),
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
				`>Overview</a>`,
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

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{
		ID:            "c1",
		Name:          "First",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}})
	mount, err := m.Mount(module.Dependencies{})
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
	if strings.Contains(body, `linear-gradient(to bottom`) {
		t.Fatalf("htmx body unexpectedly contains overlay gradient: %q", body)
	}
}

func TestMountUsesWebLayoutForNonHTMX(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}})
	mount, err := m.Mount(module.Dependencies{})
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

func TestMountCampaignsPageRendersHeadingWithStartLink(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<h1 class="mb-0">Campaigns</h1>`,
		`href="/app/campaigns/new"`,
		`>Start a new Campaign</a>`,
		`data-campaign-id="c1"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing campaigns contract marker %q", marker)
		}
	}
}

func TestMountCampaignsPageOmitsBreadcrumbsAtRoot(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	// Invariant: page roots (campaigns index) should not render breadcrumb trails.
	if strings.Contains(body, `class="breadcrumbs text-sm"`) {
		t.Fatalf("expected no breadcrumbs on campaigns root, got %q", body)
	}
}

func TestMountCampaignsHTMXRendersHeadingWithStartLink(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<h1 class="mb-0">Campaigns</h1>`,
		`href="/app/campaigns/new"`,
		`data-campaign-id="c1"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing htmx campaigns contract marker %q: %q", marker, body)
		}
	}
	// Invariant: HTMX requests must return a fragment, not a full HTML document envelope.
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx fragment body without document wrapper")
	}
}

func TestMountCampaignStartNewGetRendersChoiceCards(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignsNew, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<h1 class="mb-0">New Campaign</h1>`,
		`data-campaign-start-option="browse"`,
		`data-campaign-start-divider="or"`,
		`class="divider lg:divider-horizontal`,
		`disabled aria-disabled="true"`,
		`data-campaign-start-option="scratch"`,
		`href="/app/campaigns/create"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing start-choice marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCreateGetRendersCreateForm(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignsCreate, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`href="/app/campaigns"`,
		`<h1 class="mb-0">Create Campaign</h1>`,
		`<form method="post" action="/app/campaigns/create"`,
		`name="name"`,
		`name="system"`,
		`name="gm_mode"`,
		`name="theme_prompt"`,
		`<option value="daggerheart" selected>`,
		`<option value="human" selected>`,
		`<option value="ai">`,
		`<option value="hybrid">`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
}

func TestMountCampaignCreatePostCreatesCampaignAndRedirects(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}, createdCampaignID: "camp-777"})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	form := url.Values{
		"name":         {"New Campaign"},
		"system":       {"daggerheart"},
		"gm_mode":      {"ai"},
		"theme_prompt": {"Misty marshes"},
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignsCreate, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaign("camp-777") {
		t.Fatalf("location = %q, want %q", got, routepath.AppCampaign("camp-777"))
	}
}

func TestMountCampaignCreatePostUsesHTMXRedirect(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{createCampaignResult: CreateCampaignResult{CampaignID: "camp-htmx"}}
	m := NewWithGateway(gateway)
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	form := url.Values{"name": {"New Campaign"}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignsCreate, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaign("camp-htmx") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaign("camp-htmx"))
	}
}

func TestMountCampaignCreatePostAppliesDefaults(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{createCampaignResult: CreateCampaignResult{CampaignID: "camp-1"}}
	m := NewWithGateway(gateway)
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	form := url.Values{"name": {"New Campaign"}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignsCreate, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := gateway.lastCreateInput.System; got != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("System = %v, want %v", got, commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	}
	if got := gateway.lastCreateInput.GMMode; got != statev1.GmMode_HUMAN {
		t.Fatalf("GMMode = %v, want %v", got, statev1.GmMode_HUMAN)
	}
}

func TestMountCampaignCreatePostRejectsEmptyName(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}, createdCampaignID: "camp-777"})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignsCreate, strings.NewReader("name=   "))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	// Invariant: invalid create submissions must not redirect to a campaign route.
	if got := rr.Header().Get("Location"); got != "" {
		t.Fatalf("location = %q, want empty", got)
	}
}

func TestMountCampaignCreateValidationErrorIsLocalizedForPTBR(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{})
	mount, err := m.Mount(module.Dependencies{ResolveLanguage: func(*http.Request) string { return "pt-BR" }})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignsCreate, strings.NewReader("name=   "))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "Nome da campanha é obrigatório") {
		t.Fatalf("expected localized campaign-name validation error, got %q", rr.Body.String())
	}
}

func TestMountCampaignCreatePostUsesResolvedLanguageLocaleWhenUsingDependenciesClient(t *testing.T) {
	t.Parallel()

	client := &capturingCampaignClient{}
	deps := module.Dependencies{
		CampaignClient:  client,
		ResolveLanguage: func(*http.Request) string { return "pt-BR" },
	}
	m := NewWithGateway(NewGRPCGateway(deps))
	mount, err := m.Mount(deps)
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	form := url.Values{"name": {"Nova Campanha"}, "system": {"daggerheart"}, "gm_mode": {"human"}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignsCreate, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if client.lastCreateReq == nil {
		t.Fatalf("expected campaign client CreateCampaign call")
	}
	if got := client.lastCreateReq.GetLocale(); got != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("CreateCampaign locale = %v, want %v", got, commonv1.Locale_LOCALE_PT_BR)
	}
}

func TestMountCampaignCreateRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, routepath.AppCampaignsCreate, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodGet+", HEAD, "+http.MethodPost {
		t.Fatalf("Allow = %q, want %q", got, http.MethodGet+", HEAD, "+http.MethodPost)
	}
}

func TestMountCampaignCreatePostRejectsInvalidSystemAndGMMode(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	tests := []struct {
		name string
		form url.Values
	}{
		{name: "invalid system", form: url.Values{"name": {"New"}, "system": {"invalid-system"}}},
		{name: "invalid gm mode", form: url.Values{"name": {"New"}, "gm_mode": {"invalid-gm"}}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignsCreate, strings.NewReader(tc.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestMountCampaignCreatePostMapsServiceErrorStatus(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{createErr: apperrors.E(apperrors.KindForbidden, "forbidden")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	form := url.Values{"name": {"New Campaign"}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignsCreate, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestMountCampaignCreatePostReturnsBadRequestOnFormParseFailure(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignsCreate, nil)
	req.Body = io.NopCloser(errorReader{err: errors.New("read failed")})
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestMountCampaignSessionDetailRendersBreadcrumbs(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "The Guildhouse"}}})
	mount, err := m.Mount(module.Dependencies{})
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
	m := NewWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: longCampaignName}}})
	mount, err := m.Mount(module.Dependencies{})
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

func managerMutationDeps() module.Dependencies {
	return module.Dependencies{ResolveUserID: func(*http.Request) string { return "user-123" }}
}

func TestMountSessionStartUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := NewWithGateway(managerMutationGateway())
	mount, _ := m.Mount(managerMutationDeps())
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignSessionStart("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignSessions("c1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignSessions("c1"))
	}
}

func TestMountSessionEndUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := NewWithGateway(managerMutationGateway())
	mount, _ := m.Mount(managerMutationDeps())
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignSessionEnd("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignSessions("c1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignSessions("c1"))
	}
}

func TestMountParticipantUpdateUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := NewWithGateway(managerMutationGateway())
	mount, _ := m.Mount(managerMutationDeps())
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignParticipantUpdate("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignParticipants("c1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignParticipants("c1"))
	}
}

func TestMountSessionStartRedirectsForNonHTMX(t *testing.T) {
	t.Parallel()
	m := NewWithGateway(managerMutationGateway())
	mount, _ := m.Mount(managerMutationDeps())
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignSessionStart("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignSessions("c1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignSessions("c1"))
	}
}

func TestMountCharacterCreateUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := NewWithGateway(managerMutationGateway())
	mount, _ := m.Mount(managerMutationDeps())
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignCharacters("c1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignCharacters("c1"))
	}
}

func TestMountCharacterUpdateUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := NewWithGateway(managerMutationGateway())
	mount, _ := m.Mount(managerMutationDeps())
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterUpdate("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignCharacters("c1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignCharacters("c1"))
	}
}

func TestMountCharacterControlUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := NewWithGateway(managerMutationGateway())
	mount, _ := m.Mount(managerMutationDeps())
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterControl("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignCharacters("c1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignCharacters("c1"))
	}
}

func TestMountInviteCreateUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := NewWithGateway(managerMutationGateway())
	mount, _ := m.Mount(managerMutationDeps())
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignInviteCreate("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignInvites("c1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignInvites("c1"))
	}
}

func TestMountInviteRevokeUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := NewWithGateway(managerMutationGateway())
	mount, _ := m.Mount(managerMutationDeps())
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignInviteRevoke("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignInvites("c1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignInvites("c1"))
	}
}

func TestHandleMutationGuardsUnsupportedKind(t *testing.T) {
	t.Parallel()

	h := newHandlers(newService(fakeGateway{}), module.Dependencies{})

	unknownReq := httptest.NewRequest(http.MethodPost, routepath.AppCampaign("c1"), nil)
	unknownRR := httptest.NewRecorder()
	h.handleMutation(unknownRR, unknownReq, detailRoute{campaignID: "c1", kind: detailRouteKind("unknown")})
	if unknownRR.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", unknownRR.Code, http.StatusNotFound)
	}
}

func TestHandleMutationPropagatesResolvedUserID(t *testing.T) {
	t.Parallel()

	gateway := &mutationContextGateway{fakeGateway: fakeGateway{participants: []CampaignParticipant{{ID: "p-manager", UserID: "user-123", CampaignAccess: "Manager"}}}}
	h := newHandlers(newService(gateway), module.Dependencies{ResolveUserID: func(*http.Request) string { return "user-123" }})
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignSessionStart("c1"), nil)
	rr := httptest.NewRecorder()
	h.handleMutation(rr, req, detailRoute{campaignID: "c1", kind: detailSessionStart})

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignSessions("c1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignSessions("c1"))
	}
	if gateway.startSessionUserID != "user-123" {
		t.Fatalf("startSession user id = %q, want %q", gateway.startSessionUserID, "user-123")
	}
}

func TestRequestContextWithUserIDBehavior(t *testing.T) {
	t.Parallel()

	h := newHandlers(newService(fakeGateway{}), module.Dependencies{})
	if got := webctx.WithResolvedUserID(nil, h.deps.resolveUserID); got == nil {
		t.Fatalf("expected background context for nil request")
	}

	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	ctx := webctx.WithResolvedUserID(req, h.deps.resolveUserID)
	if md, ok := metadata.FromOutgoingContext(ctx); ok && len(md.Get(grpcmeta.UserIDHeader)) > 0 {
		t.Fatalf("unexpected user metadata when resolver is nil")
	}

	h = newHandlers(newService(fakeGateway{}), module.Dependencies{ResolveUserID: func(*http.Request) string { return "user-123" }})
	ctx = webctx.WithResolvedUserID(req, h.deps.resolveUserID)
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata")
	}
	if got := md.Get(grpcmeta.UserIDHeader); len(got) != 1 || got[0] != "user-123" {
		t.Fatalf("user metadata = %v, want [user-123]", got)
	}
}

func TestParseAppGameSystemAndGmMode(t *testing.T) {
	t.Parallel()

	if system, ok := parseAppGameSystem("daggerheart"); !ok || system != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("parseAppGameSystem daggerheart = (%v, %v)", system, ok)
	}
	if _, ok := parseAppGameSystem("unknown"); ok {
		t.Fatalf("expected unknown game system to fail parse")
	}

	if mode, ok := parseAppGmMode("ai"); !ok || mode != statev1.GmMode_AI {
		t.Fatalf("parseAppGmMode ai = (%v, %v)", mode, ok)
	}
	if mode, ok := parseAppGmMode("hybrid"); !ok || mode != statev1.GmMode_HYBRID {
		t.Fatalf("parseAppGmMode hybrid = (%v, %v)", mode, ok)
	}
	if _, ok := parseAppGmMode("invalid"); ok {
		t.Fatalf("expected invalid gm mode to fail parse")
	}
}

func TestCampaignDetailBreadcrumbsFallbackToCampaignID(t *testing.T) {
	t.Parallel()

	route := detailRoute{campaignID: "camp-1", kind: detailOverview}
	breadcrumbs := campaignDetailBreadcrumbs(route, "   ", nil)
	if len(breadcrumbs) != 2 {
		t.Fatalf("len(breadcrumbs) = %d, want 2", len(breadcrumbs))
	}
	if breadcrumbs[1].Label != "camp-1" {
		t.Fatalf("campaign breadcrumb label = %q, want %q", breadcrumbs[1].Label, "camp-1")
	}
}

func TestWriteCampaignHTMLHandlesRenderFailure(t *testing.T) {
	t.Parallel()

	h := newHandlers(newService(fakeGateway{}), module.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	rr := httptest.NewRecorder()

	h.writeCampaignHTML(rr, req, "Campaigns", campaignsListHeader(nil), webtemplates.AppMainLayoutOptions{}, failingCampaignComponent{err: errors.New("render failed")})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Campaigns | Fracturing.Space") {
		t.Fatalf("body = %q, want campaign shell title", body)
	}
	// Invariant: template/render failures must not leak internal error details to end users.
	if strings.Contains(body, "render failed") {
		t.Fatalf("body leaked internal render error: %q", body)
	}
}

func TestGRPCGatewayCampaignNameReturnsEmptyWhenCampaignMissing(t *testing.T) {
	t.Parallel()

	g := grpcGateway{client: fakeCampaignClient{getResp: &statev1.GetCampaignResponse{Campaign: nil}}}
	name, err := g.CampaignName(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("CampaignName() error = %v", err)
	}
	if name != "" {
		t.Fatalf("name = %q, want empty", name)
	}
}

func TestGRPCGatewayCreateCampaignRejectsEmptyCampaignID(t *testing.T) {
	t.Parallel()

	g := grpcGateway{client: fakeCampaignClient{createResp: &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{}}}}
	_, err := g.CreateCampaign(context.Background(), CreateCampaignInput{Name: "New", System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, GMMode: statev1.GmMode_HUMAN})
	if err == nil {
		t.Fatalf("expected empty campaign id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestGRPCGatewayMutationMethodsReturnUnavailable(t *testing.T) {
	t.Parallel()

	g := grpcGateway{}
	tests := []struct {
		name string
		err  error
	}{
		{name: "start session", err: g.StartSession(context.Background(), "c1")},
		{name: "end session", err: g.EndSession(context.Background(), "c1")},
		{name: "update participants", err: g.UpdateParticipants(context.Background(), "c1")},
		{name: "create character", err: g.CreateCharacter(context.Background(), "c1")},
		{name: "update character", err: g.UpdateCharacter(context.Background(), "c1")},
		{name: "control character", err: g.ControlCharacter(context.Background(), "c1")},
		{name: "create invite", err: g.CreateInvite(context.Background(), "c1")},
		{name: "revoke invite", err: g.RevokeInvite(context.Background(), "c1")},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.err == nil {
				t.Fatalf("expected unavailable error")
			}
			if got := apperrors.HTTPStatus(tc.err); got != http.StatusServiceUnavailable {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
			}
		})
	}
}

func TestGRPCGatewayCampaignSessionsMapsSessionRows(t *testing.T) {
	t.Parallel()

	g := grpcGateway{sessionClient: fakeSessionClient{response: &statev1.ListSessionsResponse{Sessions: []*statev1.Session{{
		Id:         "s1",
		CampaignId: "c1",
		Name:       "First Light",
		Status:     statev1.SessionStatus_SESSION_ACTIVE,
		UpdatedAt:  timestamppb.New(time.Date(2026, 2, 24, 18, 0, 0, 0, time.UTC)),
	}}}}}

	sessions, err := g.CampaignSessions(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}
	if sessions[0].ID != "s1" || sessions[0].Name != "First Light" || sessions[0].Status != "Active" {
		t.Fatalf("sessions[0] = %+v, want mapped session fields", sessions[0])
	}
	if sessions[0].UpdatedAt == "" {
		t.Fatalf("expected non-empty updated time")
	}
}

func TestGRPCGatewayCampaignInvitesMapsInviteRows(t *testing.T) {
	t.Parallel()

	g := grpcGateway{inviteClient: fakeInviteClient{response: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{{
		Id:              "inv-1",
		CampaignId:      "c1",
		ParticipantId:   "p1",
		RecipientUserId: "user-2",
		Status:          statev1.InviteStatus_PENDING,
	}}}}}

	invites, err := g.CampaignInvites(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignInvites() error = %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("len(invites) = %d, want 1", len(invites))
	}
	if invites[0].ID != "inv-1" || invites[0].ParticipantID != "p1" || invites[0].RecipientUserID != "user-2" || invites[0].Status != "Pending" {
		t.Fatalf("invites[0] = %+v, want mapped invite fields", invites[0])
	}
}

func TestGRPCGatewayCampaignSessionsFailsClosedWhenSessionClientMissing(t *testing.T) {
	t.Parallel()

	g := grpcGateway{}
	_, err := g.CampaignSessions(context.Background(), "c1")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestGRPCGatewayCampaignInvitesFailsClosedWhenInviteClientMissing(t *testing.T) {
	t.Parallel()

	g := grpcGateway{}
	_, err := g.CampaignInvites(context.Background(), "c1")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

type fakeGateway struct {
	items             []CampaignSummary
	participants      []CampaignParticipant
	participantsErr   error
	characters        []CampaignCharacter
	charactersErr     error
	sessions          []CampaignSession
	sessionsErr       error
	invites           []CampaignInvite
	invitesErr        error
	err               error
	createErr         error
	createdCampaignID string
}

type mutationContextGateway struct {
	fakeGateway
	startSessionUserID string
}

func (g *mutationContextGateway) StartSession(ctx context.Context, _ string) error {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		g.startSessionUserID = ""
		return nil
	}
	values := md.Get(grpcmeta.UserIDHeader)
	if len(values) == 0 {
		g.startSessionUserID = ""
		return nil
	}
	g.startSessionUserID = values[0]
	return nil
}

func (f fakeGateway) ListCampaigns(context.Context) ([]CampaignSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func (f fakeGateway) CampaignName(_ context.Context, campaignID string) (string, error) {
	campaignID = strings.TrimSpace(campaignID)
	for _, item := range f.items {
		if strings.TrimSpace(item.ID) != campaignID {
			continue
		}
		return strings.TrimSpace(item.Name), nil
	}
	return "", nil
}

func (f fakeGateway) CampaignWorkspace(_ context.Context, campaignID string) (CampaignWorkspace, error) {
	campaignID = strings.TrimSpace(campaignID)
	for _, item := range f.items {
		if strings.TrimSpace(item.ID) != campaignID {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = campaignID
		}
		return CampaignWorkspace{
			ID:            campaignID,
			Name:          name,
			Theme:         strings.TrimSpace(item.Theme),
			System:        "Daggerheart",
			GMMode:        "Human",
			CoverImageURL: strings.TrimSpace(item.CoverImageURL),
		}, nil
	}
	return CampaignWorkspace{}, apperrors.E(apperrors.KindNotFound, "campaign not found")
}

func (f fakeGateway) CampaignParticipants(context.Context, string) ([]CampaignParticipant, error) {
	if f.participantsErr != nil {
		return nil, f.participantsErr
	}
	return f.participants, nil
}

func (f fakeGateway) CampaignCharacters(context.Context, string) ([]CampaignCharacter, error) {
	if f.charactersErr != nil {
		return nil, f.charactersErr
	}
	return f.characters, nil
}

func (f fakeGateway) CampaignSessions(context.Context, string) ([]CampaignSession, error) {
	if f.sessionsErr != nil {
		return nil, f.sessionsErr
	}
	return f.sessions, nil
}

func (f fakeGateway) CampaignInvites(context.Context, string) ([]CampaignInvite, error) {
	if f.invitesErr != nil {
		return nil, f.invitesErr
	}
	return f.invites, nil
}

func (f fakeGateway) CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error) {
	if f.createErr != nil {
		return CreateCampaignResult{}, f.createErr
	}
	createdID := strings.TrimSpace(f.createdCampaignID)
	if createdID == "" {
		createdID = "created"
	}
	return CreateCampaignResult{CampaignID: createdID}, nil
}

func (fakeGateway) StartSession(context.Context, string) error       { return nil }
func (fakeGateway) EndSession(context.Context, string) error         { return nil }
func (fakeGateway) UpdateParticipants(context.Context, string) error { return nil }
func (fakeGateway) CreateCharacter(context.Context, string) error    { return nil }
func (fakeGateway) UpdateCharacter(context.Context, string) error    { return nil }
func (fakeGateway) ControlCharacter(context.Context, string) error   { return nil }
func (fakeGateway) CreateInvite(context.Context, string) error       { return nil }
func (fakeGateway) RevokeInvite(context.Context, string) error       { return nil }
func (fakeGateway) CanCampaignAction(context.Context, string, statev1.AuthorizationAction, statev1.AuthorizationResource, *statev1.AuthorizationTarget) (campaignAuthorizationDecision, error) {
	return campaignAuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"}, nil
}

type fakeCampaignClient struct {
	response   *statev1.ListCampaignsResponse
	err        error
	getResp    *statev1.GetCampaignResponse
	getErr     error
	createResp *statev1.CreateCampaignResponse
	createErr  error
}

type capturingCampaignClient struct {
	lastCreateReq *statev1.CreateCampaignRequest
}

type fakeSessionClient struct {
	response *statev1.ListSessionsResponse
	err      error
}

func (f fakeSessionClient) ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListSessionsResponse{}, nil
}

type fakeInviteClient struct {
	response *statev1.ListInvitesResponse
	err      error
}

func (f fakeInviteClient) ListInvites(context.Context, *statev1.ListInvitesRequest, ...grpc.CallOption) (*statev1.ListInvitesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListInvitesResponse{}, nil
}

func (c *capturingCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	return &statev1.ListCampaignsResponse{}, nil
}

func (c *capturingCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	return &statev1.GetCampaignResponse{}, nil
}

func (c *capturingCampaignClient) CreateCampaign(_ context.Context, req *statev1.CreateCampaignRequest, _ ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	c.lastCreateReq = req
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "camp-pt"}}, nil
}

func (f fakeCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

func (f fakeCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &statev1.GetCampaignResponse{Campaign: &statev1.Campaign{Id: "c1", Name: "Campaign"}}, nil
}

func (f fakeCampaignClient) CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "created"}}, nil
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return 0, io.EOF
}

type failingCampaignComponent struct {
	err error
}

func (c failingCampaignComponent) Render(context.Context, io.Writer) error {
	if c.err != nil {
		return c.err
	}
	return errors.New("render failed")
}
