package icons

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestLucideNameCoversIconIds(t *testing.T) {
	for rawID, name := range commonv1.IconId_name {
		id := commonv1.IconId(rawID)
		if id == commonv1.IconId_ICON_ID_UNSPECIFIED {
			continue
		}
		if _, ok := LucideName(id); !ok {
			t.Fatalf("missing Lucide mapping for %s", name)
		}
	}
}
