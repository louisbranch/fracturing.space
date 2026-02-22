package admin

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	sharedhtmx "github.com/louisbranch/fracturing.space/internal/services/shared/htmx"
	"golang.org/x/text/message"
)

// authClient returns the currently configured auth client.
func (h *Handler) authClient() authv1.AuthServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.AuthClient()
}

// accountClient returns the currently configured account client.
func (h *Handler) accountClient() authv1.AccountServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.AccountClient()
}

// daggerheartContentClient returns the Daggerheart content client.
func (h *Handler) daggerheartContentClient() daggerheartv1.DaggerheartContentServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.DaggerheartContentClient()
}

// campaignClient returns the currently configured campaign client.
func (h *Handler) campaignClient() statev1.CampaignServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.CampaignClient()
}

// sessionClient returns the currently configured session client.
func (h *Handler) sessionClient() statev1.SessionServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SessionClient()
}

// characterClient returns the currently configured character client.
func (h *Handler) characterClient() statev1.CharacterServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.CharacterClient()
}

// participantClient returns the currently configured participant client.
func (h *Handler) participantClient() statev1.ParticipantServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.ParticipantClient()
}

// inviteClient returns the currently configured invite client.
func (h *Handler) inviteClient() statev1.InviteServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.InviteClient()
}

// snapshotClient returns the currently configured snapshot client.
func (h *Handler) snapshotClient() statev1.SnapshotServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SnapshotClient()
}

// eventClient returns the currently configured event client.
func (h *Handler) eventClient() statev1.EventServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.EventClient()
}

// statisticsClient returns the currently configured statistics client.
func (h *Handler) statisticsClient() statev1.StatisticsServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.StatisticsClient()
}

// systemClient returns the currently configured system client.
func (h *Handler) systemClient() statev1.SystemServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SystemClient()
}

// isHTMXRequest reports whether the request originated from HTMX.
func isHTMXRequest(r *http.Request) bool {
	return sharedhtmx.IsHTMXRequest(r)
}

// splitPathParts returns non-empty path segments.
func splitPathParts(path string) []string {
	rawParts := strings.Split(path, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	return parts
}

func htmxDefaultPageTitle() string {
	return sharedhtmx.TitleTag("Admin | " + templates.AppName())
}

func htmxLocalizedPageTitle(loc *message.Printer, title string, args ...any) string {
	if loc == nil {
		return htmxDefaultPageTitle()
	}
	return sharedhtmx.TitleTag(templates.ComposeAdminPageTitle(templates.T(loc, title, args...)))
}

// renderPage renders page components with consistent HTMX and non-HTMX behavior.
func renderPage(w http.ResponseWriter, r *http.Request, fragment templ.Component, full templ.Component, htmxTitle string) {
	sharedhtmx.RenderPage(w, r, fragment, full, htmxTitle)
}
