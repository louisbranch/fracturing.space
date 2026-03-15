package workflow

import (
	"context"
	"net/url"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

// PageAppService is the narrow campaigns app surface needed by character-creation
// page assembly in the transport-owned workflow package.
type PageAppService interface {
	CampaignCharacterCreationProgress(context.Context, string, string) (Progress, error)
	CampaignCharacterCreationCatalog(context.Context, language.Tag) (Catalog, error)
	CampaignCharacterCreationProfile(context.Context, string, string) (Profile, error)
}

// MutationAppService is the narrow campaigns app surface needed by character-creation
// step and reset mutations in the transport-owned workflow package.
type MutationAppService interface {
	CampaignCharacterCreationProgress(context.Context, string, string) (Progress, error)
	ApplyCharacterCreationStep(context.Context, string, string, *StepInput) error
	ResetCharacterCreationWorkflow(context.Context, string, string) error
}

// workflowResolver centralizes workflow registry resolution shared by page and
// mutation services.
type workflowResolver struct {
	workflows Registry
}

// PageData carries the transport-owned result needed to render one creation
// page or detail fragment.
type PageData struct {
	CharacterName string
	Creation      CharacterCreationView
}

// PageService coordinates one character-creation page surface across app reads
// and system-specific workflow implementations.
type PageService struct {
	app      PageAppService
	resolver workflowResolver
}

// MutationService coordinates one character-creation mutation surface across
// app mutations and system-specific workflow implementations.
type MutationService struct {
	app      MutationAppService
	resolver workflowResolver
}

// NewPageService constructs a creation page service from the campaigns app seam
// and the installed workflow registry.
func NewPageService(app PageAppService, workflows Registry) PageService {
	return PageService{app: app, resolver: workflowResolver{workflows: workflows}}
}

// NewMutationService constructs a creation mutation service from the campaigns
// app seam and the installed workflow registry.
func NewMutationService(app MutationAppService, workflows Registry) MutationService {
	return MutationService{app: app, resolver: workflowResolver{workflows: workflows}}
}

// Enabled reports whether one supported workflow exists for the campaign system.
func (s workflowResolver) Enabled(system string) bool {
	return s.resolve(system) != nil
}

// Enabled reports whether one supported workflow exists for the campaign system.
func (s PageService) Enabled(system string) bool {
	return s.resolver.Enabled(system)
}

// Enabled reports whether one supported workflow exists for the campaign system.
func (s MutationService) Enabled(system string) bool {
	return s.resolver.Enabled(system)
}

// LoadPage assembles one system-specific character-creation page contract.
func (s PageService) LoadPage(ctx context.Context, campaignID string, characterID string, locale language.Tag, system string) (PageData, error) {
	workflow, err := s.resolver.require(system)
	if err != nil {
		return PageData{}, err
	}
	if s.app == nil {
		return PageData{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character creation service is not configured")
	}
	progress, err := s.app.CampaignCharacterCreationProgress(ctx, campaignID, characterID)
	if err != nil {
		return PageData{}, err
	}
	catalog, err := s.app.CampaignCharacterCreationCatalog(ctx, locale)
	if err != nil {
		return PageData{}, err
	}
	profile, err := s.app.CampaignCharacterCreationProfile(ctx, campaignID, characterID)
	if err != nil {
		return PageData{}, err
	}
	return PageData{
		CharacterName: strings.TrimSpace(profile.CharacterName),
		Creation:      workflow.BuildView(progress, catalog, profile),
	}, nil
}

// ApplyStep parses one workflow-specific step submission and forwards the
// resulting domain mutation to the campaigns app service.
func (s MutationService) ApplyStep(ctx context.Context, campaignID string, characterID string, system string, form url.Values) error {
	workflow, err := s.resolver.require(system)
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
func (s MutationService) Reset(ctx context.Context, campaignID string, characterID string) error {
	if s.app == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character creation service is not configured")
	}
	return s.app.ResetCharacterCreationWorkflow(ctx, campaignID, characterID)
}

// resolve returns the installed workflow implementation for one campaign system.
func (s workflowResolver) resolve(system string) CharacterCreation {
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
func (s workflowResolver) require(system string) (CharacterCreation, error) {
	workflow := s.resolve(system)
	if workflow == nil {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}
	return workflow, nil
}

// parseGameSystem maps route-level system labels to canonical workflow registry keys.
func parseGameSystem(system string) (GameSystem, bool) {
	switch normalized := strings.ToLower(strings.TrimSpace(system)); normalized {
	case string(GameSystemDaggerheart):
		return GameSystemDaggerheart, true
	default:
		return "", false
	}
}
