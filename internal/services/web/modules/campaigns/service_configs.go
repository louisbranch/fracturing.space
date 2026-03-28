package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// catalogServiceConfig groups campaign catalog app config.
type catalogServiceConfig struct {
	Catalog campaignapp.CatalogServiceConfig
}

// starterServiceConfig groups protected starter app config.
type starterServiceConfig struct {
	Starter campaignapp.StarterServiceConfig
}
