package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

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
			if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
				t.Fatalf("Content-Type = %q, want %q", got, "text/html; charset=utf-8")
			}
		})
	}
}
