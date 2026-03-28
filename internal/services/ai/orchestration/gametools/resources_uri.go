package gametools

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parseArtifactURI(uri string) (string, string, error) {
	if !strings.HasPrefix(uri, "campaign://") {
		return "", "", fmt.Errorf("URI must start with \"campaign://\"")
	}
	rest := strings.TrimPrefix(uri, "campaign://")
	parts := strings.SplitN(rest, "/artifacts/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("URI must match campaign://{campaign_id}/artifacts/{path}")
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("campaign and artifact path are required")
	}
	return parts[0], parts[1], nil
}

func parseCampaignIDFromSuffixURI(uri, suffix string) (string, error) {
	prefix := "campaign://"
	fullSuffix := "/" + suffix
	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("URI must start with %q", prefix)
	}
	if !strings.HasSuffix(uri, fullSuffix) {
		return "", fmt.Errorf("URI must end with %q", fullSuffix)
	}
	campaignID := strings.TrimSuffix(strings.TrimPrefix(uri, prefix), fullSuffix)
	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in URI")
	}
	return campaignID, nil
}

func parseSceneListURI(uri string) (string, string, error) {
	if !strings.HasPrefix(uri, "campaign://") {
		return "", "", fmt.Errorf("URI must start with \"campaign://\"")
	}
	rest := strings.TrimPrefix(uri, "campaign://")
	parts := strings.Split(rest, "/")
	if len(parts) != 4 || parts[1] != "sessions" || parts[3] != "scenes" {
		return "", "", fmt.Errorf("URI must match campaign://{campaign_id}/sessions/{session_id}/scenes")
	}
	if parts[0] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("campaign and session IDs are required in URI")
	}
	return parts[0], parts[2], nil
}

func marshalIndent(v any) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
