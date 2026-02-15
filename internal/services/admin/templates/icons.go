package templates

import commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"

// IconRow holds formatted icon catalog data for display.
type IconRow struct {
	ID          commonv1.IconId
	Name        string
	Description string
	LucideName  string
}
