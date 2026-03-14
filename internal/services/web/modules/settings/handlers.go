package settings

import (
	"context"

	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// settingsProfileService defines the profile contract used by settings handlers.
type settingsProfileService = settingsapp.ProfileService

// settingsLocaleService defines the locale contract used by settings handlers.
type settingsLocaleService = settingsapp.LocaleService

// settingsSecurityService defines the security contract used by settings handlers.
type settingsSecurityService = settingsapp.SecurityService

// settingsAIKeyService defines the AI credential contract used by settings handlers.
type settingsAIKeyService = settingsapp.AIKeyService

// settingsAIAgentService defines the AI agent contract used by settings handlers.
type settingsAIAgentService = settingsapp.AIAgentService

// DashboardSync exposes dashboard cache refresh hooks needed by settings mutations.
type DashboardSync interface {
	ProfileSaved(context.Context, string)
}

// settingsSurfaceAvailability tracks which settings pages should be discoverable.
type settingsSurfaceAvailability struct {
	profile  bool
	locale   bool
	security bool
	aiKeys   bool
	aiAgents bool
}

// any reports whether at least one settings surface is available.
func (a settingsSurfaceAvailability) any() bool {
	return a.profile || a.locale || a.security || a.aiKeys || a.aiAgents
}

// defaultPath returns the first route that should own `/app/settings`.
func (a settingsSurfaceAvailability) defaultPath() string {
	switch {
	case a.profile:
		return routepath.AppSettingsProfile
	case a.locale:
		return routepath.AppSettingsLocale
	case a.security:
		return routepath.AppSettingsSecurity
	case a.aiKeys:
		return routepath.AppSettingsAIKeys
	case a.aiAgents:
		return routepath.AppSettingsAIAgents
	default:
		return ""
	}
}

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	modulehandler.Base
	profile      settingsProfileService
	locale       settingsLocaleService
	security     settingsSecurityService
	aiKeys       settingsAIKeyService
	aiAgents     settingsAIAgentService
	availability settingsSurfaceAvailability
	flashMeta    requestmeta.SchemePolicy
	sync         DashboardSync
}

// newHandlers builds package wiring for this web seam.
func newHandlers(
	profile settingsProfileService,
	locale settingsLocaleService,
	security settingsSecurityService,
	aiKeys settingsAIKeyService,
	aiAgents settingsAIAgentService,
	availability settingsSurfaceAvailability,
	base modulehandler.Base,
	policy requestmeta.SchemePolicy,
	sync DashboardSync,
) handlers {
	return handlers{
		Base:         base,
		profile:      profile,
		locale:       locale,
		security:     security,
		aiKeys:       aiKeys,
		aiAgents:     aiAgents,
		availability: availability,
		flashMeta:    policy,
		sync:         sync,
	}
}
