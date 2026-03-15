package publicauth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/redirectpath"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestHandleRecoveryCodeGetPreservesRevealCookieUntilAcknowledge(t *testing.T) {
	t.Parallel()

	h := newHandlersFromGateway(nil, "", requestmeta.SchemePolicy{})

	if got := redirectpath.ResolveSafe("/app/dashboard"); got != "/app/dashboard" {
		t.Fatalf("ResolveSafe sanity check = %q, want %q", got, "/app/dashboard")
	}

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

	var preservedCookie *http.Cookie
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == recoveryRevealCookieName {
			preservedCookie = cookie
			break
		}
	}
	if preservedCookie != nil && preservedCookie.MaxAge == -1 {
		t.Fatalf("reveal cookie was unexpectedly cleared")
	}
}
