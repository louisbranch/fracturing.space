package campaigns

import (
	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/message"
)

// Localizer keeps campaigns-owned templates aligned with the shared translation seam.
type Localizer = webtemplates.Localizer

// AppImageView reuses the shared responsive image contract from web templates.
type AppImageView = webtemplates.AppImageView

// T keeps campaigns-owned template translation lookups on the shared helper.
func T(loc Localizer, key message.Reference, args ...any) string {
	return webtemplates.T(loc, key, args...)
}

// AppImage keeps campaigns-owned templates on the shared image helper.
func AppImage(view AppImageView) templ.Component {
	return webtemplates.AppImage(view)
}

// AppThemeName keeps full-page module templates on the shared app shell theme.
const AppThemeName = webtemplates.AppThemeName

// AppLayoutLang keeps module-owned full-page templates on the shared lang fallback rules.
func AppLayoutLang(lang string) string {
	return webtemplates.AppLayoutLang(lang)
}

// DaisyThemeHead reuses the shared app-shell head includes for module-owned full-page templates.
func DaisyThemeHead() templ.Component {
	return webtemplates.DaisyThemeHead()
}

// HTMXHead reuses the shared htmx include for module-owned full-page templates.
func HTMXHead() templ.Component {
	return webtemplates.HTMXHead()
}
