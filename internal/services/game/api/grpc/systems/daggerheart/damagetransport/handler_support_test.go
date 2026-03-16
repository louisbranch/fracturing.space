package damagetransport

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerRequireDamageDependencies(t *testing.T) {
	emptyContent := testContentStore{
		adversaryEntries: make(map[string]contentstore.DaggerheartAdversaryEntry),
		armors:           make(map[string]contentstore.DaggerheartArmor),
	}
	tests := []struct {
		name string
		deps Dependencies
	}{
		{name: "missing campaign", deps: Dependencies{}},
		{name: "missing gate", deps: Dependencies{Campaign: testCampaignStore{}}},
		{name: "missing daggerheart", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}}},
		{name: "missing content", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}}},
		{name: "missing event", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}, Content: emptyContent}},
		{name: "missing executor", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}, Content: emptyContent, Event: testEventStore{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewHandler(tt.deps).requireDamageDependencies(); status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
		})
	}
}

func TestHandlerRequireAdversaryDamageDependencies(t *testing.T) {
	deps := Dependencies{
		Campaign:             testCampaignStore{},
		SessionGate:          testSessionGateStore{},
		Daggerheart:          testDaggerheartStore{},
		Content:              testContentStore{adversaryEntries: make(map[string]contentstore.DaggerheartAdversaryEntry), armors: make(map[string]contentstore.DaggerheartArmor)},
		Event:                testEventStore{},
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error { return nil },
	}

	if err := NewHandler(deps).requireAdversaryDamageDependencies(); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
