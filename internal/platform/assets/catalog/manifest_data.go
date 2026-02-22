package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed data/campaign_covers.v1.json
var campaignCoverManifestJSON []byte

//go:embed data/avatars.v1.json
var avatarCatalogJSON []byte

var (
	loadCatalogManifestsOnce      sync.Once
	embeddedCampaignCoverManifest Manifest
	embeddedAvatarManifest        Manifest
	embeddedAvatarSheetsBySetID   map[string]AvatarSheet
	catalogLoadError              error
)

type manifestJSON struct {
	ID           string            `json:"id"`
	DefaultSet   string            `json:"default_set"`
	Sets         []setJSON         `json:"sets"`
	SetAliases   map[string]string `json:"set_aliases"`
	AssetAliases map[string]string `json:"asset_aliases"`
}

type setJSON struct {
	ID       string   `json:"id"`
	AssetIDs []string `json:"asset_ids"`
}

type avatarCatalogJSONDocument struct {
	Manifest manifestJSON      `json:"manifest"`
	Sheets   []avatarSheetJSON `json:"sheets"`
}

type avatarSheetJSON struct {
	SetID     string               `json:"set_id"`
	WidthPX   int                  `json:"width_px"`
	HeightPX  int                  `json:"height_px"`
	Portraits []avatarPortraitJSON `json:"portraits"`
}

type avatarPortraitJSON struct {
	Slot     int `json:"slot"`
	X        int `json:"x"`
	Y        int `json:"y"`
	WidthPX  int `json:"width_px"`
	HeightPX int `json:"height_px"`
}

// ValidateEmbeddedCatalogManifests returns any manifest parsing error from the embedded bundle.
func ValidateEmbeddedCatalogManifests() error {
	_, _, _, err := EmbeddedCatalogManifests()
	return err
}

func loadCampaignAssetsManifests() (Manifest, Manifest, map[string]AvatarSheet, error) {
	loadCatalogManifestsOnce.Do(func() {
		embeddedCampaignCoverManifest, embeddedAvatarManifest, embeddedAvatarSheetsBySetID, catalogLoadError = loadEmbeddedCatalogManifests()
	})
	if catalogLoadError != nil {
		return Manifest{}, Manifest{}, map[string]AvatarSheet{}, catalogLoadError
	}
	return copyManifest(embeddedCampaignCoverManifest),
		copyManifest(embeddedAvatarManifest),
		copyAvatarSheetMap(embeddedAvatarSheetsBySetID),
		nil
}

// EmbeddedCatalogManifests returns decoded, immutable embedded catalog data.
//
// It validates embedded JSON once and returns fresh copies so callers cannot
// mutate cached package state.
func EmbeddedCatalogManifests() (Manifest, Manifest, map[string]AvatarSheet, error) {
	return loadCampaignAssetsManifests()
}

func loadEmbeddedCatalogManifests() (Manifest, Manifest, map[string]AvatarSheet, error) {
	campaignManifest, err := decodeManifestJSON(campaignCoverManifestJSON)
	if err != nil {
		return Manifest{}, Manifest{}, nil, fmt.Errorf("decode campaign cover manifest: %w", err)
	}

	avatarManifest, err := decodeManifestJSONFromCatalog(avatarCatalogJSON)
	if err != nil {
		return Manifest{}, Manifest{}, nil, fmt.Errorf("decode avatar manifest: %w", err)
	}

	avatarSheets, err := decodeAvatarSheetsJSON(avatarCatalogJSON)
	if err != nil {
		return Manifest{}, Manifest{}, nil, fmt.Errorf("decode avatar sheets: %w", err)
	}
	return campaignManifest, avatarManifest, avatarSheets, nil
}

func decodeManifestJSON(raw []byte) (Manifest, error) {
	manifest, err := decodeManifest(raw)
	if err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func decodeManifest(raw []byte) (Manifest, error) {
	var payload manifestJSON
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Manifest{}, err
	}
	manifestID := strings.TrimSpace(payload.ID)
	defaultSetID := strings.TrimSpace(payload.DefaultSet)
	if manifestID == "" || defaultSetID == "" {
		return Manifest{}, fmt.Errorf("manifest id/default set are required")
	}

	sets := make(map[string]Set, len(payload.Sets))
	for _, rawSet := range payload.Sets {
		setID := strings.TrimSpace(rawSet.ID)
		if setID == "" {
			continue
		}
		if _, exists := sets[setID]; exists {
			return Manifest{}, fmt.Errorf("duplicate set id %q", setID)
		}
		sets[setID] = Set{
			ID:       setID,
			AssetIDs: normalizeStringList(rawSet.AssetIDs),
		}
	}
	if _, ok := sets[defaultSetID]; !ok {
		return Manifest{}, fmt.Errorf("default set %q is missing", defaultSetID)
	}

	return Manifest{
		ID:           manifestID,
		DefaultSet:   defaultSetID,
		Sets:         sets,
		SetAliases:   copyStringMap(payload.SetAliases),
		AssetAliases: copyStringMap(payload.AssetAliases),
	}, nil
}

func decodeManifestJSONFromCatalog(raw []byte) (Manifest, error) {
	manifestPayload, err := avatarManifestJSONFromCatalog(raw)
	if err != nil {
		return Manifest{}, err
	}
	return decodeManifest(manifestPayload)
}

func decodeAvatarSheetsJSON(raw []byte) (map[string]AvatarSheet, error) {
	var payload avatarCatalogJSONDocument
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	sheetsBySetID := make(map[string]AvatarSheet, len(payload.Sheets))
	for _, rawSheet := range payload.Sheets {
		setID := strings.TrimSpace(rawSheet.SetID)
		if setID == "" {
			continue
		}
		portraits := map[int]AvatarPortrait{}
		for _, rawPortrait := range rawSheet.Portraits {
			if rawPortrait.Slot <= 0 {
				continue
			}
			portraits[rawPortrait.Slot] = AvatarPortrait{
				Slot:     rawPortrait.Slot,
				X:        rawPortrait.X,
				Y:        rawPortrait.Y,
				WidthPX:  rawPortrait.WidthPX,
				HeightPX: rawPortrait.HeightPX,
			}
		}
		sheetsBySetID[setID] = AvatarSheet{
			WidthPX:   rawSheet.WidthPX,
			HeightPX:  rawSheet.HeightPX,
			Portraits: portraits,
		}
	}
	return sheetsBySetID, nil
}

func copyAvatarSheetMap(source map[string]AvatarSheet) map[string]AvatarSheet {
	if len(source) == 0 {
		return map[string]AvatarSheet{}
	}
	out := make(map[string]AvatarSheet, len(source))
	for setID, sheet := range source {
		out[strings.TrimSpace(setID)] = copyAvatarSheet(sheet)
	}
	return out
}

func avatarManifestJSONFromCatalog(raw []byte) ([]byte, error) {
	var payload avatarCatalogJSONDocument
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode avatar catalog: %w", err)
	}
	manifestJSONPayload, err := json.Marshal(payload.Manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal avatar manifest payload: %w", err)
	}
	return manifestJSONPayload, nil
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func copyStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(source))
	for key, value := range source {
		normalizedKey := strings.TrimSpace(key)
		normalizedValue := strings.TrimSpace(value)
		if normalizedKey == "" || normalizedValue == "" {
			continue
		}
		out[normalizedKey] = normalizedValue
	}
	return out
}

func copyManifest(source Manifest) Manifest {
	out := Manifest{
		ID:           source.ID,
		DefaultSet:   source.DefaultSet,
		Sets:         map[string]Set{},
		SetAliases:   copyStringMap(source.SetAliases),
		AssetAliases: copyStringMap(source.AssetAliases),
	}
	for setID, set := range source.Sets {
		out.Sets[setID] = Set{
			ID:       set.ID,
			AssetIDs: append([]string(nil), set.AssetIDs...),
		}
	}
	return out
}
