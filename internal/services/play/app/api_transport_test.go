package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

func TestParseChatHistoryPage(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		page, err := parseChatHistoryPage(httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/chat/history", nil))
		if err != nil {
			t.Fatalf("parseChatHistoryPage(defaults) error = %v", err)
		}
		if page.BeforeSequenceID != 1<<62 {
			t.Fatalf("BeforeSequenceID = %d, want %d", page.BeforeSequenceID, int64(1<<62))
		}
		if page.Limit != transcript.DefaultHistoryLimit {
			t.Fatalf("Limit = %d, want %d", page.Limit, transcript.DefaultHistoryLimit)
		}
	})

	t.Run("explicit values", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/chat/history?before_seq=9&limit=2", nil)
		page, err := parseChatHistoryPage(req)
		if err != nil {
			t.Fatalf("parseChatHistoryPage(explicit values) error = %v", err)
		}
		if page.BeforeSequenceID != 9 || page.Limit != 2 {
			t.Fatalf("page = %#v", page)
		}
	})

	t.Run("invalid before sequence", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/chat/history?before_seq=oops", nil)
		if _, err := parseChatHistoryPage(req); err != errInvalidBeforeSequence {
			t.Fatalf("parseChatHistoryPage(invalid before) error = %v, want %v", err, errInvalidBeforeSequence)
		}
	})

	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/chat/history?limit=oops", nil)
		if _, err := parseChatHistoryPage(req); err != errInvalidLimit {
			t.Fatalf("parseChatHistoryPage(invalid limit) error = %v, want %v", err, errInvalidLimit)
		}
	})
}

func TestRequirePlayRequest(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		server := &Server{auth: &fakePlayAuthClient{sessions: map[string]string{"ps-1": "user-1"}}}
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/bootstrap", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		got, ok := server.requirePlayRequest(rr, req)
		if !ok {
			t.Fatal("requirePlayRequest() reported failure")
		}
		if got.CampaignID != "c1" || got.UserID != "user-1" {
			t.Fatalf("request = %#v", got)
		}
	})

	t.Run("missing campaign id returns not found", func(t *testing.T) {
		t.Parallel()

		server := &Server{}
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/bootstrap", nil)
		rr := httptest.NewRecorder()

		if _, ok := server.requirePlayRequest(rr, req); ok {
			t.Fatal("requirePlayRequest() unexpectedly succeeded")
		}
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("missing play session returns unauthorized", func(t *testing.T) {
		t.Parallel()

		server := &Server{auth: &fakePlayAuthClient{}}
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/bootstrap", nil)
		req.SetPathValue("campaignID", "c1")
		rr := httptest.NewRecorder()

		if _, ok := server.requirePlayRequest(rr, req); ok {
			t.Fatal("requirePlayRequest() unexpectedly succeeded")
		}
		assertJSONError(t, rr, http.StatusUnauthorized, "authentication required")
	})
}
