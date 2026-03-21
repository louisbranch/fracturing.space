package app

import "testing"

func TestInteractionRoutesExposeExpectedMutationSurface(t *testing.T) {
	t.Parallel()

	server := &Server{interaction: fakePlayInteractionClient{}}
	routes := interactionRoutes(server)
	got := make([]string, 0, len(routes))
	seen := map[string]struct{}{}
	for _, route := range routes {
		if route.pattern == "" {
			t.Fatal("interaction route pattern is empty")
		}
		if route.handler == nil {
			t.Fatalf("interaction route %q has nil handler", route.pattern)
		}
		if _, ok := seen[route.pattern]; ok {
			t.Fatalf("duplicate interaction route pattern %q", route.pattern)
		}
		seen[route.pattern] = struct{}{}
		got = append(got, route.pattern)
	}

	want := []string{
		"POST /api/campaigns/{campaignID}/interaction/set-active-scene",
		"POST /api/campaigns/{campaignID}/interaction/start-scene-player-phase",
		"POST /api/campaigns/{campaignID}/interaction/submit-scene-player-post",
		"POST /api/campaigns/{campaignID}/interaction/yield-scene-player-phase",
		"POST /api/campaigns/{campaignID}/interaction/unyield-scene-player-phase",
		"POST /api/campaigns/{campaignID}/interaction/end-scene-player-phase",
		"POST /api/campaigns/{campaignID}/interaction/commit-scene-gm-interaction",
		"POST /api/campaigns/{campaignID}/interaction/resolve-scene-player-phase-review",
		"POST /api/campaigns/{campaignID}/interaction/pause-session-for-ooc",
		"POST /api/campaigns/{campaignID}/interaction/post-session-ooc",
		"POST /api/campaigns/{campaignID}/interaction/mark-ooc-ready-to-resume",
		"POST /api/campaigns/{campaignID}/interaction/clear-ooc-ready-to-resume",
		"POST /api/campaigns/{campaignID}/interaction/resume-from-ooc",
		"POST /api/campaigns/{campaignID}/interaction/resolve-interrupted-scene-phase",
		"POST /api/campaigns/{campaignID}/interaction/set-session-gm-authority",
		"POST /api/campaigns/{campaignID}/interaction/retry-ai-gm-turn",
	}

	if len(got) != len(want) {
		t.Fatalf("interactionRoutes() count = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("interactionRoutes()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}
