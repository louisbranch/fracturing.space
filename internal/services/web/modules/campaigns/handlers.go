package campaigns

import (
	"strings"
	"time"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

type handlers struct {
	modulehandler.Base
	service          campaignapp.Service
	chatFallbackPort string
	workflows        map[string]CharacterCreationWorkflow
	nowFunc          func() time.Time
}

func newHandlers(
	s campaignapp.Service,
	base modulehandler.Base,
	chatFallbackPort string,
	workflows ...map[string]CharacterCreationWorkflow,
) handlers {
	if s == nil {
		s = campaignapp.NewService(nil)
	}
	var workflowMap map[string]CharacterCreationWorkflow
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

func (h handlers) now() time.Time {
	if h.nowFunc != nil {
		return h.nowFunc()
	}
	return time.Now()
}

func (h handlers) resolveWorkflow(system string) CharacterCreationWorkflow {
	if h.workflows == nil {
		return nil
	}
	return h.workflows[strings.ToLower(strings.TrimSpace(system))]
}
