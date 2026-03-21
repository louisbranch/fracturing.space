package contenttransport

import (
	"context"
	"errors"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

type failingCatalogStore struct {
	*fakeContentStore
	listDomainsErr        error
	listContentStringsErr error
}

func (s *failingCatalogStore) ListDaggerheartDomains(ctx context.Context) ([]contentstore.DaggerheartDomain, error) {
	if s.listDomainsErr != nil {
		return nil, s.listDomainsErr
	}
	return s.fakeContentStore.ListDaggerheartDomains(ctx)
}

func (s *failingCatalogStore) ListDaggerheartContentStrings(ctx context.Context, contentType string, contentIDs []string, locale string) ([]contentstore.DaggerheartContentString, error) {
	if s.listContentStringsErr != nil {
		return nil, s.listContentStringsErr
	}
	return s.fakeContentStore.ListDaggerheartContentStrings(ctx, contentType, contentIDs, locale)
}

func TestContentCatalogRunLocalizesReferenceAndCompendiumContent(t *testing.T) {
	store := newFakeContentStore()
	store.contentStrings = []contentstore.DaggerheartContentString{
		{ContentID: "class-1", ContentType: contentTypeClass, Field: "name", Locale: "pt-BR", Text: "Guardiao"},
		{ContentID: "dom-1", ContentType: contentTypeDomain, Field: "name", Locale: "pt-BR", Text: "Valor"},
	}

	catalog := newContentCatalog(store, commonv1.Locale_LOCALE_PT_BR)
	if err := catalog.run(context.Background()); err != nil {
		t.Fatalf("catalog.run() error = %v", err)
	}

	if got := catalog.classes[0].Name; got != "Guardiao" {
		t.Fatalf("catalog.classes[0].Name = %q, want %q", got, "Guardiao")
	}
	if got := catalog.domains[0].Name; got != "Valor" {
		t.Fatalf("catalog.domains[0].Name = %q, want %q", got, "Valor")
	}

	proto := catalog.proto()
	if got := proto.GetClasses()[0].GetName(); got != "Guardiao" {
		t.Fatalf("proto class name = %q, want %q", got, "Guardiao")
	}
	if got := proto.GetDomains()[0].GetName(); got != "Valor" {
		t.Fatalf("proto domain name = %q, want %q", got, "Valor")
	}
}

func TestContentCatalogRunWrapsFailingStepName(t *testing.T) {
	store := &failingCatalogStore{
		fakeContentStore: newFakeContentStore(),
		listDomainsErr:   errors.New("boom"),
	}

	catalog := newContentCatalog(store, commonv1.Locale_LOCALE_UNSPECIFIED)
	err := catalog.run(context.Background())
	if err == nil {
		t.Fatal("catalog.run() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "list domains") {
		t.Fatalf("catalog.run() error = %q, want step name %q", err, "list domains")
	}
}

func TestContentCatalogStepsPreserveLoadThenLocalizeOrder(t *testing.T) {
	steps := newContentCatalog(newFakeContentStore(), commonv1.Locale_LOCALE_UNSPECIFIED).steps()
	if len(steps) != 30 {
		t.Fatalf("len(steps) = %d, want %d", len(steps), 30)
	}
	if steps[0].name != "list classes" {
		t.Fatalf("steps[0] = %q, want %q", steps[0].name, "list classes")
	}
	if steps[14].name != "list environments" {
		t.Fatalf("steps[14] = %q, want %q", steps[14].name, "list environments")
	}
	if steps[15].name != "localize classes" {
		t.Fatalf("steps[15] = %q, want %q", steps[15].name, "localize classes")
	}
	if steps[len(steps)-1].name != "localize environments" {
		t.Fatalf("last step = %q, want %q", steps[len(steps)-1].name, "localize environments")
	}
}
