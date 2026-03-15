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

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestModuleIDReturnsCampaigns(t *testing.T) {
	t.Parallel()

	if got := New(Config{}).ID(); got != "campaigns" {
		t.Fatalf("ID() = %q, want %q", got, "campaigns")
	}
}

func TestModuleHealthyReflectsGatewayState(t *testing.T) {
	t.Parallel()

	if New(Config{}).Healthy() {
		t.Fatalf("New().Healthy() = true, want false for degraded module")
	}
	if !New(configWithGateway(fakeGateway{}, modulehandler.NewTestBase(), nil)).Healthy() {
		t.Fatalf("New(Config{...}).Healthy() = false, want true")
	}
}

func TestMapCampaignCharacterCreationStepToProtoGatewayExport(t *testing.T) {
	t.Parallel()

	step := &campaignapp.CampaignCharacterCreationStepInput{
		Details: &campaignapp.CampaignCharacterCreationStepDetails{},
	}
	mapped, err := campaigngateway.MapCampaignCharacterCreationStepToProto(step)
	if err != nil {
		t.Fatalf("MapCampaignCharacterCreationStepToProto() error = %v", err)
	}
	if mapped == nil {
		t.Fatalf("MapCampaignCharacterCreationStepToProto() = nil, want non-nil")
	}
}

func TestMountServesCampaignsGet(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}, {ID: "c2", Name: "Second"}}}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if mount.Prefix != routepath.CampaignsPrefix {
		t.Fatalf("prefix = %q, want %q", mount.Prefix, routepath.CampaignsPrefix)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
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

func TestMountServesStarterPreviewUnderCampaignsPrefix(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		starterPreview: campaignapp.CampaignStarterPreview{
			EntryID:              "starter:lantern-in-the-dark",
			Title:                "The Lantern in the Dark",
			Description:          "A tight mystery for one session.",
			CampaignTheme:        "The lighthouse has gone dark.\nReturn home before the next fleet is lost.",
			Hook:                 "A lantern appears on the black tide.",
			PlaystyleLabel:       "Investigation",
			CharacterName:        "Seren Vale",
			CharacterSummary:     "A steadfast guardian chasing a vanished light.",
			System:               "Daggerheart",
			Difficulty:           "Beginner",
			Duration:             "1 session",
			GmMode:               "AI",
			Players:              "1",
			Tags:                 []string{"mystery"},
			AIAgentOptions:       []campaignapp.CampaignAIAgentOption{{ID: "agent-1", Label: "GM Agent", Enabled: true}},
			HasAvailableAIAgents: true,
		},
	}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignStarter("starter:lantern-in-the-dark"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "The Lantern in the Dark") {
		t.Fatalf("body missing starter preview title: %q", body)
	}
	if !strings.Contains(body, "Campaign Theme") || !strings.Contains(body, "Return home before the next fleet is lost.") {
		t.Fatalf("body missing campaign theme card: %q", body)
	}
	if strings.Contains(body, `<h2 class="text-3xl font-semibold">The Lantern in the Dark</h2>`) {
		t.Fatalf("body unexpectedly rendered duplicate in-content starter title: %q", body)
	}
	if !strings.Contains(body, `action="`+routepath.AppCampaignStarterLaunch("starter:lantern-in-the-dark")+`"`) {
		t.Fatalf("body missing starter launch action: %q", body)
	}
}

func TestMountStarterLaunchForksCampaignAndRedirects(t *testing.T) {
	t.Parallel()

	recorder := &starterLaunchCall{}
	gateway := fakeGateway{
		starterLaunchResult:   campaignapp.StarterLaunchResult{CampaignID: "camp-777"},
		starterLaunchRecorder: recorder,
	}
	m := New(configWithGateway(gateway, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	form := url.Values{"ai_agent_id": {"agent-1"}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignStarterLaunch("starter:lantern-in-the-dark"), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaign("camp-777") {
		t.Fatalf("location = %q, want %q", got, routepath.AppCampaign("camp-777"))
	}
	if recorder.starterKey != "starter:lantern-in-the-dark" {
		t.Fatalf("starter key = %q, want %q", recorder.starterKey, "starter:lantern-in-the-dark")
	}
	if recorder.input.AIAgentID != "agent-1" {
		t.Fatalf("launch input = %#v, want ai agent id agent-1", recorder.input)
	}
}

func TestMountRejectsMissingRequiredServices(t *testing.T) {
	t.Parallel()

	m := New(Config{})
	_, err := m.Mount()
	if err == nil {
		t.Fatalf("expected Mount() validation error")
	}
	if !strings.Contains(err.Error(), "missing required services") {
		t.Fatalf("Mount() error = %v, want missing-services validation error", err)
	}
}

func TestMountRejectsCampaignsNonGet(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, routepath.CampaignsPrefix+"123", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestMountMapsCampaignGatewayErrorToHTTPStatus(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{err: apperrors.E(apperrors.KindUnauthorized, "missing session")}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMountCampaignsGRPCNotFoundRendersAppErrorPage(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{err: status.Error(codes.NotFound, "campaign not found")}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
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

	m := New(configWithGateway(fakeGateway{err: errors.New("boom")}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
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

	m := New(configWithGateway(fakeGateway{err: status.Error(codes.NotFound, "campaign not found")}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
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

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
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

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
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

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
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

	deps := completeGRPCDeps(campaigngateway.GRPCGatewayDeps{
		CatalogRead: campaigngateway.CatalogReadDeps{
			Campaign: fakeCampaignClient{
				response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "remote-1", Name: "Remote Campaign"}}},
			},
		},
	})
	m := New(configWithGRPCDeps(deps, modulehandler.NewTestBase(), nil))
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
	if body := rr.Body.String(); !strings.Contains(body, "Remote Campaign") {
		t.Fatalf("body = %q, want remote campaign", body)
	}
}

func TestMountServesCampaignsGetWithEmptyList(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{}}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignsNew {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignsNew)
	}
	cookie := responseCookieByName(rr, flashnotice.CookieName)
	if cookie == nil {
		t.Fatalf("expected %q cookie", flashnotice.CookieName)
	}
	flashReq := httptest.NewRequest(http.MethodGet, routepath.AppCampaignsNew, nil)
	flashReq.AddCookie(cookie)
	flashRR := httptest.NewRecorder()
	notice, ok := flashnotice.ReadAndClear(flashRR, flashReq)
	if !ok {
		t.Fatalf("ReadAndClear() ok = false, want true")
	}
	if notice.Key != "game.campaigns.empty" {
		t.Fatalf("notice.Key = %q", notice.Key)
	}
	if notice.Kind != flashnotice.KindInfo {
		t.Fatalf("notice.Kind = %q, want %q", notice.Kind, flashnotice.KindInfo)
	}
}

func TestMountCampaignsGetWithEmptyListUsesHXRedirect(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{}}, modulehandler.NewTestBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignsNew {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignsNew)
	}
	if cookie := responseCookieByName(rr, flashnotice.CookieName); cookie == nil {
		t.Fatalf("expected %q cookie", flashnotice.CookieName)
	}
}

func TestCampaignBreadcrumbsFallbackToCampaignID(t *testing.T) {
	t.Parallel()

	breadcrumbs := campaignBreadcrumbs("camp-1", "   ", nil)
	if len(breadcrumbs) != 2 {
		t.Fatalf("len(breadcrumbs) = %d, want 2", len(breadcrumbs))
	}
	if breadcrumbs[1].Label != "camp-1" {
		t.Fatalf("campaign breadcrumb label = %q, want %q", breadcrumbs[1].Label, "camp-1")
	}
}

func TestWriteCampaignHTMLHandlesRenderFailure(t *testing.T) {
	t.Parallel()

	h := newHandlersFromConfig(serviceConfigWithGateway(fakeGateway{}), modulehandler.NewTestBase(), nil)
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
	rr := httptest.NewRecorder()

	h.WritePage(rr, req, "Campaigns", http.StatusOK, campaignsListHeader(nil), webtemplates.AppMainLayoutOptions{}, failingCampaignComponent{err: errors.New("render failed")})
	// Buffered rendering catches the error before headers are sent, producing
	// a clean error page instead of a partially-written response.
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: template/render failures must not leak internal error details to end users.
	if strings.Contains(body, "render failed") {
		t.Fatalf("body leaked internal render error: %q", body)
	}
}

func TestGRPCGatewayCampaignNameReturnsEmptyWhenCampaignMissing(t *testing.T) {
	t.Parallel()

	g := campaigngateway.NewWorkspaceReadGateway(campaigngateway.WorkspaceReadDeps{
		Campaign: fakeCampaignClient{getResp: &statev1.GetCampaignResponse{Campaign: nil}},
	}, "")
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

	g := campaigngateway.NewCatalogMutationGateway(campaigngateway.CatalogMutationDeps{
		Campaign: fakeCampaignClient{createResp: &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{}}},
	})
	_, err := g.CreateCampaign(context.Background(), campaignapp.CreateCampaignInput{Name: "New", System: campaignapp.GameSystemDaggerheart, GMMode: campaignapp.GmModeHuman})
	if err == nil {
		t.Fatalf("expected empty campaign id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestGRPCGatewayMutationConstructorsReturnNilWhenDependenciesMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		gotNil bool
	}{
		{name: "start session", gotNil: campaigngateway.NewSessionMutationGateway(campaigngateway.SessionMutationDeps{}) == nil},
		{name: "character control", gotNil: campaigngateway.NewCharacterControlMutationGateway(campaigngateway.CharacterControlMutationDeps{}) == nil},
		{name: "create character", gotNil: campaigngateway.NewCharacterMutationGateway(campaigngateway.CharacterMutationDeps{}) == nil},
		{name: "create participant", gotNil: campaigngateway.NewParticipantMutationGateway(campaigngateway.ParticipantMutationDeps{}) == nil},
		{name: "create invite", gotNil: campaigngateway.NewInviteMutationGateway(campaigngateway.InviteMutationDeps{}) == nil},
		{name: "create campaign", gotNil: campaigngateway.NewCatalogMutationGateway(campaigngateway.CatalogMutationDeps{}) == nil},
		{name: "update campaign", gotNil: campaigngateway.NewConfigurationMutationGateway(campaigngateway.ConfigurationMutationDeps{}) == nil},
		{name: "automation mutation", gotNil: campaigngateway.NewAutomationMutationGateway(campaigngateway.AutomationMutationDeps{}) == nil},
		{name: "creation mutation", gotNil: campaigngateway.NewCharacterCreationMutationGateway(campaigngateway.CharacterCreationMutationDeps{}) == nil},
		{name: "authorization", gotNil: campaigngateway.NewAuthorizationGateway(campaigngateway.AuthorizationDeps{}) == nil},
		{name: "batch authorization", gotNil: campaigngateway.NewBatchAuthorizationGateway(campaigngateway.AuthorizationDeps{}) == nil},
		{name: "session read", gotNil: campaigngateway.NewSessionReadGateway(campaigngateway.SessionReadDeps{}) == nil},
		{name: "invite read", gotNil: campaigngateway.NewInviteReadGateway(campaigngateway.InviteReadDeps{}) == nil},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !tc.gotNil {
				t.Fatalf("expected nil constructor result for missing dependencies")
			}
		})
	}
}

func TestGRPCGatewayCampaignSessionsMapsSessionRows(t *testing.T) {
	t.Parallel()

	g := campaigngateway.NewSessionReadGateway(campaigngateway.SessionReadDeps{
		Campaign: fakeCampaignClient{},
		Session: fakeSessionClient{response: &statev1.ListSessionsResponse{Sessions: []*statev1.Session{{
			Id:         "s1",
			CampaignId: "c1",
			Name:       "First Light",
			Status:     statev1.SessionStatus_SESSION_ACTIVE,
			UpdatedAt:  timestamppb.New(time.Date(2026, 2, 24, 18, 0, 0, 0, time.UTC)),
		}}}},
	})

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

	g := campaigngateway.NewInviteReadGateway(
		campaigngateway.InviteReadDeps{
			Invite: fakeInviteClient{response: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{{
				Id:              "inv-1",
				CampaignId:      "c1",
				ParticipantId:   "p1",
				RecipientUserId: "user-2",
				Status:          statev1.InviteStatus_PENDING,
			}}}},
			Participant: fakeParticipantClient{},
			Social:      fakeSocialClient{},
			Auth:        fakeAuthClient{},
		},
	)

	invites, err := g.CampaignInvites(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CampaignInvites() error = %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("len(invites) = %d, want 1", len(invites))
	}
	if invites[0].ID != "inv-1" || invites[0].ParticipantID != "p1" || invites[0].ParticipantName != "Pending Seat" || invites[0].RecipientUserID != "user-2" || invites[0].RecipientUsername != "user" || invites[0].Status != "Pending" {
		t.Fatalf("invites[0] = %+v, want mapped invite fields", invites[0])
	}
}

func TestGRPCGatewayCampaignSessionsConstructorRequiresDependencies(t *testing.T) {
	t.Parallel()

	if got := campaigngateway.NewSessionReadGateway(campaigngateway.SessionReadDeps{}); got != nil {
		t.Fatalf("expected nil session read gateway for missing deps")
	}
}

func TestGRPCGatewayCampaignInvitesConstructorRequiresDependencies(t *testing.T) {
	t.Parallel()

	if got := campaigngateway.NewInviteReadGateway(campaigngateway.InviteReadDeps{}); got != nil {
		t.Fatalf("expected nil invite read gateway for missing deps")
	}
}

// testCreationWorkflow implements campaignworkflow.CharacterCreation for tests.
// It passes through catalog data without Daggerheart-specific filtering,
// so test data should only include entries expected in output.
type testCreationWorkflow struct{}

func (testCreationWorkflow) BuildView(
	progress campaignworkflow.Progress,
	catalog campaignworkflow.Catalog,
	profile campaignworkflow.Profile,
) campaignrender.CampaignCharacterCreationView {
	view := campaignrender.CampaignCharacterCreationView{
		Ready:             progress.Ready,
		NextStep:          progress.NextStep,
		UnmetReasons:      append([]string(nil), progress.UnmetReasons...),
		ClassID:           profile.ClassID,
		SubclassID:        profile.SubclassID,
		AncestryID:        profile.AncestryID,
		CommunityID:       profile.CommunityID,
		Agility:           profile.Agility,
		Strength:          profile.Strength,
		Finesse:           profile.Finesse,
		Instinct:          profile.Instinct,
		Presence:          profile.Presence,
		Knowledge:         profile.Knowledge,
		PrimaryWeaponID:   profile.PrimaryWeaponID,
		SecondaryWeaponID: profile.SecondaryWeaponID,
		ArmorID:           profile.ArmorID,
		PotionItemID:      profile.PotionItemID,
		Background:        profile.Background,
		DomainCardIDs:     append([]string(nil), profile.DomainCardIDs...),
		Connections:       profile.Connections,
		Steps:             make([]campaignrender.CampaignCharacterCreationStepView, 0, len(progress.Steps)),
	}
	for _, step := range progress.Steps {
		view.Steps = append(view.Steps, campaignrender.CampaignCharacterCreationStepView{Step: step.Step, Key: step.Key, Complete: step.Complete})
	}
	for _, c := range catalog.Classes {
		view.Classes = append(view.Classes, campaignrender.CampaignCreationClassView{ID: c.ID, Name: c.Name})
	}
	for _, s := range catalog.Subclasses {
		view.Subclasses = append(view.Subclasses, campaignrender.CampaignCreationSubclassView{ID: s.ID, Name: s.Name, ClassID: s.ClassID})
	}
	for _, h := range catalog.Heritages {
		entry := campaignrender.CampaignCreationHeritageView{ID: h.ID, Name: h.Name}
		switch h.Kind {
		case "ancestry":
			view.Ancestries = append(view.Ancestries, entry)
		case "community":
			view.Communities = append(view.Communities, entry)
		}
	}
	for _, w := range catalog.Weapons {
		entry := campaignrender.CampaignCreationWeaponView{ID: w.ID, Name: w.Name}
		switch w.Category {
		case "primary":
			view.PrimaryWeapons = append(view.PrimaryWeapons, entry)
		case "secondary":
			view.SecondaryWeapons = append(view.SecondaryWeapons, entry)
		}
	}
	for _, a := range catalog.Armor {
		view.Armor = append(view.Armor, campaignrender.CampaignCreationArmorView{ID: a.ID, Name: a.Name})
	}
	for _, i := range catalog.Items {
		view.PotionItems = append(view.PotionItems, campaignrender.CampaignCreationItemView{ID: i.ID, Name: i.Name})
	}
	for _, d := range catalog.DomainCards {
		view.DomainCards = append(view.DomainCards, campaignrender.CampaignCreationDomainCardView{ID: d.ID, Name: d.Name, DomainID: d.DomainID, Level: d.Level})
	}
	return view
}

func (testCreationWorkflow) ParseStepInput(form url.Values, nextStep int32) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	switch nextStep {
	case 1:
		classID := strings.TrimSpace(form.Get("class_id"))
		subclassID := strings.TrimSpace(form.Get("subclass_id"))
		if classID == "" || subclassID == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_class_and_subclass_are_required", "class and subclass are required")
		}
		return &campaignapp.CampaignCharacterCreationStepInput{
			ClassSubclass: &campaignapp.CampaignCharacterCreationStepClassSubclass{ClassID: classID, SubclassID: subclassID},
		}, nil
	default:
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}
}

// defaultTestWorkflows returns a workflow map suitable for tests that need
// character creation enabled for Daggerheart campaigns.
func defaultTestWorkflows() map[campaignapp.GameSystem]campaignworkflow.CharacterCreation {
	return map[campaignapp.GameSystem]campaignworkflow.CharacterCreation{campaignapp.GameSystemDaggerheart: testCreationWorkflow{}}
}

type fakeGateway struct {
	items                             []campaignapp.CampaignSummary
	starterPreview                    campaignapp.CampaignStarterPreview
	starterPreviewErr                 error
	starterLaunchResult               campaignapp.StarterLaunchResult
	starterLaunchErr                  error
	starterLaunchRecorder             *starterLaunchCall
	workspaceSystem                   string
	workspaceGMMode                   string
	workspaceStatus                   string
	workspaceLocale                   string
	workspaceIntent                   string
	workspaceAccessPolicy             string
	workspaceAIAgentID                string
	gameSurface                       campaignapp.CampaignGameSurface
	gameSurfaceErr                    error
	campaignAIAgents                  []campaignapp.CampaignAIAgentOption
	campaignAIAgentsErr               error
	participants                      []campaignapp.CampaignParticipant
	participantsErr                   error
	participant                       campaignapp.CampaignParticipant
	participantErr                    error
	characters                        []campaignapp.CampaignCharacter
	charactersErr                     error
	sessions                          []campaignapp.CampaignSession
	sessionsErr                       error
	sessionReadiness                  campaignapp.CampaignSessionReadiness
	sessionReadinessErr               error
	invites                           []campaignapp.CampaignInvite
	invitesErr                        error
	inviteSearchResults               []campaignapp.InviteUserSearchResult
	inviteSearchErr                   error
	lastInviteSearchInput             *campaignapp.SearchInviteUsersInput
	characterCreationProgress         campaignapp.CampaignCharacterCreationProgress
	characterCreationProgressErr      error
	characterCreationCatalog          campaignapp.CampaignCharacterCreationCatalog
	characterCreationCatalogErr       error
	characterCreationProfile          campaignapp.CampaignCharacterCreationProfile
	characterCreationProfileErr       error
	authorizationDecision             campaignapp.AuthorizationDecision
	authorizationErr                  error
	batchAuthorizationDecisions       []campaignapp.AuthorizationDecision
	batchAuthorizationErr             error
	applyCharacterCreationStepErr     error
	resetCharacterCreationWorkflowErr error
	createCharacterErr                error
	createdCharacterID                string
	deleteCharacterErr                error
	setCharacterControllerErr         error
	claimCharacterControlErr          error
	releaseCharacterControlErr        error
	updateCampaignErr                 error
	updateCampaignAIBindingErr        error
	createParticipantErr              error
	createInviteErr                   error
	revokeInviteErr                   error
	createdParticipantID              string
	lastCreateParticipantInput        campaignapp.CreateParticipantInput
	lastCreateInviteInput             campaignapp.CreateInviteInput
	lastRevokeInviteInput             campaignapp.RevokeInviteInput
	updateParticipantErr              error
	err                               error
	createErr                         error
	createdCampaignID                 string
}

type mutationContextGateway struct {
	fakeGateway
	startSessionUserID string
}

func (g *mutationContextGateway) StartSession(ctx context.Context, _ string, _ campaignapp.StartSessionInput) error {
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

func (f fakeGateway) ListCampaigns(context.Context) ([]campaignapp.CampaignSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func (f fakeGateway) StarterPreview(context.Context, string) (campaignapp.CampaignStarterPreview, error) {
	if f.starterPreviewErr != nil {
		return campaignapp.CampaignStarterPreview{}, f.starterPreviewErr
	}
	return f.starterPreview, nil
}

type starterLaunchCall struct {
	starterKey string
	input      campaignapp.LaunchStarterInput
}

func (f fakeGateway) LaunchStarter(_ context.Context, starterKey string, input campaignapp.LaunchStarterInput) (campaignapp.StarterLaunchResult, error) {
	if f.starterLaunchRecorder != nil {
		f.starterLaunchRecorder.starterKey = starterKey
		f.starterLaunchRecorder.input = input
	}
	if f.starterLaunchErr != nil {
		return campaignapp.StarterLaunchResult{}, f.starterLaunchErr
	}
	return f.starterLaunchResult, nil
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

func (f fakeGateway) CampaignWorkspace(_ context.Context, campaignID string) (campaignapp.CampaignWorkspace, error) {
	campaignID = strings.TrimSpace(campaignID)
	for _, item := range f.items {
		if strings.TrimSpace(item.ID) != campaignID {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = campaignID
		}
		system := strings.TrimSpace(f.workspaceSystem)
		if system == "" {
			system = "Daggerheart"
		}
		gmMode := strings.TrimSpace(f.workspaceGMMode)
		if gmMode == "" {
			gmMode = "Human"
		}
		status := strings.TrimSpace(f.workspaceStatus)
		if status == "" {
			status = "Active"
		}
		locale := strings.TrimSpace(f.workspaceLocale)
		if locale == "" {
			locale = "English (US)"
		}
		intent := strings.TrimSpace(f.workspaceIntent)
		if intent == "" {
			intent = "Standard"
		}
		accessPolicy := strings.TrimSpace(f.workspaceAccessPolicy)
		if accessPolicy == "" {
			accessPolicy = "Public"
		}
		return campaignapp.CampaignWorkspace{
			ID:               campaignID,
			Name:             name,
			Theme:            strings.TrimSpace(item.Theme),
			System:           system,
			GMMode:           gmMode,
			AIAgentID:        strings.TrimSpace(f.workspaceAIAgentID),
			Status:           status,
			Locale:           locale,
			Intent:           intent,
			AccessPolicy:     accessPolicy,
			ParticipantCount: strings.TrimSpace(item.ParticipantCount),
			CharacterCount:   strings.TrimSpace(item.CharacterCount),
			CoverImageURL:    strings.TrimSpace(item.CoverImageURL),
		}, nil
	}
	return campaignapp.CampaignWorkspace{}, apperrors.E(apperrors.KindNotFound, "campaign not found")
}

func (f fakeGateway) CampaignGameSurface(_ context.Context, campaignID string) (campaignapp.CampaignGameSurface, error) {
	if f.gameSurfaceErr != nil {
		return campaignapp.CampaignGameSurface{}, f.gameSurfaceErr
	}
	surface := f.gameSurface
	if strings.TrimSpace(surface.Participant.ID) == "" {
		surface.Participant.ID = "p1"
	}
	if strings.TrimSpace(surface.Participant.Name) == "" {
		surface.Participant.Name = "Owner"
	}
	if strings.TrimSpace(surface.Participant.Role) == "" {
		surface.Participant.Role = "Player"
	}
	if strings.TrimSpace(surface.SessionID) == "" {
		surface.SessionID = "sess-1"
	}
	if strings.TrimSpace(surface.SessionName) == "" {
		surface.SessionName = "Session One"
	}
	if surface.ActiveScene == nil {
		surface.ActiveScene = &campaignapp.CampaignGameScene{
			ID:        "scene-1",
			SessionID: surface.SessionID,
			Name:      "Session One Scene",
			Characters: []campaignapp.CampaignGameCharacter{
				{ID: "char-1", Name: "Owner", OwnerParticipantID: surface.Participant.ID},
			},
		}
	}
	if surface.PlayerPhase == nil {
		surface.PlayerPhase = &campaignapp.CampaignGamePlayerPhase{
			PhaseID:              "phase-1",
			Status:               "players",
			ActingCharacterIDs:   []string{"char-1"},
			ActingParticipantIDs: []string{surface.Participant.ID},
			Slots:                []campaignapp.CampaignGamePlayerSlot{},
		}
	}
	if len(surface.OOC.Posts) == 0 {
		surface.OOC.Posts = []campaignapp.CampaignGameOOCPost{}
	}
	return surface, nil
}

func (f fakeGateway) CampaignAIAgents(context.Context) ([]campaignapp.CampaignAIAgentOption, error) {
	if f.campaignAIAgentsErr != nil {
		return nil, f.campaignAIAgentsErr
	}
	return f.campaignAIAgents, nil
}

func (f fakeGateway) CampaignParticipants(context.Context, string) ([]campaignapp.CampaignParticipant, error) {
	if f.participantsErr != nil {
		return nil, f.participantsErr
	}
	return f.participants, nil
}

func (f fakeGateway) CampaignParticipant(context.Context, string, string) (campaignapp.CampaignParticipant, error) {
	if f.participantErr != nil {
		return campaignapp.CampaignParticipant{}, f.participantErr
	}
	if strings.TrimSpace(f.participant.ID) != "" {
		return f.participant, nil
	}
	if len(f.participants) > 0 {
		return f.participants[0], nil
	}
	return campaignapp.CampaignParticipant{}, nil
}

func (f fakeGateway) CampaignCharacters(context.Context, string, campaignapp.CharacterReadContext) ([]campaignapp.CampaignCharacter, error) {
	if f.charactersErr != nil {
		return nil, f.charactersErr
	}
	return f.characters, nil
}

func (f fakeGateway) CampaignCharacter(_ context.Context, _ string, characterID string, _ campaignapp.CharacterReadContext) (campaignapp.CampaignCharacter, error) {
	if f.charactersErr != nil {
		return campaignapp.CampaignCharacter{}, f.charactersErr
	}
	for _, character := range f.characters {
		if strings.TrimSpace(character.ID) == strings.TrimSpace(characterID) {
			return character, nil
		}
	}
	return campaignapp.CampaignCharacter{ID: strings.TrimSpace(characterID)}, nil
}

func (f fakeGateway) CampaignSessions(context.Context, string) ([]campaignapp.CampaignSession, error) {
	if f.sessionsErr != nil {
		return nil, f.sessionsErr
	}
	return f.sessions, nil
}

func (f fakeGateway) CampaignSessionReadiness(context.Context, string, language.Tag) (campaignapp.CampaignSessionReadiness, error) {
	if f.sessionReadinessErr != nil {
		return campaignapp.CampaignSessionReadiness{}, f.sessionReadinessErr
	}
	if !f.sessionReadiness.Ready && len(f.sessionReadiness.Blockers) == 0 {
		return campaignapp.CampaignSessionReadiness{Ready: true, Blockers: []campaignapp.CampaignSessionReadinessBlocker{}}, nil
	}
	return f.sessionReadiness, nil
}

func (f fakeGateway) CampaignInvites(context.Context, string) ([]campaignapp.CampaignInvite, error) {
	if f.invitesErr != nil {
		return nil, f.invitesErr
	}
	return f.invites, nil
}

func (f fakeGateway) SearchInviteUsers(_ context.Context, input campaignapp.SearchInviteUsersInput) ([]campaignapp.InviteUserSearchResult, error) {
	if f.lastInviteSearchInput != nil {
		*f.lastInviteSearchInput = input
	}
	if f.inviteSearchErr != nil {
		return nil, f.inviteSearchErr
	}
	return f.inviteSearchResults, nil
}

func (f fakeGateway) CharacterCreationProgress(context.Context, string, string) (campaignapp.CampaignCharacterCreationProgress, error) {
	if f.characterCreationProgressErr != nil {
		return campaignapp.CampaignCharacterCreationProgress{}, f.characterCreationProgressErr
	}
	return f.characterCreationProgress, nil
}

func (f fakeGateway) CharacterCreationCatalog(context.Context, language.Tag) (campaignapp.CampaignCharacterCreationCatalog, error) {
	if f.characterCreationCatalogErr != nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, f.characterCreationCatalogErr
	}
	return f.characterCreationCatalog, nil
}

func (f fakeGateway) CharacterCreationProfile(context.Context, string, string) (campaignapp.CampaignCharacterCreationProfile, error) {
	if f.characterCreationProfileErr != nil {
		return campaignapp.CampaignCharacterCreationProfile{}, f.characterCreationProfileErr
	}
	return f.characterCreationProfile, nil
}

func (f fakeGateway) CreateCampaign(context.Context, campaignapp.CreateCampaignInput) (campaignapp.CreateCampaignResult, error) {
	if f.createErr != nil {
		return campaignapp.CreateCampaignResult{}, f.createErr
	}
	createdID := strings.TrimSpace(f.createdCampaignID)
	if createdID == "" {
		createdID = "created"
	}
	return campaignapp.CreateCampaignResult{CampaignID: createdID}, nil
}

func (f fakeGateway) UpdateCampaign(context.Context, string, campaignapp.UpdateCampaignInput) error {
	return f.updateCampaignErr
}

func (f fakeGateway) UpdateCampaignAIBinding(context.Context, string, campaignapp.UpdateCampaignAIBindingInput) error {
	return f.updateCampaignAIBindingErr
}

func (fakeGateway) StartSession(context.Context, string, campaignapp.StartSessionInput) error {
	return nil
}
func (fakeGateway) EndSession(context.Context, string, campaignapp.EndSessionInput) error { return nil }
func (f fakeGateway) CreateCharacter(context.Context, string, campaignapp.CreateCharacterInput) (campaignapp.CreateCharacterResult, error) {
	if f.createCharacterErr != nil {
		return campaignapp.CreateCharacterResult{}, f.createCharacterErr
	}
	createdCharacterID := strings.TrimSpace(f.createdCharacterID)
	if createdCharacterID == "" {
		createdCharacterID = "char-created"
	}
	return campaignapp.CreateCharacterResult{CharacterID: createdCharacterID}, nil
}
func (f fakeGateway) CreateParticipant(_ context.Context, _ string, input campaignapp.CreateParticipantInput) (campaignapp.CreateParticipantResult, error) {
	if f.createParticipantErr != nil {
		return campaignapp.CreateParticipantResult{}, f.createParticipantErr
	}
	f.lastCreateParticipantInput = input
	createdParticipantID := strings.TrimSpace(f.createdParticipantID)
	if createdParticipantID == "" {
		createdParticipantID = "participant-created"
	}
	return campaignapp.CreateParticipantResult{ParticipantID: createdParticipantID}, nil
}
func (fakeGateway) UpdateCharacter(context.Context, string, string, campaignapp.UpdateCharacterInput) error {
	return nil
}
func (f fakeGateway) DeleteCharacter(context.Context, string, string) error {
	return f.deleteCharacterErr
}
func (f fakeGateway) SetCharacterController(context.Context, string, string, string) error {
	return f.setCharacterControllerErr
}
func (f fakeGateway) ClaimCharacterControl(context.Context, string, string) error {
	return f.claimCharacterControlErr
}
func (f fakeGateway) ReleaseCharacterControl(context.Context, string, string) error {
	return f.releaseCharacterControlErr
}
func (f fakeGateway) UpdateParticipant(context.Context, string, campaignapp.UpdateParticipantInput) error {
	return f.updateParticipantErr
}
func (f fakeGateway) CreateInvite(_ context.Context, _ string, input campaignapp.CreateInviteInput) error {
	if f.createInviteErr != nil {
		return f.createInviteErr
	}
	f.lastCreateInviteInput = input
	return nil
}
func (f fakeGateway) RevokeInvite(_ context.Context, _ string, input campaignapp.RevokeInviteInput) error {
	if f.revokeInviteErr != nil {
		return f.revokeInviteErr
	}
	f.lastRevokeInviteInput = input
	return nil
}
func (f fakeGateway) ApplyCharacterCreationStep(context.Context, string, string, *campaignapp.CampaignCharacterCreationStepInput) error {
	return f.applyCharacterCreationStepErr
}
func (f fakeGateway) ResetCharacterCreationWorkflow(context.Context, string, string) error {
	return f.resetCharacterCreationWorkflowErr
}
func (f fakeGateway) CanCampaignAction(context.Context, string, campaignapp.AuthorizationAction, campaignapp.AuthorizationResource, *campaignapp.AuthorizationTarget) (campaignapp.AuthorizationDecision, error) {
	if f.authorizationErr != nil {
		return campaignapp.AuthorizationDecision{}, f.authorizationErr
	}
	if f.authorizationDecision.Evaluated || f.authorizationDecision.Allowed || strings.TrimSpace(f.authorizationDecision.ReasonCode) != "" {
		return f.authorizationDecision, nil
	}
	return campaignapp.AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"}, nil
}

func (f fakeGateway) BatchCanCampaignAction(context.Context, string, []campaignapp.AuthorizationCheck) ([]campaignapp.AuthorizationDecision, error) {
	if f.batchAuthorizationErr != nil {
		return nil, f.batchAuthorizationErr
	}
	return append([]campaignapp.AuthorizationDecision(nil), f.batchAuthorizationDecisions...), nil
}

type fakeCampaignClient struct {
	response          *statev1.ListCampaignsResponse
	err               error
	getResp           *statev1.GetCampaignResponse
	getErr            error
	readinessResp     *statev1.GetCampaignSessionReadinessResponse
	readinessErr      error
	createResp        *statev1.CreateCampaignResponse
	createErr         error
	updateResp        *statev1.UpdateCampaignResponse
	updateErr         error
	setAIBindingErr   error
	clearAIBindingErr error
}

type capturingCampaignClient struct {
	lastCreateReq         *statev1.CreateCampaignRequest
	lastUpdateReq         *statev1.UpdateCampaignRequest
	lastSetAIBindingReq   *statev1.SetCampaignAIBindingRequest
	lastClearAIBindingReq *statev1.ClearCampaignAIBindingRequest
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

func (f fakeSessionClient) StartSession(_ context.Context, req *statev1.StartSessionRequest, _ ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.StartSessionResponse{Session: &statev1.Session{
		Id:         "sess-created",
		CampaignId: strings.TrimSpace(req.GetCampaignId()),
		Name:       strings.TrimSpace(req.GetName()),
		Status:     statev1.SessionStatus_SESSION_ACTIVE,
	}}, nil
}

func (f fakeSessionClient) ListActiveSessionsForUser(context.Context, *statev1.ListActiveSessionsForUserRequest, ...grpc.CallOption) (*statev1.ListActiveSessionsForUserResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ListActiveSessionsForUserResponse{}, nil
}

func (f fakeSessionClient) EndSession(_ context.Context, req *statev1.EndSessionRequest, _ ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.EndSessionResponse{Session: &statev1.Session{
		Id:         strings.TrimSpace(req.GetSessionId()),
		CampaignId: strings.TrimSpace(req.GetCampaignId()),
		Status:     statev1.SessionStatus_SESSION_ENDED,
	}}, nil
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

func (f fakeInviteClient) GetPublicInvite(context.Context, *statev1.GetPublicInviteRequest, ...grpc.CallOption) (*statev1.GetPublicInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.GetPublicInviteResponse{Invite: &statev1.Invite{}}, nil
}

func (f fakeInviteClient) CreateInvite(_ context.Context, req *statev1.CreateInviteRequest, _ ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.CreateInviteResponse{Invite: &statev1.Invite{
		Id:              "invite-created",
		CampaignId:      strings.TrimSpace(req.GetCampaignId()),
		ParticipantId:   strings.TrimSpace(req.GetParticipantId()),
		RecipientUserId: strings.TrimSpace(req.GetRecipientUserId()),
		Status:          statev1.InviteStatus_PENDING,
	}}, nil
}

func (f fakeInviteClient) ClaimInvite(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ClaimInviteResponse{}, nil
}

func (f fakeInviteClient) DeclineInvite(context.Context, *statev1.DeclineInviteRequest, ...grpc.CallOption) (*statev1.DeclineInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.DeclineInviteResponse{}, nil
}

func (f fakeInviteClient) RevokeInvite(_ context.Context, req *statev1.RevokeInviteRequest, _ ...grpc.CallOption) (*statev1.RevokeInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.RevokeInviteResponse{Invite: &statev1.Invite{
		Id:     strings.TrimSpace(req.GetInviteId()),
		Status: statev1.InviteStatus_REVOKED,
	}}, nil
}

func (c *capturingCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	return &statev1.ListCampaignsResponse{}, nil
}

func (c *capturingCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	return &statev1.GetCampaignResponse{}, nil
}

func (c *capturingCampaignClient) GetCampaignSessionReadiness(context.Context, *statev1.GetCampaignSessionReadinessRequest, ...grpc.CallOption) (*statev1.GetCampaignSessionReadinessResponse, error) {
	return &statev1.GetCampaignSessionReadinessResponse{
		Readiness: &statev1.CampaignSessionReadiness{Ready: true},
	}, nil
}

func (c *capturingCampaignClient) CreateCampaign(_ context.Context, req *statev1.CreateCampaignRequest, _ ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	c.lastCreateReq = req
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "camp-pt"}}, nil
}

func (c *capturingCampaignClient) UpdateCampaign(_ context.Context, req *statev1.UpdateCampaignRequest, _ ...grpc.CallOption) (*statev1.UpdateCampaignResponse, error) {
	c.lastUpdateReq = req
	return &statev1.UpdateCampaignResponse{Campaign: &statev1.Campaign{Id: strings.TrimSpace(req.GetCampaignId())}}, nil
}

func (c *capturingCampaignClient) SetCampaignAIBinding(_ context.Context, req *statev1.SetCampaignAIBindingRequest, _ ...grpc.CallOption) (*statev1.SetCampaignAIBindingResponse, error) {
	c.lastSetAIBindingReq = req
	return &statev1.SetCampaignAIBindingResponse{}, nil
}

func (c *capturingCampaignClient) ClearCampaignAIBinding(_ context.Context, req *statev1.ClearCampaignAIBindingRequest, _ ...grpc.CallOption) (*statev1.ClearCampaignAIBindingResponse, error) {
	c.lastClearAIBindingReq = req
	return &statev1.ClearCampaignAIBindingResponse{}, nil
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

func (f fakeCampaignClient) GetCampaignSessionReadiness(context.Context, *statev1.GetCampaignSessionReadinessRequest, ...grpc.CallOption) (*statev1.GetCampaignSessionReadinessResponse, error) {
	if f.readinessErr != nil {
		return nil, f.readinessErr
	}
	if f.readinessResp != nil {
		return f.readinessResp, nil
	}
	return &statev1.GetCampaignSessionReadinessResponse{
		Readiness: &statev1.CampaignSessionReadiness{Ready: true},
	}, nil
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

func (f fakeCampaignClient) UpdateCampaign(context.Context, *statev1.UpdateCampaignRequest, ...grpc.CallOption) (*statev1.UpdateCampaignResponse, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.updateResp != nil {
		return f.updateResp, nil
	}
	return &statev1.UpdateCampaignResponse{Campaign: &statev1.Campaign{Id: "updated"}}, nil
}

func (f fakeCampaignClient) SetCampaignAIBinding(context.Context, *statev1.SetCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.SetCampaignAIBindingResponse, error) {
	if f.setAIBindingErr != nil {
		return nil, f.setAIBindingErr
	}
	return &statev1.SetCampaignAIBindingResponse{}, nil
}

func (f fakeCampaignClient) ClearCampaignAIBinding(context.Context, *statev1.ClearCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.ClearCampaignAIBindingResponse, error) {
	if f.clearAIBindingErr != nil {
		return nil, f.clearAIBindingErr
	}
	return &statev1.ClearCampaignAIBindingResponse{}, nil
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

func responseCookieByName(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	if rr == nil {
		return nil
	}
	for _, cookie := range rr.Result().Cookies() {
		if cookie != nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}

// completeGRPCDeps fills in stub clients for any nil required fields so the
// explicit campaigns gateway constructors resolve to concrete adapters instead
// of the fail-closed unavailable gateway. Tests only need to set the clients
// they exercise.
func completeGRPCDeps(deps campaigngateway.GRPCGatewayDeps) campaigngateway.GRPCGatewayDeps {
	if deps.Starter.Discovery == nil {
		deps.Starter.Discovery = stubDiscoveryClient{}
	}
	if deps.Starter.Agent == nil {
		deps.Starter.Agent = stubAgentClient{}
	}
	if deps.Starter.Campaign == nil {
		deps.Starter.Campaign = fakeCampaignClient{}
	}
	if deps.Starter.Fork == nil {
		deps.Starter.Fork = stubForkClient{}
	}
	if deps.CatalogRead.Campaign == nil {
		deps.CatalogRead.Campaign = fakeCampaignClient{}
	}
	if deps.CatalogMutation.Campaign == nil {
		deps.CatalogMutation.Campaign = fakeCampaignClient{}
	}
	if deps.WorkspaceRead.Campaign == nil {
		deps.WorkspaceRead.Campaign = deps.CatalogRead.Campaign
	}
	if deps.ConfigMutate.Campaign == nil {
		deps.ConfigMutate.Campaign = deps.CatalogMutation.Campaign
	}
	if deps.AutomationMutate.Campaign == nil {
		deps.AutomationMutate.Campaign = deps.CatalogMutation.Campaign
	}
	if deps.GameRead.Interaction == nil {
		deps.GameRead.Interaction = stubInteractionClient{}
	}
	if deps.AutomationRead.Agent == nil {
		deps.AutomationRead.Agent = stubAgentClient{}
	}
	if deps.ParticipantRead.Participant == nil {
		deps.ParticipantRead.Participant = stubParticipantReadClient{}
	}
	if deps.ParticipantMutate.Participant == nil {
		deps.ParticipantMutate.Participant = stubParticipantMutationClient{}
	}
	if deps.CharacterRead.Character == nil {
		deps.CharacterRead.Character = stubCharacterReadClient{}
	}
	if deps.CharacterRead.Participant == nil {
		deps.CharacterRead.Participant = deps.ParticipantRead.Participant
	}
	if deps.CreationRead.Character == nil {
		deps.CreationRead.Character = deps.CharacterRead.Character
	}
	if deps.CharacterMutate.Character == nil {
		deps.CharacterMutate.Character = stubCharacterMutationClient{}
	}
	if deps.CharacterControl.Character == nil {
		deps.CharacterControl.Character = deps.CharacterMutate.Character
	}
	if deps.CreationMutation.Character == nil {
		deps.CreationMutation.Character = deps.CharacterMutate.Character
	}
	if deps.CharacterRead.DaggerheartContent == nil {
		deps.CharacterRead.DaggerheartContent = stubDaggerheartContentClient{}
	}
	if deps.CreationRead.DaggerheartContent == nil {
		deps.CreationRead.DaggerheartContent = deps.CharacterRead.DaggerheartContent
	}
	if deps.CreationRead.DaggerheartAsset == nil {
		deps.CreationRead.DaggerheartAsset = stubDaggerheartAssetClient{}
	}
	if deps.SessionRead.Campaign == nil {
		deps.SessionRead.Campaign = deps.CatalogRead.Campaign
	}
	if deps.SessionRead.Session == nil {
		deps.SessionRead.Session = fakeSessionClient{}
	}
	if deps.SessionMutate.Session == nil {
		deps.SessionMutate.Session = fakeSessionClient{}
	}
	if deps.InviteRead.Invite == nil {
		deps.InviteRead.Invite = fakeInviteClient{}
	}
	if deps.InviteRead.Participant == nil {
		deps.InviteRead.Participant = deps.ParticipantRead.Participant
	}
	if deps.InviteRead.Social == nil {
		deps.InviteRead.Social = fakeSocialClient{}
	}
	if deps.InviteRead.Auth == nil {
		deps.InviteRead.Auth = fakeAuthClient{}
	}
	if deps.InviteMutate.Invite == nil {
		deps.InviteMutate.Invite = fakeInviteClient{}
	}
	if deps.InviteMutate.Auth == nil {
		deps.InviteMutate.Auth = fakeAuthClient{}
	}
	if deps.Authorization.Client == nil {
		deps.Authorization.Client = stubAuthorizationClient{}
	}
	return deps
}

// Stubs satisfy client interfaces without being called — only non-nil checks matter.
type stubParticipantReadClient struct {
	campaigngateway.ParticipantReadClient
}
type stubParticipantMutationClient struct {
	campaigngateway.ParticipantMutationClient
}
type stubInteractionClient struct {
	campaigngateway.InteractionClient
}
type stubDiscoveryClient struct {
	campaigngateway.DiscoveryClient
}
type stubAgentClient struct {
	campaigngateway.AgentClient
}
type stubForkClient struct {
	campaigngateway.ForkClient
}
type stubCharacterReadClient struct {
	campaigngateway.CharacterReadClient
}
type stubCharacterMutationClient struct {
	campaigngateway.CharacterMutationClient
}
type stubDaggerheartContentClient struct {
	campaigngateway.DaggerheartContentClient
}
type stubDaggerheartAssetClient struct {
	campaigngateway.DaggerheartAssetClient
}
type stubAuthorizationClient struct {
	campaigngateway.AuthorizationClient
}

type fakeAuthClient struct{}

type fakeParticipantClient struct{}

type fakeSocialClient struct{}

func (fakeAuthClient) LookupUserByUsername(_ context.Context, req *authv1.LookupUserByUsernameRequest, _ ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error) {
	username := strings.TrimSpace(req.GetUsername())
	if username == "" {
		return &authv1.LookupUserByUsernameResponse{}, nil
	}
	return &authv1.LookupUserByUsernameResponse{
		User: &authv1.User{Id: "user-lookup-" + username, Username: username},
	}, nil
}

func (fakeAuthClient) GetUser(_ context.Context, req *authv1.GetUserRequest, _ ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	return &authv1.GetUserResponse{User: &authv1.User{Id: strings.TrimSpace(req.GetUserId()), Username: "user"}}, nil
}

func (fakeAuthClient) IssueJoinGrant(context.Context, *authv1.IssueJoinGrantRequest, ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return &authv1.IssueJoinGrantResponse{JoinGrant: "grant"}, nil
}

func (fakeParticipantClient) ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	return &statev1.ListParticipantsResponse{
		Participants: []*statev1.Participant{{Id: "p1", Name: "Pending Seat"}},
	}, nil
}

func (fakeParticipantClient) GetParticipant(context.Context, *statev1.GetParticipantRequest, ...grpc.CallOption) (*statev1.GetParticipantResponse, error) {
	return &statev1.GetParticipantResponse{Participant: &statev1.Participant{Id: "p1", Name: "Pending Seat"}}, nil
}

func (fakeParticipantClient) CreateParticipant(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
	return &statev1.CreateParticipantResponse{}, nil
}

func (fakeParticipantClient) UpdateParticipant(context.Context, *statev1.UpdateParticipantRequest, ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error) {
	return &statev1.UpdateParticipantResponse{}, nil
}

func (fakeSocialClient) SearchUsers(context.Context, *socialv1.SearchUsersRequest, ...grpc.CallOption) (*socialv1.SearchUsersResponse, error) {
	return &socialv1.SearchUsersResponse{}, nil
}
