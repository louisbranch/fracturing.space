package game

import (
	"context"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
)

// socialProfileSnapshot carries the profile fields create flows can safely
// snapshot without coupling participant payloads to the social transport type.
type socialProfileSnapshot struct {
	Name          string
	Pronouns      string
	AvatarSetID   string
	AvatarAssetID string
}

// loadSocialProfileSnapshot returns best-effort profile data for user-linked
// create flows so callers can fill missing fields while keeping request values authoritative.
func loadSocialProfileSnapshot(ctx context.Context, socialClient socialv1.SocialServiceClient, userID string) socialProfileSnapshot {
	userID = strings.TrimSpace(userID)
	if userID == "" || socialClient == nil {
		return socialProfileSnapshot{}
	}

	resp, err := socialClient.GetUserProfile(ctx, &socialv1.GetUserProfileRequest{UserId: userID})
	if err != nil || resp == nil || resp.GetUserProfile() == nil {
		return socialProfileSnapshot{}
	}

	profile := resp.GetUserProfile()
	return socialProfileSnapshot{
		Name:          strings.TrimSpace(profile.GetName()),
		Pronouns:      sharedpronouns.FromProto(profile.GetPronouns()),
		AvatarSetID:   strings.TrimSpace(profile.GetAvatarSetId()),
		AvatarAssetID: strings.TrimSpace(profile.GetAvatarAssetId()),
	}
}
