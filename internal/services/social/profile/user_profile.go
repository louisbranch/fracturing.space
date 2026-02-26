// Package profile validates and normalizes user profile inputs.
package profile

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
)

const (
	maxNameLength          = 64
	maxBioLength           = 280
	defaultUserAvatarSetID = assetcatalog.AvatarSetPeopleV1
)

// Normalized stores validated profile field values.
type Normalized struct {
	Name          string
	AvatarSetID   string
	AvatarAssetID string
	Bio           string
	Pronouns      string
}

var userProfileAvatarManifest = assetcatalog.AvatarManifest()

// Normalize validates and trims user-supplied user profile values.
func Normalize(userID string, name string, avatarSetID string, avatarAssetID string, bio string, pronouns string) (Normalized, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Normalized{}, fmt.Errorf("user id is required")
	}

	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > maxNameLength {
		return Normalized{}, fmt.Errorf("name must be at most %d characters", maxNameLength)
	}

	avatarSetID = strings.TrimSpace(avatarSetID)
	avatarAssetID = strings.TrimSpace(avatarAssetID)
	if avatarSetID == "" && avatarAssetID != "" {
		return Normalized{}, fmt.Errorf("avatar set is required when avatar asset is provided")
	}
	if avatarSetID == "" {
		avatarSetID = defaultUserAvatarSetID
	}
	resolvedSetID, resolvedAssetID, err := userProfileAvatarManifest.ResolveSelection(assetcatalog.SelectionInput{
		EntityType: assetcatalog.AvatarRoleUser,
		EntityID:   userID,
		SetID:      avatarSetID,
		AssetID:    avatarAssetID,
	})
	switch {
	case err == nil:
		avatarSetID = resolvedSetID
		avatarAssetID = resolvedAssetID
	case errors.Is(err, assetcatalog.ErrSetNotFound):
		return Normalized{}, fmt.Errorf("avatar set is invalid")
	case errors.Is(err, assetcatalog.ErrAssetInvalid):
		return Normalized{}, fmt.Errorf("avatar asset is invalid")
	default:
		return Normalized{}, err
	}

	bio = strings.TrimSpace(bio)
	if utf8.RuneCountInString(bio) > maxBioLength {
		return Normalized{}, fmt.Errorf("bio must be at most %d characters", maxBioLength)
	}
	pronouns = strings.TrimSpace(pronouns)

	return Normalized{
		Name:          name,
		AvatarSetID:   avatarSetID,
		AvatarAssetID: avatarAssetID,
		Bio:           bio,
		Pronouns:      pronouns,
	}, nil
}
