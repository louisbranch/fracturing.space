package templates

// PageContext provides shared layout context for pages.
type PageContext struct {
	Lang          string
	Loc           Localizer
	Impersonation *ImpersonationView
}

// ImpersonationView holds the active impersonation context for the UI.
type ImpersonationView struct {
	UserID      string
	DisplayName string
}

// ImpersonationLabel returns the display label for an impersonation session.
func ImpersonationLabel(view *ImpersonationView) string {
	if view == nil {
		return ""
	}
	if view.DisplayName != "" {
		return view.DisplayName
	}
	return view.UserID
}
