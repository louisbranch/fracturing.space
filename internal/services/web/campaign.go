package web

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

const campaignNameCacheTTL = 5 * time.Minute

type campaignNameCache struct {
	name      string
	expiresAt time.Time
}

// renderCampaignPage renders the shared campaign shell once access has been
// verified by route-level auth and campaign membership checks.
func (h *handler) renderCampaignPage(w http.ResponseWriter, r *http.Request, campaignID string) {
	page := h.pageContextForCampaign(w, r, campaignID)
	campaignName := page.CampaignName
	if campaignName == "" {
		campaignName = strings.TrimSpace(campaignID)
	}
	if err := h.writePage(w, r, webtemplates.CampaignPage(page, campaignID, campaignName), ""); err != nil {
		log.Printf("web: failed to render campaign page: %v", err)
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func (h *handler) campaignCoverImage(ctx context.Context, campaignID string) string {
	campaignID = strings.TrimSpace(campaignID)
	config := Config{}
	if h != nil {
		config = h.config
	}
	if campaignID == "" {
		return campaignfeature.CampaignCoverImageURL(config.AssetBaseURL, "", "", "")
	}
	if h == nil {
		return campaignfeature.CampaignCoverImageURL(config.AssetBaseURL, campaignID, "", "")
	}

	cachedCampaign, ok := campaignfeature.NewCampaignCache(h.cacheStore).CachedCampaign(ctx, campaignID)
	if ok {
		return campaignfeature.CampaignCoverImageURL(config.AssetBaseURL, campaignID, cachedCampaign.GetCoverSetId(), cachedCampaign.GetCoverAssetId())
	}
	if h.campaignClient == nil {
		return campaignfeature.CampaignCoverImageURL(config.AssetBaseURL, campaignID, "", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	resp, err := h.campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{
		CampaignId: campaignID,
	})
	if err != nil || resp == nil || resp.GetCampaign() == nil {
		return campaignfeature.CampaignCoverImageURL(config.AssetBaseURL, campaignID, "", "")
	}
	campaignfeature.NewCampaignCache(h.cacheStore).SetCampaignCache(ctx, resp.GetCampaign())
	campaign := resp.GetCampaign()
	return campaignfeature.CampaignCoverImageURL(config.AssetBaseURL, campaignID, campaign.GetCoverSetId(), campaign.GetCoverAssetId())
}

func (h *handler) campaignDisplayName(ctx context.Context, campaignID string) string {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return ""
	}
	if cached := h.cachedCampaignName(campaignID); cached != "" {
		return cached
	}
	cachedCampaign, ok := campaignfeature.NewCampaignCache(h.cacheStore).CachedCampaign(ctx, campaignID)
	if ok {
		name := strings.TrimSpace(cachedCampaign.GetName())
		if name == "" {
			return campaignID
		}
		h.setCampaignNameCache(campaignID, name)
		return name
	}
	if h == nil || h.campaignClient == nil {
		return campaignID
	}
	if ctx == nil {
		ctx = context.Background()
	}

	resp, err := h.campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		return campaignID
	}
	if resp == nil || resp.GetCampaign() == nil {
		return campaignID
	}

	name := strings.TrimSpace(resp.GetCampaign().GetName())
	if name == "" {
		return campaignID
	}
	campaignfeature.NewCampaignCache(h.cacheStore).SetCampaignCache(ctx, resp.GetCampaign())
	h.setCampaignNameCache(campaignID, name)
	return name
}

func (h *handler) cachedCampaignName(campaignID string) string {
	if h == nil || h.campaignClient == nil || h.campaignNameCache == nil {
		return ""
	}

	h.campaignNameCacheMu.Lock()
	defer h.campaignNameCacheMu.Unlock()

	cached, ok := h.campaignNameCache[campaignID]
	if !ok {
		return ""
	}
	if time.Now().After(cached.expiresAt) {
		delete(h.campaignNameCache, campaignID)
		return ""
	}
	return cached.name
}

func (h *handler) setCampaignNameCache(campaignID, campaignName string) {
	if h == nil || campaignName == "" || h.campaignNameCache == nil {
		return
	}

	h.campaignNameCacheMu.Lock()
	defer h.campaignNameCacheMu.Unlock()
	h.campaignNameCache[campaignID] = campaignNameCache{
		name:      campaignName,
		expiresAt: time.Now().Add(campaignNameCacheTTL),
	}
}
