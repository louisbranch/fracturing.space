package workflow

import (
	"context"
	"net/url"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

// AppService is the narrow campaigns app surface needed by character-creation
// orchestration in the transport-owned workflow package.
type AppService interface {
	CampaignCharacterCreationData(context.Context, string, string, language.Tag) (campaignapp.CampaignCharacterCreationData, error)
	CampaignCharacterCreationProgress(context.Context, string, string) (campaignapp.CampaignCharacterCreationProgress, error)
	ApplyCharacterCreationStep(context.Context, string, string, *campaignapp.CampaignCharacterCreationStepInput) error
	ResetCharacterCreationWorkflow(context.Context, string, string) error
}

// Service coordinates one character-creation surface across app reads,
// app mutations, and system-specific workflow implementations.
type Service struct {
	app       AppService
	workflows Registry
}

// PageData carries the transport-owned result needed to render one creation
// page or detail fragment.
type PageData struct {
	CharacterName string
	Creation      campaignrender.CampaignCharacterCreationView
}

// NewService constructs a creation service from the campaigns app seam and the
// installed workflow registry.
func NewService(app AppService, workflows Registry) Service {
	return Service{app: app, workflows: workflows}
}

// Enabled reports whether one supported workflow exists for the campaign system.
func (s Service) Enabled(system string) bool {
	return s.resolve(system) != nil
}

// LoadPage assembles one system-specific character-creation page contract.
func (s Service) LoadPage(ctx context.Context, campaignID string, characterID string, locale language.Tag, system string) (PageData, error) {
	workflow, err := s.require(system)
	if err != nil {
		return PageData{}, err
	}
	if s.app == nil {
		return PageData{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character creation service is not configured")
	}
	data, err := s.app.CampaignCharacterCreationData(ctx, campaignID, characterID, locale)
	if err != nil {
		return PageData{}, err
	}
	creation := workflow.AssembleCatalog(data.Progress, data.Catalog, data.Profile)
	return PageData{
		CharacterName: strings.TrimSpace(data.Profile.CharacterName),
		Creation:      workflow.CreationView(creation),
	}, nil
}

// ApplyStep parses one workflow-specific step submission and forwards the
// resulting domain mutation to the campaigns app service.
func (s Service) ApplyStep(ctx context.Context, campaignID string, characterID string, system string, form url.Values) error {
	workflow, err := s.require(system)
	if err != nil {
		return err
	}
	if s.app == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character creation service is not configured")
	}
	progress, err := s.app.CampaignCharacterCreationProgress(ctx, campaignID, characterID)
	if err != nil {
		return err
	}
	if progress.Ready {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_already_complete", "character creation workflow is already complete")
	}
	stepInput, err := workflow.ParseStepInput(form, progress.NextStep)
	if err != nil {
		return err
	}
	return s.app.ApplyCharacterCreationStep(ctx, campaignID, characterID, stepInput)
}

// Reset forwards a workflow reset mutation through the campaigns app service.
func (s Service) Reset(ctx context.Context, campaignID string, characterID string) error {
	if s.app == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character creation service is not configured")
	}
	return s.app.ResetCharacterCreationWorkflow(ctx, campaignID, characterID)
}

// resolve returns the installed workflow implementation for one campaign system.
func (s Service) resolve(system string) CharacterCreation {
	if s.workflows == nil {
		return nil
	}
	resolvedSystem, ok := parseGameSystem(system)
	if !ok {
		return nil
	}
	return s.workflows[resolvedSystem]
}

// require resolves a supported workflow or returns the canonical unavailable-step error.
func (s Service) require(system string) (CharacterCreation, error) {
	workflow := s.resolve(system)
	if workflow == nil {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}
	return workflow, nil
}

// parseGameSystem maps route-level system labels to canonical workflow registry keys.
func parseGameSystem(system string) (campaignapp.GameSystem, bool) {
	switch normalized := strings.ToLower(strings.TrimSpace(system)); normalized {
	case string(campaignapp.GameSystemDaggerheart):
		return campaignapp.GameSystemDaggerheart, true
	default:
		return "", false
	}
}
