package damagetransport

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerRequireDamageDependencies(t *testing.T) {
	tests := []struct {
		name string
		deps Dependencies
	}{
		{name: "missing campaign", deps: Dependencies{}},
		{name: "missing gate", deps: Dependencies{Campaign: testCampaignStore{}}},
		{name: "missing daggerheart", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}}},
		{name: "missing event", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}}},
		{name: "missing executor", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}, Event: testEventStore{}}},
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
		Event:                testEventStore{},
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error { return nil },
	}

	if err := NewHandler(deps).requireAdversaryDamageDependencies(); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
