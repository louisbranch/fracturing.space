package support

import (
	"net/url"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
)

const (
	avatarPortraitCardWidthPX = 192
)

// AvatarImageURL resolves an avatar image URL for a given catalog selection.
func AvatarImageURL(assetBaseURL, role, entityID, avatarSetID, avatarAssetID string) string {
	resolvedSetID, resolvedAssetID := ResolveWebAvatarSelection(role, entityID, avatarSetID, avatarAssetID)
	portrait := ResolveWebAvatarPortrait(role, entityID, resolvedSetID)
	resolvedAssetURL, err := imagecdn.New(assetBaseURL).URL(imagecdn.Request{
		AssetID:   resolvedAssetID,
		Extension: ".png",
		Crop: &imagecdn.Crop{
			X:        portrait.X,
			Y:        portrait.Y,
			WidthPX:  portrait.WidthPX,
			HeightPX: portrait.HeightPX,
		},
		Delivery: &imagecdn.Delivery{
			WidthPX: avatarPortraitCardWidthPX,
		},
	})
	if err == nil {
		return resolvedAssetURL
	}
	return "/static/avatars/" + url.PathEscape(resolvedAssetID) + ".png"
}

// ResolveWebAvatarSelection returns stable avatar set and asset selectors.
func ResolveWebAvatarSelection(role, entityID, avatarSetID, avatarAssetID string) (string, string) {
	normalizedRole := strings.TrimSpace(role)
	if normalizedRole == "" {
		normalizedRole = catalog.AvatarRoleUser
	}
	normalizedEntityID := strings.TrimSpace(entityID)
	if normalizedEntityID == "" {
		normalizedEntityID = "default"
	}

	resolvedSetID, resolvedAssetID, err := catalog.AvatarManifest().ResolveSelection(catalog.SelectionInput{
		EntityType: normalizedRole,
		EntityID:   normalizedEntityID,
		SetID:      avatarSetID,
		AssetID:    avatarAssetID,
	})
	if err == nil {
		return resolvedSetID, resolvedAssetID
	}

	fallbackSetID, fallbackAssetID, fallbackErr := catalog.AvatarManifest().ResolveSelection(catalog.SelectionInput{
		EntityType: normalizedRole,
		EntityID:   normalizedEntityID,
		SetID:      "",
		AssetID:    "",
	})
	if fallbackErr == nil {
		return fallbackSetID, fallbackAssetID
	}
	return catalog.AvatarSetBlankV1, defaultWebAvatarAssetID()
}

func defaultWebAvatarAssetID() string {
	manifest := catalog.AvatarManifest()
	avatarSet, ok := manifest.Sets[catalog.AvatarSetBlankV1]
	if ok && len(avatarSet.AssetIDs) > 0 {
		return avatarSet.AssetIDs[0]
	}
	defaultSetID, defaultSetOK := manifest.NormalizeSetID("")
	if defaultSetOK {
		defaultSet, setOK := manifest.Sets[defaultSetID]
		if setOK && len(defaultSet.AssetIDs) > 0 {
			return defaultSet.AssetIDs[0]
		}
	}
	return catalog.AvatarAssetBlank
}

// ResolveWebAvatarPortrait resolves the avatar portrait slot for a selection.
func ResolveWebAvatarPortrait(role, entityID, setID string) catalog.AvatarPortrait {
	normalizedRole := strings.TrimSpace(role)
	if normalizedRole == "" {
		normalizedRole = catalog.AvatarRoleUser
	}
	slot, err := catalog.ResolveAvatarPortraitSlot(normalizedRole, strings.TrimSpace(entityID))
	if err != nil {
		if normalizedRole == catalog.AvatarRoleCharacter {
			slot = 2
		} else {
			slot = 1
		}
	}

	sheet, ok := catalog.AvatarSheetBySetID(setID)
	if !ok {
		fallbackSheet, fallbackOK := catalog.AvatarSheetBySetID(catalog.AvatarSetV1)
		if !fallbackOK {
			return catalog.AvatarPortrait{Slot: slot}
		}
		sheet = fallbackSheet
	}

	portrait, ok := sheet.Portraits[slot]
	if ok {
		return portrait
	}
	firstPortrait, ok := sheet.Portraits[1]
	if ok {
		return firstPortrait
	}
	return catalog.AvatarPortrait{Slot: slot}
}
