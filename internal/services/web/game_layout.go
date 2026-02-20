package web

import (
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"
)

const gamePageDescription = "Open-source, server-authoritative engine for deterministic tabletop RPG campaigns and AI game masters."

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

func writeGamePageStart(w http.ResponseWriter, title string, appName string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if strings.TrimSpace(appName) == "" {
		appName = branding.AppName
	}
	escapedAppName := html.EscapeString(appName)
	_, _ = io.WriteString(
		w,
		"<!doctype html><html lang=\"en\" data-theme=\"fracturing\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>"+html.EscapeString(title)+"</title><meta name=\"description\" content=\""+html.EscapeString(gamePageDescription)+"\"><link href=\"https://cdn.jsdelivr.net/npm/daisyui@5.5.18\" rel=\"stylesheet\" type=\"text/css\" integrity=\"sha384-ww1btmC3Ah3rEb6jt/coOxyQ9JYMoxQpFSB/bxdE20ZYMK4kWSb+TwcgbHR/GFCq\" crossorigin=\"anonymous\"/><script src=\"https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4.1.18\" integrity=\"sha384-WrpyCFNrMmN/IC7KmMNiXxIouXEFpoDIuJ2P+ys++uYEzegAW2MSl+X6Unsahaij\" crossorigin=\"anonymous\"></script><link rel=\"stylesheet\" href=\"/static/landing.css\"></head><body class=\"landing-body game-body\" data-layout=\"game\"><nav class=\"navbar bg-base-200 fixed top-0 w-full z-50\"><div class=\"navbar-start w-auto\"><h3 class=\"text-lg font-bold\"><a href=\"/\">"+escapedAppName+"</a></h3></div><div class=\"navbar-end w-full justify-end gap-3\"><form method=\"POST\" action=\"/auth/logout\"><button type=\"submit\" class=\"btn btn-ghost btn-sm\">Sign out</button></form></div></nav><div class=\"max-w-screen-xl mx-auto px-4 pt-24\" id=\"main\"><main aria-label=\"Game\" class=\"p-4\">",
	)
}

func writeGamePageEnd(w http.ResponseWriter) {
	_, _ = io.WriteString(w, "</main></div></body></html>")
}
