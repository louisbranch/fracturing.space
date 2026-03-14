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
	chatFallbackPort string
	workflows        campaignworkflow.Registry
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
		s = campaignapp.NewService(nil)
	}
	var workflowMap campaignworkflow.Registry
	if len(workflows) > 0 {
		workflowMap = workflows[0]
	}
	return handlers{
		Base:             base,
		service:          s,
		chatFallbackPort: chatFallbackPort,
		workflows:        workflowMap,
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

// resolveWorkflow resolves request-scoped values needed by this package.
func (h handlers) resolveWorkflow(system string) campaignworkflow.CharacterCreation {
	if h.workflows == nil {
		return nil
	}
	resolvedSystem, ok := parseAppGameSystem(system)
	if !ok {
		return nil
	}
	return h.workflows[resolvedSystem]
}
