package domain

import (
	"fmt"
	"strings"
)

// parseCampaignIDFromResourceURI extracts the campaign ID from a URI of the form campaign://{campaign_id}/{resourceType}.
// It parses URIs of the expected format but requires an actual campaign ID.
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

// parseSessionIDFromResourceURI extracts the session ID from a URI of the form session://{session_id}/{resourceType}.
// It parses URIs of the expected format but requires an actual session ID.
// The resourceType parameter should be the resource suffix (e.g., "events").
func parseSessionIDFromResourceURI(uri, resourceType string) (string, error) {
	prefix := "session://"
	suffix := "/" + resourceType

	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("URI must start with %q", prefix)
	}
	if !strings.HasSuffix(uri, suffix) {
		return "", fmt.Errorf("URI must end with %q", suffix)
	}

	sessionID := strings.TrimPrefix(uri, prefix)
	sessionID = strings.TrimSuffix(sessionID, suffix)
	sessionID = strings.TrimSpace(sessionID)

	if sessionID == "" {
		return "", fmt.Errorf("session ID is required in URI")
	}

	if sessionID == "_" {
		return "", fmt.Errorf("session ID placeholder '_' is not a valid session ID")
	}

	return sessionID, nil
}
