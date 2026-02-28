package flash

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteAndReadAndClearRoundTrip(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	writeRR := httptest.NewRecorder()

	Write(writeRR, req, NoticeSuccess("web.settings.user_profile.notice_saved"))
	setCookieHeader := writeRR.Header().Get("Set-Cookie")
	if setCookieHeader == "" {
		t.Fatalf("expected Set-Cookie header")
	}
	cookie, err := http.ParseSetCookie(setCookieHeader)
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	req.AddCookie(cookie)

	readRR := httptest.NewRecorder()
	notice, ok := ReadAndClear(readRR, req)
	if !ok {
		t.Fatalf("ReadAndClear() ok = false, want true")
	}
	if notice.Kind != KindSuccess {
		t.Fatalf("notice.Kind = %q, want %q", notice.Kind, KindSuccess)
	}
	if notice.Key != "web.settings.user_profile.notice_saved" {
		t.Fatalf("notice.Key = %q", notice.Key)
	}
	cleared := readRR.Header().Get("Set-Cookie")
	if cleared == "" {
		t.Fatalf("expected clear Set-Cookie header")
	}
}

func TestReadAndClearInvalidCookieValueStillClears(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "not-base64"})
	rr := httptest.NewRecorder()

	_, ok := ReadAndClear(rr, req)
	if ok {
		t.Fatalf("ReadAndClear() ok = true, want false")
	}
	if rr.Header().Get("Set-Cookie") == "" {
		t.Fatalf("expected clear Set-Cookie header")
	}
}

func TestWriteIgnoresInvalidNotice(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	rr := httptest.NewRecorder()

	Write(rr, req, Notice{Kind: KindSuccess, Key: ""})
	if got := rr.Header().Get("Set-Cookie"); got != "" {
		t.Fatalf("Set-Cookie = %q, want empty", got)
	}
}
