package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/campaign_covers.v1.json
var campaignCoverManifestJSON []byte

//go:embed data/avatars.v1.json
var avatarCatalogJSON []byte

var (
	campaignCoverManifestData = mustDecodeManifestJSON(campaignCoverManifestJSON, "campaign cover manifest")
	avatarManifestData        = mustDecodeManifestJSON(avatarManifestJSONFromCatalog(avatarCatalogJSON), "avatar manifest")
	avatarSheetsDataBySetID   = mustDecodeAvatarSheetsJSON(avatarCatalogJSON)
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

func mustDecodeManifestJSON(raw []byte, source string) Manifest {
	manifest, err := decodeManifestJSON(raw)
	if err != nil {
		panic(fmt.Sprintf("decode %s: %v", source, err))
	}
	return manifest
}

func decodeManifestJSON(raw []byte) (Manifest, error) {
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

func mustDecodeAvatarSheetsJSON(raw []byte) map[string]AvatarSheet {
	sheets, err := decodeAvatarSheetsJSON(raw)
	if err != nil {
		panic(fmt.Sprintf("decode avatar sheets: %v", err))
	}
	return sheets
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

func avatarManifestJSONFromCatalog(raw []byte) []byte {
	var payload avatarCatalogJSONDocument
	if err := json.Unmarshal(raw, &payload); err != nil {
		panic(fmt.Sprintf("decode avatar catalog manifest: %v", err))
	}
	manifestJSONPayload, err := json.Marshal(payload.Manifest)
	if err != nil {
		panic(fmt.Sprintf("marshal avatar manifest payload: %v", err))
	}
	return manifestJSONPayload
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
