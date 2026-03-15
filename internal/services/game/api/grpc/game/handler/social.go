package handler

import (
	"context"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
)

// SocialProfileSnapshot carries the profile fields create flows can safely
// snapshot without coupling participant payloads to the social transport type.
type SocialProfileSnapshot struct {
	Name          string
	Pronouns      string
	AvatarSetID   string
	AvatarAssetID string
}

// LoadSocialProfileSnapshot returns best-effort profile data for user-linked
// create flows so callers can fill missing fields while keeping request values authoritative.
func LoadSocialProfileSnapshot(ctx context.Context, socialClient socialv1.SocialServiceClient, userID string) SocialProfileSnapshot {
	userID = strings.TrimSpace(userID)
	if userID == "" || socialClient == nil {
		return SocialProfileSnapshot{}
	}

	resp, err := socialClient.GetUserProfile(ctx, &socialv1.GetUserProfileRequest{UserId: userID})
	if err != nil || resp == nil || resp.GetUserProfile() == nil {
		return SocialProfileSnapshot{}
	}

	profile := resp.GetUserProfile()
	return SocialProfileSnapshot{
		Name:          strings.TrimSpace(profile.GetName()),
		Pronouns:      sharedpronouns.FromProto(profile.GetPronouns()),
		AvatarSetID:   strings.TrimSpace(profile.GetAvatarSetId()),
		AvatarAssetID: strings.TrimSpace(profile.GetAvatarAssetId()),
	}
}
