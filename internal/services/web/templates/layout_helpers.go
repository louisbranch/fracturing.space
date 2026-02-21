package templates

import "strings"

func campaignNamesForPath(currentPath, campaignName string) map[string]string {
	normalizedPath := strings.TrimSpace(currentPath)
	normalizedCampaignName := strings.TrimSpace(campaignName)
	if normalizedPath == "" || normalizedCampaignName == "" {
		return nil
	}
	if !strings.HasPrefix(normalizedPath, "/campaigns/") {
		return nil
	}
	rawCampaignID := strings.TrimPrefix(normalizedPath, "/campaigns/")
	parts := strings.SplitN(rawCampaignID, "/", 2)
	campaignID := strings.TrimSpace(parts[0])
	if campaignID == "" || campaignID == "create" {
		return nil
	}
	return map[string]string{
		campaignID: normalizedCampaignName,
	}
}
