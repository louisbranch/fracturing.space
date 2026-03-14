package invite

import (
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/message"
)

// Localizer keeps invite-owned templates aligned with the shared translation seam.
type Localizer = webtemplates.Localizer

// T keeps invite-owned template translation lookups on the shared helper.
func T(loc Localizer, key message.Reference, args ...any) string {
	return webtemplates.T(loc, key, args...)
}
