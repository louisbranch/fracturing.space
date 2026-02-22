package icons

import (
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestCatalogContainsAllEnumValues(t *testing.T) {
	defs := Catalog()
	if len(defs) == 0 {
		t.Fatal("expected catalog to include icon definitions")
	}

	seen := make(map[commonv1.IconId]struct{})
	for _, def := range defs {
		if def.ID == commonv1.IconId_ICON_ID_UNSPECIFIED {
			t.Errorf("unexpected unspecified icon id in catalog")
		}
		if _, ok := seen[def.ID]; ok {
			t.Errorf("duplicate icon id in catalog: %s", def.ID.String())
		}
		seen[def.ID] = struct{}{}
		if strings.TrimSpace(def.Name) == "" {
			t.Errorf("icon %s missing name", def.ID.String())
		}
	}

	for rawID, name := range commonv1.IconId_name {
		id := commonv1.IconId(rawID)
		if id == commonv1.IconId_ICON_ID_UNSPECIFIED {
			continue
		}
		if _, ok := seen[id]; !ok {
			t.Errorf("icon id missing from catalog: %s", name)
		}
	}
}

func TestCatalogMarkdownIncludesIconIds(t *testing.T) {
	markdown := CatalogMarkdown()
	if strings.TrimSpace(markdown) == "" {
		t.Fatal("expected catalog markdown to be non-empty")
	}

	for rawID, name := range commonv1.IconId_name {
		id := commonv1.IconId(rawID)
		if id == commonv1.IconId_ICON_ID_UNSPECIFIED {
			continue
		}
		if !strings.Contains(markdown, name) {
			t.Errorf("catalog markdown missing icon id %s", name)
		}
	}
}

func TestLucideMappingsAreCataloged(t *testing.T) {
	catalogIDs := make(map[commonv1.IconId]struct{}, len(Catalog()))
	for _, def := range Catalog() {
		catalogIDs[def.ID] = struct{}{}
	}

	for id, name := range lucideIconNames {
		if _, ok := catalogIDs[id]; !ok {
			t.Errorf("lucide mapping for %s exists but icon id is missing from catalog", name)
		}
	}
}

func TestCatalogIconsHaveLucideMappings(t *testing.T) {
	for _, def := range Catalog() {
		if _, ok := LucideName(def.ID); !ok {
			t.Errorf("catalog icon %s does not have a Lucide mapping", def.ID.String())
		}
	}
}
