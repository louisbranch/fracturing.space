package gmmovetransport

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmconsequence"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerRequireDependencies(t *testing.T) {
	noDomain := func(context.Context, DomainCommandInput) error { return nil }
	noCore := func(context.Context, gmconsequence.CoreCommandInput) error { return nil }

	tests := []struct {
		name string
		deps Dependencies
	}{
		{name: "missing campaign", deps: Dependencies{}},
		{name: "missing session", deps: Dependencies{Campaign: testCampaignStore{}}},
		{name: "missing gate", deps: Dependencies{Campaign: testCampaignStore{}, Session: testSessionStore{}}},
		{name: "missing spotlight", deps: Dependencies{Campaign: testCampaignStore{}, Session: testSessionStore{}, SessionGate: testGateStore{}}},
		{name: "missing daggerheart", deps: Dependencies{Campaign: testCampaignStore{}, Session: testSessionStore{}, SessionGate: testGateStore{}, SessionSpotlight: testSpotlightStore{}}},
		{name: "missing content", deps: Dependencies{Campaign: testCampaignStore{}, Session: testSessionStore{}, SessionGate: testGateStore{}, SessionSpotlight: testSpotlightStore{}, Daggerheart: testDaggerheartStore{}}},
		{name: "missing domain executor", deps: Dependencies{Campaign: testCampaignStore{}, Session: testSessionStore{}, SessionGate: testGateStore{}, SessionSpotlight: testSpotlightStore{}, Daggerheart: testDaggerheartStore{}, Content: testContentStore{}}},
		{name: "missing core executor", deps: Dependencies{Campaign: testCampaignStore{}, Session: testSessionStore{}, SessionGate: testGateStore{}, SessionSpotlight: testSpotlightStore{}, Daggerheart: testDaggerheartStore{}, Content: testContentStore{}, ExecuteDomainCommand: noDomain}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewHandler(tt.deps).requireDependencies(); status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
		})
	}

	// Invariant: all deps present must pass.
	allDeps := Dependencies{
		Campaign:             testCampaignStore{},
		Session:              testSessionStore{},
		SessionGate:          testGateStore{},
		SessionSpotlight:     testSpotlightStore{},
		Daggerheart:          testDaggerheartStore{},
		Content:              testContentStore{},
		ExecuteDomainCommand: noDomain,
		ExecuteCoreCommand:   noCore,
	}
	if err := NewHandler(allDeps).requireDependencies(); err != nil {
		t.Fatalf("requireDependencies with all deps returned error: %v", err)
	}
}
