package daggerhearttools

import (
	"context"
	"fmt"
	"strings"
)

// ReadResource handles the Daggerheart-specific subset of AI resource URIs.
// The generic gametools package delegates here when one of those URIs is
// encountered, and keeps ownership of non-system-specific dispatch.
func ReadResource(runtime Runtime, ctx context.Context, uri string) (string, bool, error) {
	switch {
	case uri == "daggerheart://rules/version":
		value, err := ReadRulesVersionResource(runtime, ctx)
		return value, true, err

	case strings.HasPrefix(uri, "daggerheart://campaign/") && strings.HasSuffix(uri, "/snapshot"):
		campaignID, err := parseSnapshotResourceURI(uri)
		if err != nil {
			return "", true, err
		}
		value, err := ReadSnapshotResource(runtime, ctx, campaignID)
		return value, true, err

	case strings.HasPrefix(uri, "daggerheart://campaign/") && strings.Contains(uri, "/sessions/") && strings.HasSuffix(uri, "/combat_board"):
		campaignID, sessionID, err := parseCombatBoardResourceURI(uri)
		if err != nil {
			return "", true, err
		}
		value, err := ReadCombatBoardResource(runtime, ctx, campaignID, sessionID)
		return value, true, err

	case strings.HasPrefix(uri, "daggerheart://campaign/") && strings.HasSuffix(uri, "/campaign_countdowns"):
		campaignID, err := parseCampaignCountdownsResourceURI(uri)
		if err != nil {
			return "", true, err
		}
		value, err := ReadCampaignCountdownsResource(runtime, ctx, campaignID)
		return value, true, err

	case strings.HasPrefix(uri, "campaign://") && strings.Contains(uri, "/characters/") && strings.HasSuffix(uri, "/sheet"):
		campaignID, characterID, err := parseCharacterSheetResourceURI(uri)
		if err != nil {
			return "", true, err
		}
		value, err := ReadCharacterSheetResource(runtime, ctx, campaignID, characterID)
		return value, true, err

	default:
		return "", false, nil
	}
}

func parseSnapshotResourceURI(uri string) (string, error) {
	campaignID := strings.TrimPrefix(uri, "daggerheart://campaign/")
	campaignID = strings.TrimSuffix(campaignID, "/snapshot")
	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in snapshot URI")
	}
	return campaignID, nil
}

func parseCombatBoardResourceURI(uri string) (string, string, error) {
	if !strings.HasPrefix(uri, "daggerheart://campaign/") {
		return "", "", fmt.Errorf("URI must start with \"daggerheart://campaign/\"")
	}
	rest := strings.TrimPrefix(uri, "daggerheart://campaign/")
	parts := strings.Split(rest, "/")
	if len(parts) != 4 || parts[1] != "sessions" || parts[3] != "combat_board" {
		return "", "", fmt.Errorf("URI must match daggerheart://campaign/{campaign_id}/sessions/{session_id}/combat_board")
	}
	if parts[0] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("campaign and session IDs are required in URI")
	}
	return parts[0], parts[2], nil
}

func parseCampaignCountdownsResourceURI(uri string) (string, error) {
	if !strings.HasPrefix(uri, "daggerheart://campaign/") {
		return "", fmt.Errorf("URI must start with \"daggerheart://campaign/\"")
	}
	rest := strings.TrimPrefix(uri, "daggerheart://campaign/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "campaign_countdowns" {
		return "", fmt.Errorf("URI must match daggerheart://campaign/{campaign_id}/campaign_countdowns")
	}
	if parts[0] == "" {
		return "", fmt.Errorf("campaign ID is required in URI")
	}
	return parts[0], nil
}

func parseCharacterSheetResourceURI(uri string) (string, string, error) {
	if !strings.HasPrefix(uri, "campaign://") {
		return "", "", fmt.Errorf("URI must start with \"campaign://\"")
	}
	rest := strings.TrimPrefix(uri, "campaign://")
	parts := strings.Split(rest, "/")
	if len(parts) != 4 || parts[1] != "characters" || parts[3] != "sheet" {
		return "", "", fmt.Errorf("URI must match campaign://{campaign_id}/characters/{character_id}/sheet")
	}
	if parts[0] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("campaign and character IDs are required in URI")
	}
	return parts[0], parts[2], nil
}
