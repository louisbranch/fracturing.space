package templates

import sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"

// NotificationsLayoutOptions returns layout options for the notifications page.
func NotificationsLayoutOptions(page PageContext) LayoutOptions {
	options := LayoutOptionsForPage(page, "game.notifications.title", true)
	options.CustomBreadcrumbs = []sharedtemplates.BreadcrumbItem{}
	return options
}
