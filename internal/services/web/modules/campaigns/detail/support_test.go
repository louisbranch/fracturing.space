package detail

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

type detailSync struct{}

func (detailSync) ProfileSaved(_ any, _ string)              {}
func (detailSync) CampaignCreated(_ any, _, _ string)        {}
func (detailSync) SessionStarted(_ any, _, _ string)         {}
func (detailSync) SessionEnded(_ any, _, _ string)           {}
func (detailSync) InviteChanged(_ any, _ []string, _ string) {}

func TestNewSupportDefaultsAndAccessors(t *testing.T) {
	t.Parallel()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		func(*http.Request) module.Viewer { return module.Viewer{} },
	)
	meta := requestmeta.SchemePolicy{TrustForwardedProto: true}
	support := NewSupport(base, meta, nil)

	if _, ok := support.Sync().(dashboardsync.Noop); !ok {
		t.Fatalf("Sync() = %#v, want dashboardsync.Noop", support.Sync())
	}
	if support.RequestMeta() != meta {
		t.Fatalf("RequestMeta() = %#v, want %#v", support.RequestMeta(), meta)
	}
	if got := support.Now(); got.IsZero() {
		t.Fatal("Now() unexpectedly returned zero time")
	}
}

func TestSupportNowUsesInjectedClock(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, 3, 23, 13, 0, 0, 0, time.UTC)
	support := Support{nowFunc: func() time.Time { return fixed }}
	if got := support.Now(); !got.Equal(fixed) {
		t.Fatalf("Now() = %v, want %v", got, fixed)
	}
}

func TestWriteMutationErrorUsesLocalizedErrorKeyAndRedirect(t *testing.T) {
	t.Parallel()

	support := NewSupport(modulehandler.NewBase(nil, nil, nil), requestmeta.SchemePolicy{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/c1", nil)
	rr := httptest.NewRecorder()

	support.WriteMutationError(rr, req, apperrors.EK(apperrors.KindInvalidInput, "error.web.custom", "boom"), "fallback.key", "/target")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/target" {
		t.Fatalf("Location = %q, want %q", got, "/target")
	}
	notice := flashNoticeFromRecorder(t, rr)
	if notice.Kind != flash.KindError || notice.Key != "error.web.custom" {
		t.Fatalf("flash notice = %#v", notice)
	}
}

func TestWriteMutationSuccessUsesSuccessNoticeAndRedirect(t *testing.T) {
	t.Parallel()

	support := NewSupport(modulehandler.NewBase(nil, nil, nil), requestmeta.SchemePolicy{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/c1", nil)
	rr := httptest.NewRecorder()

	support.WriteMutationSuccess(rr, req, "web.notice.saved", "/saved")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/saved" {
		t.Fatalf("Location = %q, want %q", got, "/saved")
	}
	notice := flashNoticeFromRecorder(t, rr)
	if notice.Kind != flash.KindSuccess || notice.Key != "web.notice.saved" {
		t.Fatalf("flash notice = %#v", notice)
	}
}

func TestRouteParamsAndWrappersUseCanonicalIDs(t *testing.T) {
	t.Parallel()

	support := NewSupport(modulehandler.NewBase(nil, nil, nil), requestmeta.SchemePolicy{}, nil)

	t.Run("campaign and character", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("campaignID", " camp-1 ")
		req.SetPathValue("characterID", " char-1 ")
		if got, ok := support.RouteCampaignID(req); !ok || got != "camp-1" {
			t.Fatalf("RouteCampaignID() = %q, %v", got, ok)
		}
		if got, ok := support.RouteCharacterID(req); !ok || got != "char-1" {
			t.Fatalf("RouteCharacterID() = %q, %v", got, ok)
		}

		rr := httptest.NewRecorder()
		var gotCampaign, gotCharacter string
		support.WithCampaignAndCharacterID(func(w http.ResponseWriter, _ *http.Request, campaignID, characterID string) {
			gotCampaign, gotCharacter = campaignID, characterID
			w.WriteHeader(http.StatusNoContent)
		})(rr, req)
		if rr.Code != http.StatusNoContent || gotCampaign != "camp-1" || gotCharacter != "char-1" {
			t.Fatalf("dispatch = code:%d campaign:%q character:%q", rr.Code, gotCampaign, gotCharacter)
		}
	})

	t.Run("participant and session", func(t *testing.T) {
		t.Parallel()

		participantReq := httptest.NewRequest(http.MethodGet, "/", nil)
		participantReq.SetPathValue("campaignID", "camp-1")
		participantReq.SetPathValue("participantID", "part-1")
		if got, ok := support.RouteParticipantID(participantReq); !ok || got != "part-1" {
			t.Fatalf("RouteParticipantID() = %q, %v", got, ok)
		}
		participantRR := httptest.NewRecorder()
		var gotParticipant string
		support.WithCampaignAndParticipantID(func(w http.ResponseWriter, _ *http.Request, _, participantID string) {
			gotParticipant = participantID
			w.WriteHeader(http.StatusNoContent)
		})(participantRR, participantReq)
		if participantRR.Code != http.StatusNoContent || gotParticipant != "part-1" {
			t.Fatalf("participant dispatch = code:%d participant:%q", participantRR.Code, gotParticipant)
		}

		sessionReq := httptest.NewRequest(http.MethodGet, "/", nil)
		sessionReq.SetPathValue("campaignID", "camp-1")
		sessionReq.SetPathValue("sessionID", "sess-1")
		if got, ok := support.RouteSessionID(sessionReq); !ok || got != "sess-1" {
			t.Fatalf("RouteSessionID() = %q, %v", got, ok)
		}
		sessionRR := httptest.NewRecorder()
		var gotSession string
		support.WithCampaignAndSessionID(func(w http.ResponseWriter, _ *http.Request, _, sessionID string) {
			gotSession = sessionID
			w.WriteHeader(http.StatusNoContent)
		})(sessionRR, sessionReq)
		if sessionRR.Code != http.StatusNoContent || gotSession != "sess-1" {
			t.Fatalf("session dispatch = code:%d session:%q", sessionRR.Code, gotSession)
		}
	})

	t.Run("starter and missing campaign", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("starterKey", " starter-1 ")
		if got, ok := support.RouteStarterKey(req); !ok || got != "starter-1" {
			t.Fatalf("RouteStarterKey() = %q, %v", got, ok)
		}

		rr := httptest.NewRecorder()
		support.WithCampaignID(func(_ http.ResponseWriter, _ *http.Request, _ string) {
			t.Fatal("WithCampaignID unexpectedly dispatched without a campaign ID")
		})(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})
}

func flashNoticeFromRecorder(t *testing.T, rr *httptest.ResponseRecorder) flash.Notice {
	t.Helper()

	res := rr.Result()
	for _, cookie := range res.Cookies() {
		if cookie.Name != flash.CookieName {
			continue
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(cookie)
		notice, ok := flash.ReadAndClear(nil, req)
		if !ok {
			t.Fatal("flash.ReadAndClear() failed to decode notice")
		}
		return notice
	}
	t.Fatal("missing flash cookie")
	return flash.Notice{}
}
