package settings

import (
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// DashboardSync keeps settings mutations aligned with dashboard freshness.
type DashboardSync = dashboardsync.Service

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
	account      settingsapp.AccountService
	ai           settingsapp.AIService
	availability settingsSurfaceAvailability
	flashMeta    requestmeta.SchemePolicy
	sync         DashboardSync
}

// handlerServices groups the settings app seams consumed by transport.
type handlerServices struct {
	Account settingsapp.AccountService
	AI      settingsapp.AIService
}

// handlersConfig keeps root transport wiring explicit by owned service group.
type handlersConfig struct {
	Services     handlerServices
	Availability settingsSurfaceAvailability
	Base         modulehandler.Base
	Policy       requestmeta.SchemePolicy
	Sync         DashboardSync
}

// newHandlers builds package wiring for this web seam.
func newHandlers(config handlersConfig) handlers {
	sync := config.Sync
	if sync == nil {
		sync = dashboardsync.Noop{}
	}
	return handlers{
		Base:         config.Base,
		account:      config.Services.Account,
		ai:           config.Services.AI,
		availability: config.Availability,
		flashMeta:    config.Policy,
		sync:         sync,
	}
}
