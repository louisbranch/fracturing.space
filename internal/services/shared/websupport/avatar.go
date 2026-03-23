package websupport

import (
	"net/url"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
)

// AvatarImageURL resolves an avatar image URL for a given catalog selection.
func AvatarImageURL(assetBaseURL, role, entityID, avatarSetID, avatarAssetID string, deliveryWidthPX int) string {
	resolvedSetID, resolvedAssetID := ResolveWebAvatarSelection(role, entityID, avatarSetID, avatarAssetID)
	resolvedCDNAssetID := catalog.ResolveCDNAssetID(resolvedSetID, resolvedAssetID)
	portrait := ResolveWebAvatarPortrait(role, entityID, resolvedSetID)
	resolvedAssetURL, err := imagecdn.New(assetBaseURL).URL(imagecdn.Request{
		AssetID:   resolvedCDNAssetID,
		Extension: ".png",
		Crop: &imagecdn.Crop{
			X:        portrait.X,
			Y:        portrait.Y,
			WidthPX:  portrait.WidthPX,
			HeightPX: portrait.HeightPX,
		},
		Delivery: &imagecdn.Delivery{WidthPX: deliveryWidthPX},
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

	defaultSetID := defaultWebAvatarSetID(normalizedRole, avatarSetID, avatarAssetID)
	resolvedSetID, resolvedAssetID, err := catalog.AvatarManifest().ResolveSelection(catalog.SelectionInput{
		EntityType: normalizedRole,
		EntityID:   normalizedEntityID,
		SetID:      defaultSetID,
		AssetID:    avatarAssetID,
	})
	if err == nil {
		return resolvedSetID, resolvedAssetID
	}

	fallbackSetID, fallbackAssetID, fallbackErr := catalog.AvatarManifest().ResolveSelection(catalog.SelectionInput{
		EntityType: normalizedRole,
		EntityID:   normalizedEntityID,
		SetID:      defaultSetID,
		AssetID:    "",
	})
	if fallbackErr == nil {
		return fallbackSetID, fallbackAssetID
	}
	return catalog.AvatarSetBlankV1, defaultWebAvatarAssetID()
}

func defaultWebAvatarSetID(role, avatarSetID, avatarAssetID string) string {
	if strings.TrimSpace(avatarSetID) != "" || strings.TrimSpace(avatarAssetID) != "" {
		return avatarSetID
	}
	if strings.EqualFold(strings.TrimSpace(role), catalog.AvatarRoleUser) {
		return catalog.AvatarSetPeopleV1
	}
	return catalog.AvatarSetBlankV1
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
		fallbackSheet, fallbackOK := catalog.AvatarSheetBySetID(catalog.AvatarSetPeopleV1)
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
