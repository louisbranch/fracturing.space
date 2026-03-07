package game

import (
	"context"
	"fmt"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestCalculateDepth_LinearChain(t *testing.T) {
	store := newFakeCampaignForkStore()
	store.metadata["child"] = storage.ForkMetadata{ParentCampaignID: "parent"}
	store.metadata["parent"] = storage.ForkMetadata{ParentCampaignID: "origin"}
	store.metadata["origin"] = storage.ForkMetadata{}

	depth := calculateDepth(context.Background(), store, "child")
	if depth != 2 {
		t.Fatalf("calculateDepth(child) = %d, want 2", depth)
	}
}

func TestCalculateDepth_StopsWhenMetadataMissing(t *testing.T) {
	store := newFakeCampaignForkStore()
	store.metadata["child"] = storage.ForkMetadata{ParentCampaignID: "missing-parent"}

	depth := calculateDepth(context.Background(), store, "child")
	if depth != 1 {
		t.Fatalf("calculateDepth(child) = %d, want 1", depth)
	}
}

func TestCalculateDepth_CapsAtLoopGuard(t *testing.T) {
	store := newFakeCampaignForkStore()
	for i := 0; i < 150; i++ {
		currentID := fmt.Sprintf("camp-%d", i)
		parentID := fmt.Sprintf("camp-%d", i+1)
		store.metadata[currentID] = storage.ForkMetadata{ParentCampaignID: parentID}
	}

	depth := calculateDepth(context.Background(), store, "camp-0")
	if depth != 100 {
		t.Fatalf("calculateDepth(camp-0) = %d, want 100 (loop guard)", depth)
	}
}
