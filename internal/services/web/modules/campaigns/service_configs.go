package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigncharacters "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/characters"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaigninvites "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/invites"
	campaignoverview "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/overview"
	campaignparticipants "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/participants"
	campaignsessions "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/sessions"
)

// pageServiceConfig groups shared workspace-shell app config for detail
// surfaces.
type pageServiceConfig = campaigndetail.PageServiceConfig

// catalogServiceConfig groups campaign catalog app config.
type catalogServiceConfig struct {
	Catalog campaignapp.CatalogServiceConfig
}

// starterServiceConfig groups protected starter app config.
type starterServiceConfig struct {
	Starter campaignapp.StarterServiceConfig
}

// overviewServiceConfig groups overview, AI binding, and campaign settings app
// config.
type overviewServiceConfig = campaignoverview.ServiceConfig

// participantServiceConfig groups participant read and mutation app config.
type participantServiceConfig = campaignparticipants.ServiceConfig

// characterServiceConfig groups character read, control, mutation, and
// creation app config.
type characterServiceConfig = campaigncharacters.ServiceConfig

// sessionServiceConfig groups session mutation app config.
type sessionServiceConfig = campaignsessions.ServiceConfig

// inviteServiceConfig groups invite read, mutation, and search-adjacent app
// config.
type inviteServiceConfig = campaigninvites.ServiceConfig
