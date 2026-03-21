package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	gogrpccodes "google.golang.org/grpc/codes"
	gogrpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestDecodeStrictJSONVariants(t *testing.T) {
	t.Parallel()

	t.Run("nil target is a no-op", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ignored":true}`))
		if err := decodeStrictJSON(req, nil); err != nil {
			t.Fatalf("decodeStrictJSON(nil target) error = %v", err)
		}
	})

	t.Run("empty body leaves target untouched", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
		payload := struct {
			Name string `json:"name"`
		}{Name: "unchanged"}
		if err := decodeStrictJSON(req, &payload); err != nil {
			t.Fatalf("decodeStrictJSON(empty body) error = %v", err)
		}
		if payload.Name != "unchanged" {
			t.Fatalf("payload = %#v", payload)
		}
	})

	t.Run("valid body decodes", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Avery"}`))
		payload := struct {
			Name string `json:"name"`
		}{}
		if err := decodeStrictJSON(req, &payload); err != nil {
			t.Fatalf("decodeStrictJSON(valid) error = %v", err)
		}
		if payload.Name != "Avery" {
			t.Fatalf("payload = %#v", payload)
		}
	})

	t.Run("unknown field is rejected", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"unknown":true}`))
		payload := struct {
			Name string `json:"name"`
		}{}
		if err := decodeStrictJSON(req, &payload); err == nil {
			t.Fatal("decodeStrictJSON(unknown field) error = nil, want non-nil")
		}
	})

	t.Run("multiple json values are rejected", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Avery"}{"name":"Bryn"}`))
		payload := struct {
			Name string `json:"name"`
		}{}
		if err := decodeStrictJSON(req, &payload); err == nil {
			t.Fatal("decodeStrictJSON(multiple values) error = nil, want non-nil")
		}
	})

	t.Run("oversized body is rejected", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", maxJSONBodyBytes+1)))
		payload := struct{}{}
		if err := decodeStrictJSON(req, &payload); err != io.ErrUnexpectedEOF {
			t.Fatalf("decodeStrictJSON(oversized) error = %v, want %v", err, io.ErrUnexpectedEOF)
		}
	})
}

func TestWriteJSONHelpers(t *testing.T) {
	t.Parallel()

	t.Run("write json", func(t *testing.T) {
		t.Parallel()
		rr := httptest.NewRecorder()
		writeJSON(rr, http.StatusAccepted, map[string]string{"status": "ok"})
		if rr.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("Content-Type = %q", got)
		}
		var payload map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["status"] != "ok" {
			t.Fatalf("payload = %#v", payload)
		}
	})

	t.Run("write json error", func(t *testing.T) {
		t.Parallel()
		rr := httptest.NewRecorder()
		writeJSONError(rr, http.StatusBadRequest, "invalid")
		assertJSONError(t, rr, http.StatusBadRequest, "invalid")
	})
}

func TestProtoJSONHelpers(t *testing.T) {
	t.Parallel()

	raw, err := marshalProtoJSON(nil)
	if err != nil {
		t.Fatalf("marshalProtoJSON(nil) error = %v", err)
	}
	if string(raw) != "{}" {
		t.Fatalf("marshalProtoJSON(nil) = %q, want {}", string(raw))
	}

	rr := httptest.NewRecorder()
	if err := writeProtoJSON(rr, http.StatusCreated, &emptypb.Empty{}); err != nil {
		t.Fatalf("writeProtoJSON() error = %v", err)
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if strings.TrimSpace(rr.Body.String()) != "{}" {
		t.Fatalf("body = %q, want {}", rr.Body.String())
	}
}

func TestPlaySessionCookieHelpers(t *testing.T) {
	t.Parallel()

	if value, ok := readPlaySessionCookie(nil); ok || value != "" {
		t.Fatalf("readPlaySessionCookie(nil) = (%q, %v)", value, ok)
	}

	req := httptest.NewRequest(http.MethodGet, "https://play.example.com/campaigns/c1", nil)
	req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: " ps-1 "})
	if value, ok := readPlaySessionCookie(req); !ok || value != "ps-1" {
		t.Fatalf("readPlaySessionCookie() = (%q, %v)", value, ok)
	}

	rr := httptest.NewRecorder()
	writePlaySessionCookie(rr, req, " ps-2 ", requestmeta.SchemePolicy{})
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value != "ps-2" || !cookies[0].Secure {
		t.Fatalf("writePlaySessionCookie() cookies = %#v", cookies)
	}

	rr = httptest.NewRecorder()
	clearPlaySessionCookie(rr, req, requestmeta.SchemePolicy{})
	cookies = rr.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != -1 {
		t.Fatalf("clearPlaySessionCookie() cookies = %#v", cookies)
	}
}

func TestShellAssetsHelpers(t *testing.T) {
	t.Parallel()

	t.Run("dev server url is normalized", func(t *testing.T) {
		t.Parallel()
		assets, err := loadShellAssets("http://localhost:5173/")
		if err != nil {
			t.Fatalf("loadShellAssets(dev server) error = %v", err)
		}
		if assets.devServerURL != "http://localhost:5173" {
			t.Fatalf("devServerURL = %q", assets.devServerURL)
		}
	})

	t.Run("embedded dist manifest resolves asset paths", func(t *testing.T) {
		t.Parallel()
		assets, err := loadShellAssets("")
		if err != nil {
			t.Fatalf("loadShellAssets(dist) error = %v", err)
		}
		if !strings.HasPrefix(assets.entryJS, "/assets/play/") {
			t.Fatalf("entryJS = %q", assets.entryJS)
		}
		if !strings.HasPrefix(assets.entryCSS, "/assets/play/") {
			t.Fatalf("entryCSS = %q", assets.entryCSS)
		}
		html, err := assets.renderHTML(shellRenderInput{
			CampaignID:    "c1",
			BootstrapPath: "/api/campaigns/c1/bootstrap",
			RealtimePath:  "/realtime",
			BackURL:       "/app/campaigns/c1/game",
		})
		if err != nil {
			t.Fatalf("renderHTML() error = %v", err)
		}
		body := string(html)
		for _, want := range []string{
			`<div id="root"></div>`,
			`type="module" src="`,
			`/api/campaigns/c1/bootstrap`,
			`/realtime`,
			`/app/campaigns/c1/game`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("renderHTML() missing %q in %q", want, body)
			}
		}
	})
}

func TestStripLaunchGrantRemovesOnlyLaunchQuery(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/campaigns/c1?launch=abc&foo=bar", nil)
	if got := stripLaunchGrant(req); got != "http://play.example.com/campaigns/c1?foo=bar" {
		t.Fatalf("stripLaunchGrant() = %q", got)
	}
}

func TestWriteRPCErrorMapsGRPCCodesToHTTPStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err    error
		status int
	}{
		{err: gogrpcstatus.Error(gogrpccodes.InvalidArgument, "bad"), status: http.StatusBadRequest},
		{err: gogrpcstatus.Error(gogrpccodes.PermissionDenied, "bad"), status: http.StatusForbidden},
		{err: gogrpcstatus.Error(gogrpccodes.NotFound, "bad"), status: http.StatusNotFound},
		{err: gogrpcstatus.Error(gogrpccodes.FailedPrecondition, "bad"), status: http.StatusConflict},
		{err: gogrpcstatus.Error(gogrpccodes.Unauthenticated, "bad"), status: http.StatusUnauthorized},
		{err: errors.New("boom"), status: http.StatusBadGateway},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		writeRPCError(rr, tc.err)
		if rr.Code != tc.status {
			t.Fatalf("status = %d, want %d for %v", rr.Code, tc.status, tc.err)
		}
	}
}

func TestMiscHelpers(t *testing.T) {
	t.Parallel()

	if value, err := parseInt64(" 42 "); err != nil || value != 42 {
		t.Fatalf("parseInt64() = %d, %v", value, err)
	}
	if value, err := parseInt(" 7 "); err != nil || value != 7 {
		t.Fatalf("parseInt() = %d, %v", value, err)
	}
	if got := gameSystemIDString(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART); got != "daggerheart" {
		t.Fatalf("gameSystemIDString() = %q", got)
	}
	if got := pathForCampaignAPI("c1", "chat/history"); got != "/api/campaigns/c1/chat/history" {
		t.Fatalf("pathForCampaignAPI() = %q", got)
	}
	if loggerOrDefault(nil) == nil {
		t.Fatal("loggerOrDefault(nil) returned nil")
	}
}
