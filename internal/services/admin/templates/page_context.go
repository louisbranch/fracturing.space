package templates

// PageContext provides shared layout context for admin pages.
type PageContext struct {
	Lang         string
	Loc          Localizer
	CurrentPath  string
	CurrentQuery string
}
