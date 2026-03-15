package daggerheart

import "testing"

func TestContentAndAssetStoreHelpers(t *testing.T) {
	contentStore := newFakeContentStore()

	contentService := &DaggerheartContentService{store: contentStore}
	if _, err := contentService.contentStore(); err != nil {
		t.Fatalf("contentStore configured: %v", err)
	}

	assetService := &DaggerheartAssetService{store: contentStore}
	if _, err := assetService.assetStore(); err != nil {
		t.Fatalf("assetStore configured: %v", err)
	}
}

func TestNewDaggerheartContentService(t *testing.T) {
	cs := newFakeContentStore()
	svc, err := NewDaggerheartContentService(cs)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewDaggerheartContentServiceRejectsMissingStore(t *testing.T) {
	svc, err := NewDaggerheartContentService(nil)
	if err == nil {
		t.Fatal("expected constructor error for missing content store")
	}
	if svc != nil {
		t.Fatal("expected nil service on constructor error")
	}
}
