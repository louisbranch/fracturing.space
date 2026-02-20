package templates

// PageContext provides shared layout context for pages.
type PageContext struct {
	Lang         string
	Loc          Localizer
	CurrentPath  string
	CurrentQuery string
}
