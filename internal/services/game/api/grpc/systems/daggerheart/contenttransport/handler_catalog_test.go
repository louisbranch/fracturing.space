package contenttransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestHandlerCatalogEndpoints(t *testing.T) {
	ctx := context.Background()
	handler := NewHandler(newFakeContentStore())

	t.Run("GetContentCatalog", func(t *testing.T) {
		resp, err := handler.GetContentCatalog(ctx, &pb.GetDaggerheartContentCatalogRequest{})
		if err != nil {
			t.Fatalf("GetContentCatalog: %v", err)
		}
		if len(resp.GetCatalog().GetClasses()) != 1 || len(resp.GetCatalog().GetEnvironments()) != 1 {
			t.Fatalf("catalog counts mismatch: %+v", resp.GetCatalog())
		}
	})

	t.Run("GetAssetMap", func(t *testing.T) {
		resp, err := handler.GetAssetMap(ctx, &pb.GetDaggerheartAssetMapRequest{})
		if err != nil {
			t.Fatalf("GetAssetMap: %v", err)
		}
		if len(resp.GetAssetMap().GetAssets()) == 0 {
			t.Fatal("expected non-empty asset map")
		}
	})
}
