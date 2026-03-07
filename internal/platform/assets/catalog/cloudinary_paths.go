package catalog

import (
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
)

//go:embed data/cloudinary_assets.high_fantasy.v1.json
var cloudinaryAssetsCatalogJSON []byte

var (
	loadCloudinaryAssetPathsOnce  sync.Once
	embeddedCloudinaryAssetPathsV map[string]string
)

type cloudinaryAssetsCatalogJSONDocument struct {
	CampaignScene []cloudinaryAssetJSON `json:"campaign_scene"`
	AvatarSet     []cloudinaryAssetJSON `json:"avatar_set"`
}

type cloudinaryAssetJSON struct {
	SetID      string              `json:"set_id"`
	FSAssetID  string              `json:"fs_asset_id"`
	Cloudinary cloudinaryAssetMeta `json:"cloudinary"`
}

type cloudinaryAssetMeta struct {
	PublicID string `json:"public_id"`
	Version  int64  `json:"version"`
}

// ResolveCDNAssetID returns the image delivery id for a set/asset selection.
//
// If an embedded cloudinary public_id exists for the set/asset pair, the
// mapped nested path is returned; otherwise the canonical asset id is returned.
func ResolveCDNAssetID(setID, assetID string) string {
	normalizedAssetID := strings.TrimSpace(assetID)
	if normalizedAssetID == "" {
		return ""
	}
	publicID, ok := CloudinaryPublicID(setID, normalizedAssetID)
	if ok {
		return publicID
	}
	return normalizedAssetID
}

// CloudinaryPublicID resolves one embedded cloudinary public_id.
func CloudinaryPublicID(setID, assetID string) (string, bool) {
	normalizedSetID := strings.TrimSpace(setID)
	normalizedAssetID := strings.TrimSpace(assetID)
	if normalizedSetID == "" || normalizedAssetID == "" {
		return "", false
	}
	publicID, ok := embeddedCloudinaryAssetPaths()[cloudinaryAssetPathLookupKey(normalizedSetID, normalizedAssetID)]
	if !ok {
		return "", false
	}
	return publicID, true
}

func embeddedCloudinaryAssetPaths() map[string]string {
	loadCloudinaryAssetPathsOnce.Do(func() {
		embeddedCloudinaryAssetPathsV = decodeCloudinaryAssetPaths(cloudinaryAssetsCatalogJSON)
	})
	if embeddedCloudinaryAssetPathsV == nil {
		return map[string]string{}
	}
	return embeddedCloudinaryAssetPathsV
}

func decodeCloudinaryAssetPaths(raw []byte) map[string]string {
	var payload cloudinaryAssetsCatalogJSONDocument
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]string{}
	}
	out := map[string]string{}
	appendAssets := func(entries []cloudinaryAssetJSON) {
		for _, entry := range entries {
			setID := strings.TrimSpace(entry.SetID)
			assetID := strings.TrimSpace(entry.FSAssetID)
			publicID := strings.TrimSpace(entry.Cloudinary.PublicID)
			if setID == "" || assetID == "" || publicID == "" {
				continue
			}
			if entry.Cloudinary.Version > 0 {
				publicID = "v" + strconv.FormatInt(entry.Cloudinary.Version, 10) + "/" + publicID
			}
			out[cloudinaryAssetPathLookupKey(setID, assetID)] = publicID
		}
	}
	appendAssets(payload.CampaignScene)
	appendAssets(payload.AvatarSet)
	return out
}

func cloudinaryAssetPathLookupKey(setID, assetID string) string {
	return setID + "\x00" + assetID
}
