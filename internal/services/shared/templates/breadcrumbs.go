package templates

import (
	"strings"
)

// BreadcrumbItem represents one breadcrumb entry in a page trail.
type BreadcrumbItem struct {
	// Label is the visible breadcrumb text.
	Label string
	// URL is the optional destination for this breadcrumb entry.
	URL string
}

// BreadcrumbSegmentLabeler returns the label for a path segment.
//
// segment is the individual path segment while fullPath is the full accumulated path
// to the segment (for example, "/campaigns/abc/sessions").
type BreadcrumbSegmentLabeler func(segment string, fullPath string, loc Localizer) string

// PathBreadcrumbOptions controls how a breadcrumb trail is built from a path.
type PathBreadcrumbOptions struct {
	// IncludeRoot adds a dashboard-like root breadcrumb when enabled.
	IncludeRoot bool
	// RootPath is the URL used for the root breadcrumb when IncludeRoot is true.
	RootPath string
	// RootLabel is the localization key (or fallback string) for the root breadcrumb.
	RootLabel string
	// LabelForSegment resolves labels for each non-root segment.
	LabelForSegment BreadcrumbSegmentLabeler
	// CampaignNames maps campaign IDs to display names for path segments under `/campaigns/`.
	CampaignNames map[string]string
}

// BuildPathBreadcrumbs builds breadcrumb items from a request path for game pages.
func BuildPathBreadcrumbs(path string, loc Localizer) []BreadcrumbItem {
	return BuildPathBreadcrumbsWithOptions(path, loc, PathBreadcrumbOptions{
		IncludeRoot:     true,
		RootPath:        "/",
		RootLabel:       "dashboard.title",
		LabelForSegment: gamePathSegmentLabel,
	})
}

// BuildPathBreadcrumbsWithOptions builds breadcrumb items for a request path using
// caller-provided labeling behavior.
func BuildPathBreadcrumbsWithOptions(path string, loc Localizer, options PathBreadcrumbOptions) []BreadcrumbItem {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return []BreadcrumbItem{}
	}

	cleanPath := strings.Trim(path, "/")
	if cleanPath == "" {
		return []BreadcrumbItem{}
	}

	segments := strings.Split(cleanPath, "/")
	if options.LabelForSegment == nil {
		options.LabelForSegment = defaultSegmentLabel
	}
	if len(options.CampaignNames) > 0 {
		options.LabelForSegment = labelCampaignName(options.CampaignNames, options.LabelForSegment)
	}

	nonEmptyCount := 0
	for _, segment := range segments {
		if strings.TrimSpace(segment) != "" {
			nonEmptyCount++
		}
	}
	if nonEmptyCount == 0 {
		return []BreadcrumbItem{}
	}

	breadcrumbs := make([]BreadcrumbItem, 0, len(segments)+1)
	if options.IncludeRoot {
		rootPath := strings.TrimSpace(options.RootPath)
		if rootPath == "" {
			rootPath = "/"
		}
		breadcrumbs = append(breadcrumbs, BreadcrumbItem{Label: T(loc, options.RootLabel), URL: rootPath})
	}

	pathSoFar := ""
	validIndex := 0
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}

		pathSoFar += "/" + segment
		label := options.LabelForSegment(segment, pathSoFar, loc)
		if strings.TrimSpace(label) == "" {
			label = segment
		}
		breadcrumb := BreadcrumbItem{Label: label}
		if validIndex < nonEmptyCount-1 || nonEmptyCount == 1 {
			breadcrumb.URL = pathSoFar
		}
		breadcrumbs = append(breadcrumbs, breadcrumb)
		validIndex++
	}

	if len(breadcrumbs) == 1 && options.IncludeRoot {
		return []BreadcrumbItem{}
	}

	return breadcrumbs
}

func labelCampaignName(campaignNames map[string]string, next BreadcrumbSegmentLabeler) BreadcrumbSegmentLabeler {
	return func(segment string, fullPath string, loc Localizer) string {
		if campaignName := campaignNameForSegment(segment, fullPath, campaignNames); campaignName != "" {
			return campaignName
		}
		return next(segment, fullPath, loc)
	}
}

func campaignNameForSegment(segment string, fullPath string, campaignNames map[string]string) string {
	if segment == "" || len(campaignNames) == 0 {
		return ""
	}
	if strings.TrimSpace(segment) == "create" {
		return ""
	}
	fullPath = strings.TrimSpace(strings.Trim(fullPath, "/"))
	if fullPath == "" {
		return ""
	}
	parts := strings.Split(fullPath, "/")
	if len(parts) < 2 || parts[0] != "campaigns" {
		return ""
	}
	if parts[1] != strings.TrimSpace(segment) {
		return ""
	}
	campaignName, ok := campaignNames[strings.TrimSpace(segment)]
	if !ok {
		return ""
	}
	return strings.TrimSpace(campaignName)
}

func gamePathSegmentLabel(segment string, fullPath string, loc Localizer) string {
	switch segment {
	case "campaigns":
		return T(loc, "game.campaigns.title")
	case "invites":
		if fullPath == "/invites" {
			return T(loc, "game.my_invites.title")
		}
		return T(loc, "game.campaign_invites.title")
	case "notifications":
		return T(loc, "game.notifications.title")
	case "create":
		return T(loc, "game.create.title")
	case "sessions":
		return T(loc, "game.sessions.title")
	case "participants":
		return T(loc, "game.participants.title")
	case "characters":
		return T(loc, "game.characters.title")
	default:
		if segment == "" {
			return segment
		}
		return segment
	}
}

func defaultSegmentLabel(segment string, fullPath string, loc Localizer) string {
	_ = fullPath
	_ = loc
	return segment
}

func BuildPathBreadcrumbsForWeb(path string, loc Localizer, campaignNames ...map[string]string) []BreadcrumbItem {
	path = strings.TrimSpace(path)
	if path == "/app" || path == "/app/" {
		return []BreadcrumbItem{}
	}
	if !strings.HasPrefix(path, "/app/") {
		return []BreadcrumbItem{}
	}
	path = strings.TrimPrefix(path, "/app")

	var withCampaignNames map[string]string
	if len(campaignNames) > 0 {
		withCampaignNames = campaignNames[0]
	}
	breadcrumbs := BuildPathBreadcrumbsWithOptions(path, loc, PathBreadcrumbOptions{
		IncludeRoot:     false,
		LabelForSegment: gamePathSegmentLabel,
		CampaignNames:   withCampaignNames,
	})
	for idx := range breadcrumbs {
		if strings.TrimSpace(breadcrumbs[idx].URL) == "" {
			continue
		}
		breadcrumbs[idx].URL = "/app" + breadcrumbs[idx].URL
	}
	return breadcrumbs
}
