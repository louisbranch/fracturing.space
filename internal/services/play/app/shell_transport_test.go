package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleCampaignShellInvalidLaunchRedirectsToWebAndClearsCookie(t *testing.T) {
	t.Parallel()

	server := &Server{
		deps: Dependencies{Auth: &fakePlayAuthClient{sessions: map[string]string{"stale-session": "stale-user"}}},
	}
	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/campaigns/c1?launch=bad-token", nil)
	req.SetPathValue("campaignID", "c1")
	req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "stale-session"})
	rr := httptest.NewRecorder()

	server.handleCampaignShell(rr, req, testPlayLaunchGrantConfig(t))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "http://example.com/app/campaigns/c1" {
		t.Fatalf("Location = %q, want %q", got, "http://example.com/app/campaigns/c1")
	}
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != playSessionCookieName || cookies[0].MaxAge != -1 {
		t.Fatalf("cookies = %#v, want cleared play_session cookie", cookies)
	}
}

func TestHandleCampaignShellStalePlaySessionRedirectsToWebAndClearsCookie(t *testing.T) {
	t.Parallel()

	server := &Server{
		deps: Dependencies{Auth: &fakePlayAuthClient{sessions: map[string]string{}}},
	}
	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/campaigns/c1", nil)
	req.SetPathValue("campaignID", "c1")
	req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "stale-session"})
	rr := httptest.NewRecorder()

	server.handleCampaignShell(rr, req, testPlayLaunchGrantConfig(t))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "http://example.com/app/campaigns/c1" {
		t.Fatalf("Location = %q, want %q", got, "http://example.com/app/campaigns/c1")
	}
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != playSessionCookieName || cookies[0].MaxAge != -1 {
		t.Fatalf("cookies = %#v, want cleared play_session cookie", cookies)
	}
}
