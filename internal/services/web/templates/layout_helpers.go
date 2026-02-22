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

func campaignWorkspaceInfo(currentPath string) (campaignID string, section string, ok bool) {
	normalizedPath := strings.TrimSpace(currentPath)
	if !strings.HasPrefix(normalizedPath, "/campaigns/") {
		return "", "", false
	}

	rawCampaignPath := strings.TrimPrefix(normalizedPath, "/campaigns/")
	rawCampaignPath = strings.TrimSpace(strings.Trim(rawCampaignPath, "/"))
	if rawCampaignPath == "" {
		return "", "", false
	}

	parts := strings.SplitN(rawCampaignPath, "/", 2)
	campaignID = strings.TrimSpace(parts[0])
	if campaignID == "" || campaignID == "create" {
		return "", "", false
	}
	if len(parts) == 1 {
		return campaignID, "chat", true
	}
	return campaignID, strings.TrimSpace(parts[1]), true
}

func campaignWorkspaceLinkClass(currentSection, targetSection string) string {
	current := strings.TrimSpace(strings.ToLower(currentSection))
	target := strings.TrimSpace(strings.ToLower(targetSection))
	if current == "" || target == "" {
		return ""
	}
	if current == target {
		return "menu-active"
	}
	if strings.HasPrefix(current, target+"/") {
		return "menu-active"
	}
	return ""
}

func settingsMenuLinkClass(currentPath, targetPath string) string {
	normalizedPath := strings.TrimSpace(currentPath)
	normalizedTarget := strings.TrimSpace(targetPath)
	if normalizedPath == "" || normalizedTarget == "" {
		return ""
	}
	if normalizedPath == normalizedTarget {
		return "menu-active"
	}
	if strings.HasPrefix(normalizedPath, normalizedTarget+"/") {
		return "menu-active"
	}
	return ""
}
