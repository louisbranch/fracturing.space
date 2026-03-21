package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlayRoutesExposeExpectedSurface(t *testing.T) {
	t.Parallel()

	server := &Server{interaction: fakePlayInteractionClient{}}
	server.realtime = newRealtimeHub(server)

	routes := server.playRoutes(testPlayLaunchGrantConfig(t))
	got := make([]string, 0, len(routes))
	seen := map[string]struct{}{}
	for _, route := range routes {
		if route.pattern == "" {
			t.Fatal("play route pattern is empty")
		}
		if route.handler == nil {
			t.Fatalf("play route %q has nil handler", route.pattern)
		}
		if _, ok := seen[route.pattern]; ok {
			t.Fatalf("duplicate play route pattern %q", route.pattern)
		}
		seen[route.pattern] = struct{}{}
		got = append(got, route.pattern)
	}

	want := []string{
		"GET /up",
		"GET /{$}",
		"GET /campaigns/{campaignID}",
		"GET /api/campaigns/{campaignID}/bootstrap",
		"GET /api/campaigns/{campaignID}/chat/history",
		"GET /realtime",
		"POST /api/campaigns/{campaignID}/interaction/activate-scene",
		"POST /api/campaigns/{campaignID}/interaction/open-scene-player-phase",
		"POST /api/campaigns/{campaignID}/interaction/submit-scene-player-action",
		"POST /api/campaigns/{campaignID}/interaction/yield-scene-player-phase",
		"POST /api/campaigns/{campaignID}/interaction/withdraw-scene-player-yield",
		"POST /api/campaigns/{campaignID}/interaction/interrupt-scene-player-phase",
		"POST /api/campaigns/{campaignID}/interaction/record-scene-gm-interaction",
		"POST /api/campaigns/{campaignID}/interaction/resolve-scene-player-review",
		"POST /api/campaigns/{campaignID}/interaction/open-session-ooc",
		"POST /api/campaigns/{campaignID}/interaction/post-session-ooc",
		"POST /api/campaigns/{campaignID}/interaction/mark-ooc-ready-to-resume",
		"POST /api/campaigns/{campaignID}/interaction/clear-ooc-ready-to-resume",
		"POST /api/campaigns/{campaignID}/interaction/resolve-session-ooc",
		"POST /api/campaigns/{campaignID}/interaction/set-session-gm-authority",
		"POST /api/campaigns/{campaignID}/interaction/retry-ai-gm-turn",
	}

	if len(got) != len(want) {
		t.Fatalf("playRoutes() count = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("playRoutes()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestPlayRoutesDoNotServeRetiredPreviewPath(t *testing.T) {
	t.Parallel()

	server := &Server{interaction: fakePlayInteractionClient{}}
	server.realtime = newRealtimeHub(server)

	mux := http.NewServeMux()
	server.registerRoutes(mux, testPlayLaunchGrantConfig(t))

	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/preview/character-card", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	// Invariant: the isolated component catalog now lives only in Storybook, so
	// the play service must fail fast for the retired preview path.
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}
