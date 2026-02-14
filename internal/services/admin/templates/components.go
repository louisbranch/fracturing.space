package templates

import (
	"net/url"
	"strings"
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

// Breadcrumb represents a single breadcrumb item.
type Breadcrumb struct {
	// Label is the visible label.
	Label string
	// URL is the optional navigation target.
	URL string
}

// AppendQueryParam appends a single query parameter to a URL.
func AppendQueryParam(baseURL string, key string, value string) string {
	encodedKey := url.QueryEscape(key)
	encodedValue := url.QueryEscape(value)
	if strings.Contains(baseURL, "?") {
		return baseURL + "&" + encodedKey + "=" + encodedValue
	}
	return baseURL + "?" + encodedKey + "=" + encodedValue
}
