// Package modulehandler provides shared transport helpers for admin modules.
package modulehandler

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/requestctx"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	sharedhtmx "github.com/louisbranch/fracturing.space/internal/services/shared/htmx"
	"golang.org/x/text/message"
)

const grpcRequestTimeout = timeouts.GRPCRequest

// ClientProvider supplies gRPC clients for admin modules.
type ClientProvider interface {
	AuthClient() authv1.AuthServiceClient
	AccountClient() authv1.AccountServiceClient
	CampaignClient() statev1.CampaignServiceClient
	SessionClient() statev1.SessionServiceClient
	CharacterClient() statev1.CharacterServiceClient
	ParticipantClient() statev1.ParticipantServiceClient
	InviteClient() statev1.InviteServiceClient
	SnapshotClient() statev1.SnapshotServiceClient
	EventClient() statev1.EventServiceClient
	StatisticsClient() statev1.StatisticsServiceClient
	SystemClient() statev1.SystemServiceClient
	DaggerheartContentClient() daggerheartv1.DaggerheartContentServiceClient
}

// Base carries shared request-scoped module dependencies.
type Base struct {
	clientProvider ClientProvider
}

// NewBase returns a shared module handler base.
func NewBase(clientProvider ClientProvider) Base {
	return Base{clientProvider: clientProvider}
}

// Localizer resolves request localizer and selected language.
func (b Base) Localizer(w http.ResponseWriter, r *http.Request) (*message.Printer, string) {
	tag, persist := i18n.ResolveTag(r)
	if persist {
		i18n.SetLanguageCookie(w, tag)
	}
	return i18n.Printer(tag), tag.String()
}

// PageContext builds common template page context from request state.
func (b Base) PageContext(lang string, loc *message.Printer, r *http.Request) templates.PageContext {
	path := ""
	query := ""
	if r != nil && r.URL != nil {
		path = r.URL.Path
		query = r.URL.RawQuery
	}
	return templates.PageContext{
		Lang:         lang,
		Loc:          loc,
		CurrentPath:  path,
		CurrentQuery: query,
	}
}

// GameGRPCCallContext creates a bounded game RPC context with user identity.
func (b Base) GameGRPCCallContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, grpcRequestTimeout)
	if userID := strings.TrimSpace(requestctx.UserIDFromContext(parent)); userID != "" {
		ctx = grpcauthctx.WithUserID(ctx, userID)
	}
	return ctx, cancel
}

// IsHTMXRequest reports whether the request originated from HTMX.
func (b Base) IsHTMXRequest(r *http.Request) bool {
	return sharedhtmx.IsHTMXRequest(r)
}

// RenderPage applies shared HTMX/full-page rendering behavior.
func (b Base) RenderPage(w http.ResponseWriter, r *http.Request, fragment templ.Component, full templ.Component, htmxTitle string) {
	sharedhtmx.RenderPage(w, r, fragment, full, htmxTitle)
}

// HTMXLocalizedPageTitle returns a localized page title for HTMX swaps.
func (b Base) HTMXLocalizedPageTitle(loc *message.Printer, title string, args ...any) string {
	if loc == nil {
		return sharedhtmx.TitleTag("Admin | " + templates.AppName())
	}
	return sharedhtmx.TitleTag(templates.ComposeAdminPageTitle(templates.T(loc, title, args...)))
}

// AuthClient returns the configured auth client.
func (b Base) AuthClient() authv1.AuthServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.AuthClient()
}

// AccountClient returns the configured account client.
func (b Base) AccountClient() authv1.AccountServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.AccountClient()
}

// CampaignClient returns the configured campaign client.
func (b Base) CampaignClient() statev1.CampaignServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.CampaignClient()
}

// SessionClient returns the configured session client.
func (b Base) SessionClient() statev1.SessionServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.SessionClient()
}

// CharacterClient returns the configured character client.
func (b Base) CharacterClient() statev1.CharacterServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.CharacterClient()
}

// ParticipantClient returns the configured participant client.
func (b Base) ParticipantClient() statev1.ParticipantServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.ParticipantClient()
}

// InviteClient returns the configured invite client.
func (b Base) InviteClient() statev1.InviteServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.InviteClient()
}

// SnapshotClient returns the configured snapshot client.
func (b Base) SnapshotClient() statev1.SnapshotServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.SnapshotClient()
}

// EventClient returns the configured event client.
func (b Base) EventClient() statev1.EventServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.EventClient()
}

// StatisticsClient returns the configured statistics client.
func (b Base) StatisticsClient() statev1.StatisticsServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.StatisticsClient()
}

// SystemClient returns the configured system client.
func (b Base) SystemClient() statev1.SystemServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.SystemClient()
}

// DaggerheartContentClient returns the configured content client.
func (b Base) DaggerheartContentClient() daggerheartv1.DaggerheartContentServiceClient {
	if b.clientProvider == nil {
		return nil
	}
	return b.clientProvider.DaggerheartContentClient()
}
