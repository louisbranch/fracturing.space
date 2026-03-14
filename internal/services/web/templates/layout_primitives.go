package templates

import "github.com/a-h/templ"

// AppThemeName exposes the canonical shell theme for module-owned full-page templates.
const AppThemeName = daisyTheme

// AppLayoutLang keeps module-owned full-page templates aligned with layout language fallback rules.
func AppLayoutLang(lang string) string {
	return appLayoutLang(lang)
}

// DaisyThemeHead exposes the shared stylesheet/script shell dependencies for module-owned full-page templates.
func DaisyThemeHead() templ.Component {
	return daisyThemeHead()
}

// HTMXHead exposes the shared htmx script include for module-owned full-page templates.
func HTMXHead() templ.Component {
	return htmxHead()
}
