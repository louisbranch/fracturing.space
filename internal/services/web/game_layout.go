package web

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"
)

const gamePageContentType = "text/html; charset=utf-8"

func (h *handler) resolvedAppName() string {
	if h == nil {
		return branding.AppName
	}
	appName := strings.TrimSpace(h.config.AppName)
	if appName == "" {
		return branding.AppName
	}
	return appName
}

func writeGameContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", gamePageContentType)
}
