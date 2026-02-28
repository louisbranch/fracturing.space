package campaigns

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"golang.org/x/text/language"
)

// campaignService defines the service operations used by campaign handlers.
type campaignService interface {
	listCampaigns(ctx context.Context) ([]CampaignSummary, error)
	createCampaign(ctx context.Context, input CreateCampaignInput) (CreateCampaignResult, error)
	campaignWorkspace(ctx context.Context, campaignID string) (CampaignWorkspace, error)
	campaignParticipants(ctx context.Context, campaignID string) ([]CampaignParticipant, error)
	campaignCharacters(ctx context.Context, campaignID string) ([]CampaignCharacter, error)
	campaignSessions(ctx context.Context, campaignID string) ([]CampaignSession, error)
	campaignInvites(ctx context.Context, campaignID string) ([]CampaignInvite, error)
	startSession(ctx context.Context, campaignID string) error
	endSession(ctx context.Context, campaignID string) error
	updateParticipants(ctx context.Context, campaignID string) error
	createCharacter(ctx context.Context, campaignID string, input CreateCharacterInput) (CreateCharacterResult, error)
	updateCharacter(ctx context.Context, campaignID string) error
	controlCharacter(ctx context.Context, campaignID string) error
	createInvite(ctx context.Context, campaignID string) error
	revokeInvite(ctx context.Context, campaignID string) error
	resolveWorkflow(system string) CharacterCreationWorkflow
	campaignCharacterCreation(ctx context.Context, campaignID string, characterID string, locale language.Tag, workflow CharacterCreationWorkflow) (CampaignCharacterCreation, error)
	campaignCharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProgress, error)
	applyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *CampaignCharacterCreationStepInput) error
	resetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error
}

type handlers struct {
	modulehandler.Base
	service          campaignService
	chatFallbackPort string
	nowFunc          func() time.Time
}

func newHandlers(s service, base modulehandler.Base, chatFallbackPort string) handlers {
	return handlers{
		Base:             base,
		service:          s,
		chatFallbackPort: chatFallbackPort,
		nowFunc:          time.Now,
	}
}

func (h handlers) now() time.Time {
	if h.nowFunc != nil {
		return h.nowFunc()
	}
	return time.Now()
}
