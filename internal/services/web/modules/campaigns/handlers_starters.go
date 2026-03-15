package campaigns

import (
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/forminput"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// starterPreviewHeader keeps starter preview breadcrumbs aligned with the campaigns shell.
func starterPreviewHeader(loc webtemplates.Localizer, title string) *webtemplates.AppMainHeader {
	if strings.TrimSpace(title) == "" {
		title = "Starter"
	}
	return &webtemplates.AppMainHeader{
		Title: title,
		Breadcrumbs: []sharedtemplates.BreadcrumbItem{
			{Label: webtemplates.T(loc, "game.campaigns.title"), URL: routepath.AppCampaigns},
			{Label: "Starter"},
		},
	}
}

// handleStarterPreview renders the protected starter preview page from discovery-owned entry data.
func (h handlers) handleStarterPreview(w http.ResponseWriter, r *http.Request) {
	starterKey, ok := h.routeStarterKey(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	loc, _ := h.PageLocalizer(w, r)
	ctx, _ := h.RequestContextAndUserID(r)
	preview, err := h.starters.starters.StarterPreview(ctx, starterKey)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.WritePage(
		w,
		r,
		preview.Title,
		http.StatusOK,
		starterPreviewHeader(loc, preview.Title),
		webtemplates.AppMainLayoutOptions{},
		StarterPreviewFragment(mapStarterPreview(preview), loc),
	)
}

// handleStarterLaunch validates the AI selection and redirects into the new forked campaign.
func (h handlers) handleStarterLaunch(w http.ResponseWriter, r *http.Request) {
	starterKey, ok := h.routeStarterKey(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	redirectURL := routepath.AppCampaignStarter(starterKey)
	if !forminput.ParseOrRedirectErrorNotice(w, r, "error.web.message.failed_to_launch_starter", redirectURL) {
		return
	}

	ctx, _ := h.RequestContextAndUserID(r)
	result, err := h.starters.starters.LaunchStarter(ctx, starterKey, campaignapp.LaunchStarterInput{
		AIAgentID: strings.TrimSpace(r.FormValue("ai_agent_id")),
	})
	if err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_launch_starter", redirectURL)
		return
	}
	httpx.WriteRedirect(w, r, routepath.AppCampaign(result.CampaignID))
}

// starterPreviewView keeps transport-local rendering fields explicit for the preview template.
type starterPreviewView struct {
	EntryID              string
	Title                string
	Description          string
	CampaignTheme        string
	Hook                 string
	PlaystyleLabel       string
	CharacterName        string
	CharacterSummary     string
	System               string
	Difficulty           string
	Duration             string
	GmMode               string
	Players              string
	Tags                 []string
	AIAgentOptions       []campaignapp.CampaignAIAgentOption
	HasAvailableAIAgents bool
}

// mapStarterPreview isolates app-to-template field mapping from the handler flow.
func mapStarterPreview(preview campaignapp.CampaignStarterPreview) starterPreviewView {
	return starterPreviewView{
		EntryID:              preview.EntryID,
		Title:                preview.Title,
		Description:          preview.Description,
		CampaignTheme:        preview.CampaignTheme,
		Hook:                 preview.Hook,
		PlaystyleLabel:       preview.PlaystyleLabel,
		CharacterName:        preview.CharacterName,
		CharacterSummary:     preview.CharacterSummary,
		System:               preview.System,
		Difficulty:           preview.Difficulty,
		Duration:             preview.Duration,
		GmMode:               preview.GmMode,
		Players:              preview.Players,
		Tags:                 preview.Tags,
		AIAgentOptions:       preview.AIAgentOptions,
		HasAvailableAIAgents: preview.HasAvailableAIAgents,
	}
}
