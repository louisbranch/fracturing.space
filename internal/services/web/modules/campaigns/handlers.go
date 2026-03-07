package campaigns

import (
	"time"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	modulehandler.Base
	service          campaignapp.Service
	chatFallbackPort string
	workflows        map[GameSystem]CharacterCreationWorkflow
	nowFunc          func() time.Time
}

// newHandlers builds package wiring for this web seam.
func newHandlers(
	s campaignapp.Service,
	base modulehandler.Base,
	chatFallbackPort string,
	workflows ...map[GameSystem]CharacterCreationWorkflow,
) handlers {
	if s == nil {
		s = campaignapp.NewService(nil)
	}
	var workflowMap map[GameSystem]CharacterCreationWorkflow
	if len(workflows) > 0 {
		workflowMap = workflows[0]
	}
	return handlers{
		Base:             base,
		service:          s,
		chatFallbackPort: chatFallbackPort,
		workflows:        workflowMap,
		nowFunc:          time.Now,
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
func (h handlers) resolveWorkflow(system string) CharacterCreationWorkflow {
	if h.workflows == nil {
		return nil
	}
	resolvedSystem, ok := parseAppGameSystem(system)
	if !ok {
		return nil
	}
	return h.workflows[resolvedSystem]
}
