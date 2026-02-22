package templates

import (
	"strings"
)

// PageContext provides shared layout context for pages.
type PageContext struct {
	Lang                  string
	Loc                   Localizer
	CurrentPath           string
	CurrentQuery          string
	CampaignName          string
	CampaignCoverImageURL string
	UserName              string
	UserAvatarURL         string
	AppName               string
}

func isGamePagePath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if path == "/dashboard" {
		return true
	}
	if path == "/profile" {
		return true
	}
	if path == "/settings" || strings.HasPrefix(path, "/settings/") {
		return true
	}
	if path == "/campaigns" || strings.HasPrefix(path, "/campaigns/") {
		return true
	}
	if path == "/invites" || strings.HasPrefix(path, "/invites/") {
		return true
	}
	return false
}
