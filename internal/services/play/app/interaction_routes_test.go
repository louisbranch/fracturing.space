package app

import "testing"

func TestInteractionRoutesExposeExpectedMutationSurface(t *testing.T) {
	t.Parallel()

	server := &Server{deps: Dependencies{Interaction: stubInteractionClient{}}}
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
		t.Fatalf("interactionRoutes() count = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("interactionRoutes()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}
