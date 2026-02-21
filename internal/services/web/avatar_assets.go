package web

import (
	"net/url"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
)

const (
	avatarPortraitCardWidthPX = 192
)

var webAvatarManifest = catalog.AvatarManifest()

func avatarImageURL(config Config, role, entityID, avatarSetID, avatarAssetID string) string {
	resolvedSetID, resolvedAssetID := resolveWebAvatarSelection(role, entityID, avatarSetID, avatarAssetID)
	portrait := resolveWebAvatarPortrait(role, entityID, resolvedSetID)
	resolvedAssetURL, err := imagecdn.New(config.AssetBaseURL).URL(imagecdn.Request{
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

func resolveWebAvatarSelection(role, entityID, avatarSetID, avatarAssetID string) (string, string) {
	normalizedRole := strings.TrimSpace(role)
	if normalizedRole == "" {
		normalizedRole = catalog.AvatarRoleUser
	}
	normalizedEntityID := strings.TrimSpace(entityID)
	if normalizedEntityID == "" {
		normalizedEntityID = "default"
	}

	resolvedSetID, resolvedAssetID, err := webAvatarManifest.ResolveSelection(catalog.SelectionInput{
		EntityType: normalizedRole,
		EntityID:   normalizedEntityID,
		SetID:      avatarSetID,
		AssetID:    avatarAssetID,
	})
	if err == nil {
		return resolvedSetID, resolvedAssetID
	}

	fallbackSetID, fallbackAssetID, fallbackErr := webAvatarManifest.ResolveSelection(catalog.SelectionInput{
		EntityType: normalizedRole,
		EntityID:   normalizedEntityID,
		SetID:      catalog.AvatarSetV1,
		AssetID:    "",
	})
	if fallbackErr == nil {
		return fallbackSetID, fallbackAssetID
	}
	return catalog.AvatarSetV1, defaultWebAvatarAssetID()
}

func defaultWebAvatarAssetID() string {
	avatarSet, ok := webAvatarManifest.Sets[catalog.AvatarSetV1]
	if !ok || len(avatarSet.AssetIDs) == 0 {
		return ""
	}
	return avatarSet.AssetIDs[0]
}

func resolveWebAvatarPortrait(role, entityID, setID string) catalog.AvatarPortrait {
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
