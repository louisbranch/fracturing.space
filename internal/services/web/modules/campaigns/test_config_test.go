package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

func configWithGateway(gateway campaignapp.CampaignGateway, base modulehandler.Base, workflows campaignworkflow.Registry) Config {
	return Config{
		ReadGateway:      gateway,
		MutationGateway:  gateway,
		AuthzGateway:     gateway,
		Base:             base,
		ChatFallbackPort: "",
		Workflows:        workflows,
	}
}

func configWithGatewayAndChatFallback(
	gateway campaignapp.CampaignGateway,
	base modulehandler.Base,
	workflows campaignworkflow.Registry,
	chatFallbackPort string,
) Config {
	cfg := configWithGateway(gateway, base, workflows)
	cfg.ChatFallbackPort = chatFallbackPort
	return cfg
}

func configWithGatewayAndSync(
	gateway campaignapp.CampaignGateway,
	base modulehandler.Base,
	workflows campaignworkflow.Registry,
	sync DashboardSync,
) Config {
	cfg := configWithGateway(gateway, base, workflows)
	cfg.DashboardSync = sync
	return cfg
}
