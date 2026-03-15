package charactermutationtransport

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerRequireDependencies(t *testing.T) {
	tests := []struct {
		name string
		deps Dependencies
	}{
		{name: "missing campaign", deps: Dependencies{}},
		{name: "missing daggerheart", deps: Dependencies{Campaign: testCampaignStore{}}},
		{name: "missing executor", deps: Dependencies{Campaign: testCampaignStore{}, Daggerheart: &testDaggerheartStore{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewHandler(tt.deps).requireDependencies(); status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
		})
	}
}
