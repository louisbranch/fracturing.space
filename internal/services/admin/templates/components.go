package templates

import (
	"net/url"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
)

// PageHeading holds header metadata for pages.
type PageHeading struct {
	// Title is the page heading.
	Title string
	// Breadcrumbs renders a path trail for the page.
	Breadcrumbs []Breadcrumb
	// ActionURL renders a CTA button when set.
	ActionURL string
	// ActionLabel is the CTA button label.
	ActionLabel string
}

type Breadcrumb = sharedtemplates.BreadcrumbItem

// AppendQueryParam appends a single query parameter to a URL.
func AppendQueryParam(baseURL string, key string, value string) string {
	encodedKey := url.QueryEscape(key)
	encodedValue := url.QueryEscape(value)
	if strings.Contains(baseURL, "?") {
		return baseURL + "&" + encodedKey + "=" + encodedValue
	}
	return baseURL + "?" + encodedKey + "=" + encodedValue
}

func userDetailBreadcrumbLabel(view UserDetailPageView, loc Localizer) string {
	if view.Detail == nil {
		return T(loc, "users.detail.heading")
	}
	if strings.TrimSpace(view.Detail.Email) != "" {
		return view.Detail.Email
	}
	return strings.TrimSpace(view.Detail.ID)
}
