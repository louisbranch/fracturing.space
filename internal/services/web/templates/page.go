package templates

import (
	"strings"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// PageContext provides shared layout context for pages.
type PageContext struct {
	Lang                   string
	Loc                    Localizer
	CurrentPath            string
	CurrentQuery           string
	ChatFallbackPort       string
	CampaignName           string
	CampaignCoverImageURL  string
	UserName               string
	UserAvatarURL          string
	HasUnreadNotifications bool
	AppName                string
}

func isGamePagePath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if path == routepath.AppRoot {
		return true
	}
	if path == routepath.AppProfile {
		return true
	}
	if path == routepath.AppSettings || strings.HasPrefix(path, routepath.AppSettingsPrefix) {
		return true
	}
	if path == routepath.AppCampaigns || strings.HasPrefix(path, routepath.AppCampaignsPrefix) {
		return true
	}
	if path == routepath.AppInvites || strings.HasPrefix(path, routepath.AppInvites+"/") {
		return true
	}
	if path == routepath.AppNotifications || strings.HasPrefix(path, routepath.AppNotificationsPrefix) {
		return true
	}
	return false
}
