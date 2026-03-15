package contenttransport

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func TestLocaleString(t *testing.T) {
	if got, ok := localeString(commonv1.Locale_LOCALE_UNSPECIFIED); got != "" || ok {
		t.Fatalf("localeString(unspecified) = (%q, %v), want empty false", got, ok)
	}

	got, ok := localeString(commonv1.Locale_LOCALE_EN_US)
	if !ok || got != "en-US" {
		t.Fatalf("localeString(en_us) = (%q, %v), want %q true", got, ok, "en-US")
	}
}

func TestFetchContentStrings(t *testing.T) {
	store := newFakeContentStore()
	store.contentStrings = []contentstore.DaggerheartContentString{
		{ContentID: "class-1", ContentType: contentTypeClass, Field: "name", Locale: "fr-CA", Text: "Gardien"},
		{ContentID: "class-1", ContentType: contentTypeClass, Field: "name", Locale: "en-US", Text: "Guardian"},
		{ContentID: "class-2", ContentType: contentTypeClass, Field: "name", Locale: "fr-CA", Text: "Sentinelle"},
	}

	lookup, err := fetchContentStrings(context.Background(), store, contentTypeClass, []string{"class-1"}, "fr-CA")
	if err != nil {
		t.Fatalf("fetchContentStrings error: %v", err)
	}

	if got := lookup[contentStringKey{ContentID: "class-1", Field: "name"}]; got != "Gardien" {
		t.Fatalf("lookup name = %q, want %q", got, "Gardien")
	}
	if _, ok := lookup[contentStringKey{ContentID: "class-2", Field: "name"}]; ok {
		t.Fatal("unexpected lookup entry for filtered content id")
	}
}

func TestApplyLocalizedStringNilTarget(t *testing.T) {
	applyLocalizedString(
		contentStringLookup{{ContentID: "class-1", Field: "name"}: "Gardien"},
		"class-1",
		"name",
		nil,
	)
}
