package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/message"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type fakeHTMXLocalizer struct{}

func (fakeHTMXLocalizer) Sprintf(key message.Reference, _ ...any) string {
	if s, ok := key.(string); ok {
		return "localized:" + s
	}
	return ""
}

func TestWriteGameContentType(t *testing.T) {
	w := httptest.NewRecorder()

	writeGameContentType(w)

	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want %q", got, "text/html; charset=utf-8")
	}
}

func TestGameRenderersUseGameLayoutMarker(t *testing.T) {
	tests := []struct {
		name   string
		render func(*httptest.ResponseRecorder)
	}{
		{
			name: "campaigns",
			render: func(w *httptest.ResponseRecorder) {
				renderAppCampaignsPage(w, httptest.NewRequest(http.MethodGet, "/campaigns", nil), []*statev1.Campaign{
					{Id: "camp-1", Name: "Campaign One"},
				})
			},
		},
		{
			name: "invites",
			render: func(w *httptest.ResponseRecorder) {
				renderAppInvitesPage(w, httptest.NewRequest(http.MethodGet, "/invites", nil), []*statev1.PendingUserInvite{
					{
						Campaign:    &statev1.Campaign{Id: "camp-1", Name: "Campaign One"},
						Participant: &statev1.Participant{Id: "part-1", Name: "Alice"},
						Invite:      &statev1.Invite{Id: "inv-1", CampaignId: "camp-1"},
					},
				})
			},
		},
		{
			name: "sessions",
			render: func(w *httptest.ResponseRecorder) {
				renderAppCampaignSessionsPage(w, httptest.NewRequest(http.MethodGet, "/campaigns/camp-1/sessions", nil), webtemplates.PageContext{}, "camp-1", []*statev1.Session{
					{Id: "sess-1", Name: "Session One", Status: statev1.SessionStatus_SESSION_ACTIVE},
				}, true)
			},
		},
		{
			name: "session detail",
			render: func(w *httptest.ResponseRecorder) {
				renderAppCampaignSessionDetailPage(w, httptest.NewRequest(http.MethodGet, "/campaigns/camp-1/sessions/sess-1", nil), webtemplates.PageContext{}, "camp-1", &statev1.Session{
					Id:     "sess-1",
					Name:   "Session One",
					Status: statev1.SessionStatus_SESSION_ACTIVE,
				})
			},
		},
		{
			name: "participants",
			render: func(w *httptest.ResponseRecorder) {
				renderAppCampaignParticipantsPage(w, httptest.NewRequest(http.MethodGet, "/campaigns/camp-1/participants", nil), webtemplates.PageContext{}, "camp-1", []*statev1.Participant{
					{Id: "part-1", Name: "Alice"},
				}, true)
			},
		},
		{
			name: "characters",
			render: func(w *httptest.ResponseRecorder) {
				renderAppCampaignCharactersPage(w, httptest.NewRequest(http.MethodGet, "/campaigns/camp-1/characters", nil), webtemplates.PageContext{}, "camp-1", []*statev1.Character{
					{Id: "char-1", Name: "Mira", Kind: statev1.CharacterKind_PC},
				}, true, []*statev1.Participant{
					{Id: "part-1", Name: "Alice"},
				})
			},
		},
		{
			name: "character detail",
			render: func(w *httptest.ResponseRecorder) {
				renderAppCampaignCharacterDetailPage(w, httptest.NewRequest(http.MethodGet, "/campaigns/camp-1/characters/char-1", nil), webtemplates.PageContext{}, "camp-1", &statev1.Character{
					Id:            "char-1",
					Name:          "Mira",
					Kind:          statev1.CharacterKind_PC,
					ParticipantId: wrapperspb.String("part-1"),
				})
			},
		},
		{
			name: "campaign invites",
			render: func(w *httptest.ResponseRecorder) {
				renderAppCampaignInvitesPage(w, httptest.NewRequest(http.MethodGet, "/campaigns/camp-1/invites", nil), webtemplates.PageContext{}, "camp-1", []*statev1.Invite{
					{Id: "inv-1", RecipientUserId: "user-2"},
				}, true)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			tc.render(w)
			body := w.Body.String()
			if !strings.Contains(body, `data-layout="game"`) {
				t.Fatalf("expected game layout marker in %s renderer output", tc.name)
			}
			expectedSuffix := ` | ` + branding.AppName + `</title>`
			if !strings.Contains(body, expectedSuffix) {
				t.Fatalf("expected %s renderer output to include title suffix %q", tc.name, expectedSuffix)
			}
			if strings.Contains(body, ` - `+branding.AppName+` | `+branding.AppName) {
				t.Fatalf("expected %s renderer output to normalize title suffix once", tc.name)
			}
			if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
				t.Fatalf("Content-Type = %q, want %q", got, "text/html; charset=utf-8")
			}
		})
	}
}

func TestGameLayoutIncludesCampaignsTopNavLink(t *testing.T) {
	w := httptest.NewRecorder()
	renderAppCampaignCreatePage(w, httptest.NewRequest(http.MethodGet, "/campaigns/create", nil), webtemplates.PageContext{})

	body := w.Body.String()
	if !strings.Contains(body, `href="/campaigns"`) {
		t.Fatalf("expected campaigns link in top navbar")
	}
}

func TestGameLayoutIncludesBreadcrumbs(t *testing.T) {
	w := httptest.NewRecorder()
	renderAppCampaignSessionDetailPage(w, httptest.NewRequest(http.MethodGet, "/campaigns/camp-1/sessions/sess-1", nil), webtemplates.PageContext{
		CurrentPath: "/campaigns/camp-1/sessions/sess-1",
	}, "camp-1", &statev1.Session{
		Id:     "sess-1",
		Name:   "Session One",
		Status: statev1.SessionStatus_SESSION_ACTIVE,
	})

	body := w.Body.String()
	for _, marker := range []string{
		`href="/campaigns"`,
		`href="/campaigns/camp-1"`,
		`href="/campaigns/camp-1/sessions"`,
		`<li>sess-1</li>`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected breadcrumb marker %q in rendered output", marker)
		}
	}
}

func TestGameLayoutBreadcrumbsUseCampaignName(t *testing.T) {
	w := httptest.NewRecorder()
	renderAppCampaignSessionDetailPage(w, httptest.NewRequest(http.MethodGet, "/campaigns/camp-1/sessions/sess-1", nil), webtemplates.PageContext{
		CurrentPath:  "/campaigns/camp-1/sessions/sess-1",
		CampaignName: "Guildhouse Campaign",
	}, "camp-1", &statev1.Session{
		Id:     "sess-1",
		Name:   "Session One",
		Status: statev1.SessionStatus_SESSION_ACTIVE,
	})

	body := w.Body.String()
	if !strings.Contains(body, "Guildhouse Campaign") {
		t.Fatalf("expected campaign name in breadcrumb, got %q", body)
	}
	if !strings.Contains(body, `href="/campaigns/camp-1"`) {
		t.Fatalf("expected breadcrumb campaign link for campaign ID path")
	}
	if strings.Contains(body, `">camp-1</a>`) {
		t.Fatalf("expected campaign ID to be replaced by campaign name in breadcrumb")
	}
}

func TestWritePageReturnsErrorWhenComponentMissing(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	err := writePage(w, r, nil, "")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, errNoWebPageComponent) {
		t.Fatalf("expected errNoWebPageComponent, got %v", err)
	}
}

func TestWritePageReturnsFullPageWithoutHTMXTitle(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	component := stubComponent{content: "<main>full</main>"}
	if err := writePage(w, r, component, "<title>Injected Title</title>"); err != nil {
		t.Fatalf("writePage() = %v", err)
	}
	if got := w.Body.String(); got != "<main>full</main>" {
		t.Fatalf("response body = %q, want %q", got, "<main>full</main>")
	}
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want %q", got, "text/html; charset=utf-8")
	}
}

func TestWritePageInjectsTitleForHTMXRequests(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set("HX-Request", "true")

	component := stubComponent{content: "<main>fragment</main>"}
	if err := writePage(w, r, component, "<title>Injected Title</title>"); err != nil {
		t.Fatalf("writePage() = %v", err)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<title>Injected Title</title>") {
		t.Fatalf("expected injected title in HTMX response, got %q", body)
	}
	if !strings.HasSuffix(body, "fragment") {
		t.Fatalf("expected rendered fragment body in HTMX response, got %q", body)
	}
}

func TestComposeHTMXTitleForPageUsesLocalizedTitle(t *testing.T) {
	page := webtemplates.PageContext{
		Loc: fakeHTMXLocalizer{},
	}
	got := composeHTMXTitleForPage(page, "layout.profile")

	if got == "" {
		t.Fatal("expected non-empty htmx title")
	}
	if !strings.HasPrefix(got, "<title>") || !strings.HasSuffix(got, "</title>") {
		t.Fatalf("expected composed title tag, got %q", got)
	}
	if !strings.Contains(got, "localized:layout.profile | "+branding.AppName) {
		t.Fatalf("expected composed title to use composed localized title, got %q", got)
	}
}

func TestChatFallbackPort(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{name: "empty", addr: "", want: ""},
		{name: "raw port", addr: "8086", want: "8086"},
		{name: "localhost", addr: "localhost:8086", want: "8086"},
		{name: "loopback ipv4", addr: "127.0.0.1:8086", want: "8086"},
		{name: "loopback ipv6", addr: "[::1]:8086", want: "8086"},
		{name: "ipv6 host", addr: "[2001:db8::1]:8086", want: "8086"},
		{name: "hostless", addr: ":8086", want: "8086"},
		{name: "whitespace", addr: "  localhost:8086  ", want: "8086"},
		{name: "no port", addr: "localhost", want: ""},
		{name: "invalid port", addr: "localhost:not-a-port", want: ""},
		{name: "out of range high", addr: "localhost:65536", want: ""},
		{name: "out of range low", addr: "localhost:0", want: ""},
		{name: "multiple colons non ipv6", addr: "foo:bar:baz", want: ""},
		{name: "bare ipv6 no brackets", addr: "2001:db8::1", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := chatFallbackPort(tc.addr)
			if got != tc.want {
				t.Fatalf("chatFallbackPort(%q) = %q, want %q", tc.addr, got, tc.want)
			}
		})
	}
}

func TestSanitizePort(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{name: "space", raw: " ", want: ""},
		{name: "minimum", raw: "1", want: "1"},
		{name: "maximum", raw: "65535", want: "65535"},
		{name: "trimmed", raw: " 8086 ", want: "8086"},
		{name: "zero", raw: "0", want: ""},
		{name: "negative", raw: "-1", want: ""},
		{name: "too large", raw: "65536", want: ""},
		{name: "alpha", raw: "abc", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizePort(tc.raw)
			if got != tc.want {
				t.Fatalf("sanitizePort(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

type stubComponent struct {
	content string
}

func (s stubComponent) Render(_ context.Context, w io.Writer) error {
	_, err := w.Write([]byte(s.content))
	return err
}
