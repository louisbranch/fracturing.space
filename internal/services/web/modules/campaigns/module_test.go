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

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
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
	if !New(Config{Gateway: fakeGateway{}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil}).Healthy() {
		t.Fatalf("New(Config{...}).Healthy() = false, want true")
	}
}

func TestMapCampaignCharacterCreationStepToProtoWrapper(t *testing.T) {
	t.Parallel()

	step := &campaignapp.CampaignCharacterCreationStepInput{
		Details: &campaignapp.CampaignCharacterCreationStepDetails{},
	}
	mapped, err := mapCampaignCharacterCreationStepToProto(step)
	if err != nil {
		t.Fatalf("mapCampaignCharacterCreationStepToProto() error = %v", err)
	}
	if mapped == nil {
		t.Fatalf("mapCampaignCharacterCreationStepToProto() = nil, want non-nil")
	}
}

func TestMountServesCampaignsGet(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}, {ID: "c2", Name: "Second"}}}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
	mount, err := m.Mount()
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

	m := New(Config{})
	mount, err := m.Mount()
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

	m := New(Config{})
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.CampaignsPrefix+"123", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestMountMapsCampaignGatewayErrorToHTTPStatus(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: fakeGateway{err: apperrors.E(apperrors.KindUnauthorized, "missing session")}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
	mount, err := m.Mount()
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

	m := New(Config{Gateway: fakeGateway{err: status.Error(codes.NotFound, "campaign not found")}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
	mount, err := m.Mount()
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

	m := New(Config{Gateway: fakeGateway{err: errors.New("boom")}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
	mount, err := m.Mount()
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

	m := New(Config{Gateway: fakeGateway{err: status.Error(codes.NotFound, "campaign not found")}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
	mount, err := m.Mount()
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

	m := New(Config{Gateway: fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
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

	m := New(Config{Gateway: fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
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

	m := New(Config{Gateway: fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
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

	deps := completeGRPCDeps(GRPCGatewayDeps{
		CampaignClient: fakeCampaignClient{
			response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "remote-1", Name: "Remote Campaign"}}},
		},
	})
	m := New(Config{Gateway: NewGRPCGateway(deps), Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
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
	if body := rr.Body.String(); !strings.Contains(body, "Remote Campaign") {
		t.Fatalf("body = %q, want remote campaign", body)
	}
}

func TestMountServesCampaignsGetWithEmptyList(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: fakeGateway{items: []CampaignSummary{}}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
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

	m := New(Config{Gateway: fakeGateway{items: []CampaignSummary{}}, Base: modulehandler.NewTestBase(), ChatFallbackPort: "", Workflows: nil})
	mount, err := m.Mount()
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

	h := newHandlers(newService(fakeGateway{}), modulehandler.NewTestBase(), "", nil)
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
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

	g := grpcGateway{Client: fakeCampaignClient{getResp: &statev1.GetCampaignResponse{Campaign: nil}}}
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

	g := grpcGateway{Client: fakeCampaignClient{createResp: &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{}}}}
	_, err := g.CreateCampaign(context.Background(), CreateCampaignInput{Name: "New", System: GameSystemDaggerheart, GMMode: GmModeHuman})
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
		{name: "start session", err: g.StartSession(context.Background(), "c1", StartSessionInput{Name: "Session One"})},
		{name: "end session", err: g.EndSession(context.Background(), "c1", EndSessionInput{SessionID: "sess-1"})},
		{name: "create character", err: func() error {
			_, err := g.CreateCharacter(context.Background(), "c1", CreateCharacterInput{Name: "Hero", Kind: CharacterKindPC})
			return err
		}()},
		{name: "apply character creation step", err: g.ApplyCharacterCreationStep(context.Background(), "c1", "char-1", &CampaignCharacterCreationStepInput{Details: &CampaignCharacterCreationStepDetails{}})},
		{name: "reset character creation workflow", err: g.ResetCharacterCreationWorkflow(context.Background(), "c1", "char-1")},
		{name: "create invite", err: g.CreateInvite(context.Background(), "c1", CreateInviteInput{ParticipantID: "p-1", RecipientUserID: "user-2"})},
		{name: "revoke invite", err: g.RevokeInvite(context.Background(), "c1", RevokeInviteInput{InviteID: "inv-1"})},
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

	g := grpcGateway{SessionClient: fakeSessionClient{response: &statev1.ListSessionsResponse{Sessions: []*statev1.Session{{
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

	g := grpcGateway{InviteClient: fakeInviteClient{response: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{{
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

// testCreationWorkflow implements CharacterCreationWorkflow for tests.
// It passes through catalog data without Daggerheart-specific filtering,
// so test data should only include entries expected in output.
type testCreationWorkflow struct{}

func (testCreationWorkflow) AssembleCatalog(
	progress CampaignCharacterCreationProgress,
	catalog CampaignCharacterCreationCatalog,
	profile CampaignCharacterCreationProfile,
) CampaignCharacterCreation {
	creation := CampaignCharacterCreation{
		Progress: progress,
		Profile:  profile,
	}
	for _, c := range catalog.Classes {
		creation.Classes = append(creation.Classes, c)
	}
	for _, s := range catalog.Subclasses {
		creation.Subclasses = append(creation.Subclasses, s)
	}
	for _, h := range catalog.Heritages {
		entry := CatalogHeritage{ID: h.ID, Name: h.Name, Kind: h.Kind}
		switch h.Kind {
		case "ancestry":
			creation.Ancestries = append(creation.Ancestries, entry)
		case "community":
			creation.Communities = append(creation.Communities, entry)
		}
	}
	for _, w := range catalog.Weapons {
		entry := CatalogWeapon{ID: w.ID, Name: w.Name, Category: w.Category, Tier: w.Tier}
		switch w.Category {
		case "primary":
			creation.PrimaryWeapons = append(creation.PrimaryWeapons, entry)
		case "secondary":
			creation.SecondaryWeapons = append(creation.SecondaryWeapons, entry)
		}
	}
	for _, a := range catalog.Armor {
		creation.Armor = append(creation.Armor, a)
	}
	for _, i := range catalog.Items {
		creation.PotionItems = append(creation.PotionItems, i)
	}
	for _, d := range catalog.DomainCards {
		creation.DomainCards = append(creation.DomainCards, d)
	}
	return creation
}

func (testCreationWorkflow) CreationView(creation CampaignCharacterCreation) webtemplates.CampaignCharacterCreationView {
	view := webtemplates.CampaignCharacterCreationView{
		Ready:             creation.Progress.Ready,
		NextStep:          creation.Progress.NextStep,
		UnmetReasons:      append([]string(nil), creation.Progress.UnmetReasons...),
		ClassID:           creation.Profile.ClassID,
		SubclassID:        creation.Profile.SubclassID,
		AncestryID:        creation.Profile.AncestryID,
		CommunityID:       creation.Profile.CommunityID,
		Agility:           creation.Profile.Agility,
		Strength:          creation.Profile.Strength,
		Finesse:           creation.Profile.Finesse,
		Instinct:          creation.Profile.Instinct,
		Presence:          creation.Profile.Presence,
		Knowledge:         creation.Profile.Knowledge,
		PrimaryWeaponID:   creation.Profile.PrimaryWeaponID,
		SecondaryWeaponID: creation.Profile.SecondaryWeaponID,
		ArmorID:           creation.Profile.ArmorID,
		PotionItemID:      creation.Profile.PotionItemID,
		Background:        creation.Profile.Background,
		DomainCardIDs:     append([]string(nil), creation.Profile.DomainCardIDs...),
		Connections:       creation.Profile.Connections,
		Steps:             make([]webtemplates.CampaignCharacterCreationStepView, 0, len(creation.Progress.Steps)),
		Classes:           make([]webtemplates.CampaignCreationClassView, 0, len(creation.Classes)),
		Subclasses:        make([]webtemplates.CampaignCreationSubclassView, 0, len(creation.Subclasses)),
		Ancestries:        make([]webtemplates.CampaignCreationHeritageView, 0, len(creation.Ancestries)),
		Communities:       make([]webtemplates.CampaignCreationHeritageView, 0, len(creation.Communities)),
		PrimaryWeapons:    make([]webtemplates.CampaignCreationWeaponView, 0, len(creation.PrimaryWeapons)),
		SecondaryWeapons:  make([]webtemplates.CampaignCreationWeaponView, 0, len(creation.SecondaryWeapons)),
		Armor:             make([]webtemplates.CampaignCreationArmorView, 0, len(creation.Armor)),
		PotionItems:       make([]webtemplates.CampaignCreationItemView, 0, len(creation.PotionItems)),
		DomainCards:       make([]webtemplates.CampaignCreationDomainCardView, 0, len(creation.DomainCards)),
	}
	for _, step := range creation.Progress.Steps {
		view.Steps = append(view.Steps, webtemplates.CampaignCharacterCreationStepView{Step: step.Step, Key: step.Key, Complete: step.Complete})
	}
	for _, class := range creation.Classes {
		view.Classes = append(view.Classes, webtemplates.CampaignCreationClassView{ID: class.ID, Name: class.Name})
	}
	for _, subclass := range creation.Subclasses {
		view.Subclasses = append(view.Subclasses, webtemplates.CampaignCreationSubclassView{ID: subclass.ID, Name: subclass.Name, ClassID: subclass.ClassID})
	}
	for _, ancestry := range creation.Ancestries {
		view.Ancestries = append(view.Ancestries, webtemplates.CampaignCreationHeritageView{ID: ancestry.ID, Name: ancestry.Name})
	}
	for _, community := range creation.Communities {
		view.Communities = append(view.Communities, webtemplates.CampaignCreationHeritageView{ID: community.ID, Name: community.Name})
	}
	for _, weapon := range creation.PrimaryWeapons {
		view.PrimaryWeapons = append(view.PrimaryWeapons, webtemplates.CampaignCreationWeaponView{ID: weapon.ID, Name: weapon.Name})
	}
	for _, weapon := range creation.SecondaryWeapons {
		view.SecondaryWeapons = append(view.SecondaryWeapons, webtemplates.CampaignCreationWeaponView{ID: weapon.ID, Name: weapon.Name})
	}
	for _, armor := range creation.Armor {
		view.Armor = append(view.Armor, webtemplates.CampaignCreationArmorView{ID: armor.ID, Name: armor.Name})
	}
	for _, item := range creation.PotionItems {
		view.PotionItems = append(view.PotionItems, webtemplates.CampaignCreationItemView{ID: item.ID, Name: item.Name})
	}
	for _, card := range creation.DomainCards {
		view.DomainCards = append(view.DomainCards, webtemplates.CampaignCreationDomainCardView{ID: card.ID, Name: card.Name, DomainID: card.DomainID, Level: card.Level})
	}
	return view
}

func (testCreationWorkflow) ParseStepInput(form url.Values, nextStep int32) (*CampaignCharacterCreationStepInput, error) {
	switch nextStep {
	case 1:
		classID := strings.TrimSpace(form.Get("class_id"))
		subclassID := strings.TrimSpace(form.Get("subclass_id"))
		if classID == "" || subclassID == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_class_and_subclass_are_required", "class and subclass are required")
		}
		return &CampaignCharacterCreationStepInput{
			ClassSubclass: &CampaignCharacterCreationStepClassSubclass{ClassID: classID, SubclassID: subclassID},
		}, nil
	default:
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}
}

// defaultTestWorkflows returns a workflow map suitable for tests that need
// character creation enabled for Daggerheart campaigns.
func defaultTestWorkflows() map[GameSystem]CharacterCreationWorkflow {
	return map[GameSystem]CharacterCreationWorkflow{GameSystemDaggerheart: testCreationWorkflow{}}
}

type fakeGateway struct {
	items                             []CampaignSummary
	workspaceSystem                   string
	workspaceStatus                   string
	workspaceLocale                   string
	workspaceIntent                   string
	workspaceAccessPolicy             string
	workspaceAIAgentID                string
	campaignAIAgents                  []CampaignAIAgentOption
	campaignAIAgentsErr               error
	participants                      []CampaignParticipant
	participantsErr                   error
	participant                       CampaignParticipant
	participantErr                    error
	characters                        []CampaignCharacter
	charactersErr                     error
	sessions                          []CampaignSession
	sessionsErr                       error
	sessionReadiness                  CampaignSessionReadiness
	sessionReadinessErr               error
	invites                           []CampaignInvite
	invitesErr                        error
	characterCreationProgress         CampaignCharacterCreationProgress
	characterCreationProgressErr      error
	characterCreationCatalog          CampaignCharacterCreationCatalog
	characterCreationCatalogErr       error
	characterCreationProfile          CampaignCharacterCreationProfile
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
	updateParticipantErr              error
	err                               error
	createErr                         error
	createdCampaignID                 string
}

type mutationContextGateway struct {
	fakeGateway
	startSessionUserID string
}

func (g *mutationContextGateway) StartSession(ctx context.Context, _ string, _ StartSessionInput) error {
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
		system := strings.TrimSpace(f.workspaceSystem)
		if system == "" {
			system = "Daggerheart"
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
		return CampaignWorkspace{
			ID:               campaignID,
			Name:             name,
			Theme:            strings.TrimSpace(item.Theme),
			System:           system,
			GMMode:           "Human",
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
	return CampaignWorkspace{}, apperrors.E(apperrors.KindNotFound, "campaign not found")
}

func (f fakeGateway) CampaignAIAgents(context.Context) ([]CampaignAIAgentOption, error) {
	if f.campaignAIAgentsErr != nil {
		return nil, f.campaignAIAgentsErr
	}
	return f.campaignAIAgents, nil
}

func (f fakeGateway) CampaignParticipants(context.Context, string) ([]CampaignParticipant, error) {
	if f.participantsErr != nil {
		return nil, f.participantsErr
	}
	return f.participants, nil
}

func (f fakeGateway) CampaignParticipant(context.Context, string, string) (CampaignParticipant, error) {
	if f.participantErr != nil {
		return CampaignParticipant{}, f.participantErr
	}
	if strings.TrimSpace(f.participant.ID) != "" {
		return f.participant, nil
	}
	if len(f.participants) > 0 {
		return f.participants[0], nil
	}
	return CampaignParticipant{}, nil
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

func (f fakeGateway) CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error) {
	if f.sessionReadinessErr != nil {
		return CampaignSessionReadiness{}, f.sessionReadinessErr
	}
	if !f.sessionReadiness.Ready && len(f.sessionReadiness.Blockers) == 0 {
		return CampaignSessionReadiness{Ready: true, Blockers: []CampaignSessionReadinessBlocker{}}, nil
	}
	return f.sessionReadiness, nil
}

func (f fakeGateway) CampaignInvites(context.Context, string) ([]CampaignInvite, error) {
	if f.invitesErr != nil {
		return nil, f.invitesErr
	}
	return f.invites, nil
}

func (f fakeGateway) CharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error) {
	if f.characterCreationProgressErr != nil {
		return CampaignCharacterCreationProgress{}, f.characterCreationProgressErr
	}
	return f.characterCreationProgress, nil
}

func (f fakeGateway) CharacterCreationCatalog(context.Context, language.Tag) (CampaignCharacterCreationCatalog, error) {
	if f.characterCreationCatalogErr != nil {
		return CampaignCharacterCreationCatalog{}, f.characterCreationCatalogErr
	}
	return f.characterCreationCatalog, nil
}

func (f fakeGateway) CharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error) {
	if f.characterCreationProfileErr != nil {
		return CampaignCharacterCreationProfile{}, f.characterCreationProfileErr
	}
	return f.characterCreationProfile, nil
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

func (f fakeGateway) UpdateCampaign(context.Context, string, UpdateCampaignInput) error {
	return f.updateCampaignErr
}

func (f fakeGateway) UpdateCampaignAIBinding(context.Context, string, UpdateCampaignAIBindingInput) error {
	return f.updateCampaignAIBindingErr
}

func (fakeGateway) StartSession(context.Context, string, StartSessionInput) error { return nil }
func (fakeGateway) EndSession(context.Context, string, EndSessionInput) error     { return nil }
func (f fakeGateway) CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error) {
	if f.createCharacterErr != nil {
		return CreateCharacterResult{}, f.createCharacterErr
	}
	createdCharacterID := strings.TrimSpace(f.createdCharacterID)
	if createdCharacterID == "" {
		createdCharacterID = "char-created"
	}
	return CreateCharacterResult{CharacterID: createdCharacterID}, nil
}
func (fakeGateway) UpdateCharacter(context.Context, string, string, UpdateCharacterInput) error {
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
func (f fakeGateway) UpdateParticipant(context.Context, string, UpdateParticipantInput) error {
	return f.updateParticipantErr
}
func (fakeGateway) CreateInvite(context.Context, string, CreateInviteInput) error { return nil }
func (fakeGateway) RevokeInvite(context.Context, string, RevokeInviteInput) error { return nil }
func (f fakeGateway) ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error {
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

// completeGRPCDeps fills in stub clients for any nil required fields so that
// NewGRPCGateway returns a grpcGateway instead of unavailableGateway.
// Tests only need to set the clients they exercise.
func completeGRPCDeps(deps GRPCGatewayDeps) GRPCGatewayDeps {
	if deps.CampaignClient == nil {
		deps.CampaignClient = fakeCampaignClient{}
	}
	if deps.ParticipantClient == nil {
		deps.ParticipantClient = stubParticipantClient{}
	}
	if deps.CharacterClient == nil {
		deps.CharacterClient = stubCharacterClient{}
	}
	if deps.DaggerheartContentClient == nil {
		deps.DaggerheartContentClient = stubDaggerheartContentClient{}
	}
	if deps.DaggerheartAssetClient == nil {
		deps.DaggerheartAssetClient = stubDaggerheartAssetClient{}
	}
	if deps.SessionClient == nil {
		deps.SessionClient = fakeSessionClient{}
	}
	if deps.InviteClient == nil {
		deps.InviteClient = fakeInviteClient{}
	}
	if deps.AuthorizationClient == nil {
		deps.AuthorizationClient = stubAuthorizationClient{}
	}
	return deps
}

// Stubs satisfy client interfaces without being called — only non-nil checks matter.
type stubParticipantClient struct{ ParticipantClient }
type stubCharacterClient struct{ CharacterClient }
type stubDaggerheartContentClient struct{ DaggerheartContentClient }
type stubDaggerheartAssetClient struct{ DaggerheartAssetClient }
type stubAuthorizationClient struct{ AuthorizationClient }
