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
