package campaigns

import (
	"context"
	"time"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// DashboardSync exposes dashboard refresh hooks needed by campaign mutations.
type DashboardSync interface {
	CampaignCreated(context.Context, string, string)
	SessionStarted(context.Context, string, string)
	SessionEnded(context.Context, string, string)
	InviteChanged(context.Context, []string, string)
}

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	modulehandler.Base
	service          campaignapp.Service
	creation         campaignworkflow.Service
	chatFallbackPort string
	nowFunc          func() time.Time
	sync             DashboardSync
}

// newHandlers builds package wiring for this web seam.
func newHandlers(
	s campaignapp.Service,
	base modulehandler.Base,
	chatFallbackPort string,
	sync DashboardSync,
	workflows ...campaignworkflow.Registry,
) handlers {
	if s == nil {
		s = campaignapp.NewService(campaignapp.ServiceConfig{})
	}
	var workflowMap campaignworkflow.Registry
	if len(workflows) > 0 {
		workflowMap = workflows[0]
	}
	return handlers{
		Base:             base,
		service:          s,
		creation:         campaignworkflow.NewService(s, workflowMap),
		chatFallbackPort: chatFallbackPort,
		nowFunc:          time.Now,
		sync:             sync,
	}
}

// now centralizes this web behavior in one helper seam.
func (h handlers) now() time.Time {
	if h.nowFunc != nil {
		return h.nowFunc()
	}
	return time.Now()
}
