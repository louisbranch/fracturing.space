package domain

import (
	"fmt"
	"strings"
)

// parseCampaignIDFromResourceURI extracts the campaign ID from a URI of the form campaign://{campaign_id}/{resourceType}.
// It parses URIs of the expected format but requires an actual campaign ID and rejects the placeholder (campaign://_/{resourceType}).
// The resourceType parameter should be the resource suffix (e.g., "participants", "characters", "sessions").
func parseCampaignIDFromResourceURI(uri, resourceType string) (string, error) {
	prefix := "campaign://"
	suffix := "/" + resourceType

	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("URI must start with %q", prefix)
	}
	if !strings.HasSuffix(uri, suffix) {
		return "", fmt.Errorf("URI must end with %q", suffix)
	}

	campaignID := strings.TrimPrefix(uri, prefix)
	campaignID = strings.TrimSuffix(campaignID, suffix)
	campaignID = strings.TrimSpace(campaignID)

	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in URI")
	}

	// Reject the placeholder value - actual campaign IDs must be provided
	if campaignID == "_" {
		return "", fmt.Errorf("campaign ID placeholder '_' is not a valid campaign ID")
	}

	return campaignID, nil
}
