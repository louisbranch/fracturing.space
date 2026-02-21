package templates

import "golang.org/x/text/message"

// LayoutOptionsForPage builds the shared layout options from a page context and title key.
func LayoutOptionsForPage(page PageContext, titleKey message.Reference, useCustomBreadcrumbs bool) LayoutOptions {
	return LayoutOptions{
		Title:                T(page.Loc, titleKey),
		Lang:                 page.Lang,
		AppName:              page.AppName,
		Loc:                  page.Loc,
		CurrentPath:          page.CurrentPath,
		CampaignName:         page.CampaignName,
		UserName:             page.UserName,
		UserAvatarURL:        page.UserAvatarURL,
		UseCustomBreadcrumbs: useCustomBreadcrumbs,
	}
}
