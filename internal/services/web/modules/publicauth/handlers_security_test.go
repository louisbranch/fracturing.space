package publicauth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestResolveSafeRedirectPathRejectsUnsafeTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "external url", input: "https://evil.example/app/campaigns", want: ""},
		{name: "app root", input: routepath.AppPrefix, want: ""},
		{name: "invite root", input: strings.TrimRight(routepath.InvitePrefix, "/"), want: ""},
		{name: "encoded slash", input: "/app/campaigns/%2fadmin", want: ""},
		{name: "dot segment", input: "/app/../settings", want: ""},
		{name: "valid app path", input: "/app/campaigns/camp-1?tab=people", want: "/app/campaigns/camp-1?tab=people"},
		{name: "valid invite path", input: "/invite/inv-1", want: "/invite/inv-1"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveSafeRedirectPath(tc.input); got != tc.want {
				t.Fatalf("resolveSafeRedirectPath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestHandleRecoveryCodeGetConsumesRevealCookie(t *testing.T) {
	t.Parallel()

	h := newHandlers(publicauthapp.NewService(nil, ""), requestmeta.SchemePolicy{})

	seedReq := httptest.NewRequest(http.MethodGet, routepath.LoginRecoveryCode, nil)
	seedResp := httptest.NewRecorder()
	writeRecoveryRevealState(seedResp, seedReq, requestmeta.SchemePolicy{}, recoveryRevealState{
		Code:      "ABCD-EFGH",
		PendingID: "pending-1",
		Mode:      recoveryRevealModeRecovery,
	})

	var revealCookie *http.Cookie
	for _, cookie := range seedResp.Result().Cookies() {
		if cookie.Name == recoveryRevealCookieName {
			revealCookie = cookie
			break
		}
	}
	if revealCookie == nil {
		t.Fatal("expected reveal cookie")
	}

	req := httptest.NewRequest(http.MethodGet, routepath.LoginRecoveryCode, nil)
	req.AddCookie(revealCookie)
	rr := httptest.NewRecorder()

	h.handleRecoveryCodeGet(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "ABCD-EFGH") {
		t.Fatalf("body missing recovery code: %s", rr.Body.String())
	}

	var clearedCookie *http.Cookie
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == recoveryRevealCookieName {
			clearedCookie = cookie
			break
		}
	}
	if clearedCookie == nil {
		t.Fatal("expected recovery reveal cookie to be cleared")
	}
	if clearedCookie.MaxAge != -1 {
		t.Fatalf("cleared cookie MaxAge = %d, want -1", clearedCookie.MaxAge)
	}
}
