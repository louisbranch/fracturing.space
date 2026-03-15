package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestGetContentCatalog_WithTypes(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetContentCatalog(context.Background(), &pb.GetDaggerheartContentCatalogRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	catalog := resp.GetCatalog()
	if catalog == nil {
		t.Fatal("expected non-nil catalog")
	}
	if len(catalog.GetClasses()) == 0 {
		t.Error("expected non-empty classes in catalog")
	}
	if len(catalog.GetWeapons()) == 0 {
		t.Error("expected non-empty weapons in catalog")
	}
}
