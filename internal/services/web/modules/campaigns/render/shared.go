package render

import (
	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/message"
)

// Localizer keeps the render package aligned with the shared translation seam.
type Localizer = webtemplates.Localizer

// AppImageView reuses the shared responsive image contract from web templates.
type AppImageView = webtemplates.AppImageView

// DateTimeDisplay reuses the shared timestamp formatting contract.
type DateTimeDisplay = webtemplates.DateTimeDisplay

// T keeps translation lookups in module-owned templates on the canonical helper.
func T(loc Localizer, key message.Reference, args ...any) string {
	return webtemplates.T(loc, key, args...)
}

// AppImage reuses the shared image component without moving image ownership.
func AppImage(view AppImageView) templ.Component {
	return webtemplates.AppImage(view)
}

// DateTimeTooltip reuses the shared timestamp tooltip component from web templates.
func DateTimeTooltip(display DateTimeDisplay) templ.Component {
	return webtemplates.DateTimeTooltip(display)
}

// FormatDateTimeNow keeps campaign detail timestamps on the shared formatting rules.
func FormatDateTimeNow(timestamp string, loc Localizer) DateTimeDisplay {
	return webtemplates.FormatDateTimeNow(timestamp, loc)
}

// characterCreationSummaryBody reuses the shared character-creation summary panel.
func characterCreationSummaryBody(creation webtemplates.CampaignCharacterCreationView, loc Localizer) templ.Component {
	return webtemplates.CampaignCharacterCreationSummaryBody(creation, loc)
}
