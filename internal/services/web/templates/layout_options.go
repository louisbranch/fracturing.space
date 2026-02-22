package templates

import (
	"github.com/a-h/templ"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"golang.org/x/text/message"
)

type LayoutOptions struct {
	Title                  string
	Lang                   string
	AppName                string
	Loc                    Localizer
	CurrentPath            string
	CampaignName           string
	MainStyle              string
	MainClass              string
	UserName               string
	UserAvatarURL          string
	HasUnreadNotifications bool
	HeadingAction          templ.Component
	ChromeMenu             templ.Component
	CustomBreadcrumbs      []sharedtemplates.BreadcrumbItem
	UseCustomBreadcrumbs   bool
}

// LayoutOptionsForPage builds the shared layout options from a page context and title key.
func LayoutOptionsForPage(page PageContext, titleKey message.Reference, useCustomBreadcrumbs bool) LayoutOptions {
	return LayoutOptions{
		Title:                  T(page.Loc, titleKey),
		Lang:                   page.Lang,
		AppName:                page.AppName,
		Loc:                    page.Loc,
		CurrentPath:            page.CurrentPath,
		CampaignName:           page.CampaignName,
		UserName:               page.UserName,
		UserAvatarURL:          page.UserAvatarURL,
		HasUnreadNotifications: page.HasUnreadNotifications,
		UseCustomBreadcrumbs:   useCustomBreadcrumbs,
	}
}
