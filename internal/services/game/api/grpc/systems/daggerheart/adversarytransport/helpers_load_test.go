package adversarytransport

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLoadAdversaryForSession(t *testing.T) {
	store := &testDaggerheartStore{
		adversaries: map[string]projectionstore.DaggerheartAdversary{
			"adv-1": {CampaignID: "camp-1", AdversaryID: "adv-1", SessionID: "sess-1"},
		},
	}
	adversary, err := LoadAdversaryForSession(context.Background(), store, "camp-1", "sess-1", "adv-1")
	if err != nil {
		t.Fatalf("LoadAdversaryForSession returned error: %v", err)
	}
	if adversary.AdversaryID != "adv-1" {
		t.Fatalf("adversary id = %q, want adv-1", adversary.AdversaryID)
	}
	if _, err := LoadAdversaryForSession(context.Background(), store, "camp-1", "other", "adv-1"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
	store.err = storage.ErrNotFound
	if _, err := LoadAdversaryForSession(context.Background(), store, "camp-1", "sess-1", "missing"); status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.NotFound)
	}
}
