package campaigns

import (
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
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMountCampaignsPageRendersCardGridWithCover(t *testing.T) {
	t.Parallel()

	deps := completeGRPCDeps(GRPCGatewayDeps{
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
	})
	m := NewStableWithGateway(NewGRPCGateway(deps), modulehandler.NewTestBase(), "", nil)

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

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{
		ID:               "camp/1",
		Name:             "Escaped Campaign",
		ParticipantCount: "1",
		CharacterCount:   "1",
	}}}, modulehandler.NewTestBase(), "", nil)
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
	body := rr.Body.String()
	if !strings.Contains(body, `href="/app/campaigns/camp%2F1"`) {
		t.Fatalf("body missing escaped campaign route: %q", body)
	}
}

func TestMountCampaignsPageRendersCardIconsFromCatalog(t *testing.T) {
	t.Parallel()

	deps := completeGRPCDeps(GRPCGatewayDeps{
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
	})
	m := NewStableWithGateway(NewGRPCGateway(deps), modulehandler.NewTestBase(), "", nil)

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

func TestMountCampaignsPageRendersHeadingWithStartLink(t *testing.T) {
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

	body := rr.Body.String()
	// Invariant: page roots (campaigns index) should not render breadcrumb trails.
	if strings.Contains(body, `class="breadcrumbs text-sm"`) {
		t.Fatalf("expected no breadcrumbs on campaigns root, got %q", body)
	}
}

func TestMountCampaignsHTMXRendersHeadingWithStartLink(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), "", nil)
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

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
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

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
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

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}, createdCampaignID: "camp-777"}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
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
	m := NewStableWithGateway(gateway, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
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
	m := NewStableWithGateway(gateway, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
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
	if got := gateway.lastCreateInput.System; got != GameSystemDaggerheart {
		t.Fatalf("System = %v, want %v", got, GameSystemDaggerheart)
	}
	if got := gateway.lastCreateInput.GMMode; got != GmModeHuman {
		t.Fatalf("GMMode = %v, want %v", got, GmModeHuman)
	}
}

func TestMountCampaignCreatePostRejectsEmptyName(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "First"}}, createdCampaignID: "camp-777"}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
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

	m := NewStableWithGateway(fakeGateway{}, modulehandler.NewBase(nil, func(*http.Request) string { return "pt-BR" }, nil), "", nil)
	mount, err := m.Mount()
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
	deps := completeGRPCDeps(GRPCGatewayDeps{
		CampaignClient: client,
	})
	m := NewStableWithGateway(NewGRPCGateway(deps), modulehandler.NewBase(nil, func(*http.Request) string { return "pt-BR" }, nil), "", nil)
	mount, err := m.Mount()
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

func TestMountCampaignCreatePostRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
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

	m := NewStableWithGateway(fakeGateway{}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
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

	m := NewStableWithGateway(fakeGateway{createErr: apperrors.E(apperrors.KindForbidden, "forbidden")}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
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

	m := NewStableWithGateway(fakeGateway{}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
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

func TestParseAppGameSystemAndGmMode(t *testing.T) {
	t.Parallel()

	if system, ok := parseAppGameSystem("daggerheart"); !ok || system != GameSystemDaggerheart {
		t.Fatalf("parseAppGameSystem daggerheart = (%v, %v)", system, ok)
	}
	if _, ok := parseAppGameSystem("unknown"); ok {
		t.Fatalf("expected unknown game system to fail parse")
	}

	if mode, ok := parseAppGmMode("ai"); !ok || mode != GmModeAI {
		t.Fatalf("parseAppGmMode ai = (%v, %v)", mode, ok)
	}
	if mode, ok := parseAppGmMode("hybrid"); !ok || mode != GmModeHybrid {
		t.Fatalf("parseAppGmMode hybrid = (%v, %v)", mode, ok)
	}
	if _, ok := parseAppGmMode("invalid"); ok {
		t.Fatalf("expected invalid gm mode to fail parse")
	}
}
