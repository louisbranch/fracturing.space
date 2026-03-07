package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

const (
	// DaggerheartAssetTypeClassIllustration maps to class hero artwork.
	DaggerheartAssetTypeClassIllustration = "daggerheart_class_illustration"
	// DaggerheartAssetTypeClassIcon maps to class iconography.
	DaggerheartAssetTypeClassIcon = "daggerheart_class_icon"
	// DaggerheartAssetTypeSubclassIllustration maps to subclass artwork.
	DaggerheartAssetTypeSubclassIllustration = "daggerheart_subclass_illustration"
	// DaggerheartAssetTypeAncestryIllustration maps to ancestry artwork.
	DaggerheartAssetTypeAncestryIllustration = "daggerheart_ancestry_illustration"
	// DaggerheartAssetTypeCommunityIllustration maps to community artwork.
	DaggerheartAssetTypeCommunityIllustration = "daggerheart_community_illustration"
	// DaggerheartAssetTypeDomainIllustration maps to domain artwork.
	DaggerheartAssetTypeDomainIllustration = "daggerheart_domain_illustration"
	// DaggerheartAssetTypeDomainIcon maps to domain iconography.
	DaggerheartAssetTypeDomainIcon = "daggerheart_domain_icon"
	// DaggerheartAssetTypeDomainCardIllustration maps to domain-card artwork.
	DaggerheartAssetTypeDomainCardIllustration = "daggerheart_domain_card_illustration"
	// DaggerheartAssetTypeAdversaryIllustration maps to adversary artwork.
	DaggerheartAssetTypeAdversaryIllustration = "daggerheart_adversary_illustration"
	// DaggerheartAssetTypeEnvironmentIllustration maps to environment artwork.
	DaggerheartAssetTypeEnvironmentIllustration = "daggerheart_environment_illustration"
	// DaggerheartAssetTypeWeaponIllustration maps to weapon artwork.
	DaggerheartAssetTypeWeaponIllustration = "daggerheart_weapon_illustration"
	// DaggerheartAssetTypeArmorIllustration maps to armor artwork.
	DaggerheartAssetTypeArmorIllustration = "daggerheart_armor_illustration"
	// DaggerheartAssetTypeItemIllustration maps to item artwork.
	DaggerheartAssetTypeItemIllustration = "daggerheart_item_illustration"
)

const (
	// DaggerheartEntityTypeClass is a class catalog entity.
	DaggerheartEntityTypeClass = "class"
	// DaggerheartEntityTypeSubclass is a subclass catalog entity.
	DaggerheartEntityTypeSubclass = "subclass"
	// DaggerheartEntityTypeAncestry is an ancestry catalog entity.
	DaggerheartEntityTypeAncestry = "ancestry"
	// DaggerheartEntityTypeCommunity is a community catalog entity.
	DaggerheartEntityTypeCommunity = "community"
	// DaggerheartEntityTypeDomain is a domain catalog entity.
	DaggerheartEntityTypeDomain = "domain"
	// DaggerheartEntityTypeDomainCard is a domain-card catalog entity.
	DaggerheartEntityTypeDomainCard = "domain_card"
	// DaggerheartEntityTypeAdversary is an adversary catalog entity.
	DaggerheartEntityTypeAdversary = "adversary"
	// DaggerheartEntityTypeEnvironment is an environment catalog entity.
	DaggerheartEntityTypeEnvironment = "environment"
	// DaggerheartEntityTypeWeapon is a weapon catalog entity.
	DaggerheartEntityTypeWeapon = "weapon"
	// DaggerheartEntityTypeArmor is an armor catalog entity.
	DaggerheartEntityTypeArmor = "armor"
	// DaggerheartEntityTypeItem is an item catalog entity.
	DaggerheartEntityTypeItem = "item"
)

// DaggerheartAssetResolutionStatus reports how a catalog image selector was resolved.
type DaggerheartAssetResolutionStatus string

const (
	// DaggerheartAssetResolutionStatusMapped means an explicit entity->asset binding was used.
	DaggerheartAssetResolutionStatusMapped DaggerheartAssetResolutionStatus = "mapped"
	// DaggerheartAssetResolutionStatusSetDefault means a deterministic set-level fallback was used.
	DaggerheartAssetResolutionStatusSetDefault DaggerheartAssetResolutionStatus = "set_default"
	// DaggerheartAssetResolutionStatusUnavailable means no deliverable mapping is currently available.
	DaggerheartAssetResolutionStatusUnavailable DaggerheartAssetResolutionStatus = "unavailable"
)

// DaggerheartResolvedAsset is one resolved image selection for a content entity.
type DaggerheartResolvedAsset struct {
	EntityType string
	EntityID   string
	AssetType  string
	SetID      string
	AssetID    string
	CDNAssetID string
	Status     DaggerheartAssetResolutionStatus
}

// DaggerheartAssetSet defines one asset set for a Daggerheart content asset family.
type DaggerheartAssetSet struct {
	ID        string
	AssetType string
	AssetIDs  []string
}

// DaggerheartEntityAsset defines one explicit entity -> set/asset mapping.
type DaggerheartEntityAsset struct {
	EntityType string
	EntityID   string
	AssetType  string
	SetID      string
	AssetID    string
}

// DaggerheartAssetManifest defines the embedded Daggerheart content-image manifest.
type DaggerheartAssetManifest struct {
	ID            string
	SystemID      string
	SystemVersion string
	Locale        string
	Theme         string

	setsByID                map[string]DaggerheartAssetSet
	defaultSetIDByAssetType map[string]string
	entityAssets            map[string]DaggerheartEntityAsset
}

type cloudinaryAssetLookupFn func(setID, assetID string) (string, bool)

type daggerheartAssetManifestJSONDocument struct {
	ID             string                          `json:"id"`
	SystemID       string                          `json:"system_id"`
	SystemVersion  string                          `json:"system_version"`
	Locale         string                          `json:"locale"`
	Theme          string                          `json:"theme"`
	Sets           []daggerheartAssetSetJSON       `json:"sets"`
	EntityAssetMap []daggerheartEntityAssetMapJSON `json:"entity_asset_map"`
}

type daggerheartAssetSetJSON struct {
	ID        string   `json:"id"`
	AssetType string   `json:"asset_type"`
	AssetIDs  []string `json:"asset_ids"`
}

type daggerheartEntityAssetMapJSON struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	AssetType  string `json:"asset_type"`
	SetID      string `json:"set_id"`
	AssetID    string `json:"asset_id"`
}

//go:embed data/daggerheart_assets.v1.json
var daggerheartAssetManifestJSON []byte

var (
	loadDaggerheartAssetManifestOnce sync.Once
	embeddedDaggerheartAssetManifest DaggerheartAssetManifest
	daggerheartAssetManifestLoadErr  error
)

// DaggerheartAssetsManifest returns the canonical built-in Daggerheart asset manifest.
func DaggerheartAssetsManifest() DaggerheartAssetManifest {
	manifest, err := EmbeddedDaggerheartAssetManifest()
	if err != nil {
		return DaggerheartAssetManifest{}
	}
	return manifest
}

// EmbeddedDaggerheartAssetManifest returns decoded embedded Daggerheart asset data.
//
// It validates embedded JSON once and returns a fresh copy so callers cannot
// mutate cached package state.
func EmbeddedDaggerheartAssetManifest() (DaggerheartAssetManifest, error) {
	loadDaggerheartAssetManifestOnce.Do(func() {
		embeddedDaggerheartAssetManifest, daggerheartAssetManifestLoadErr = decodeDaggerheartAssetManifest(daggerheartAssetManifestJSON)
	})
	if daggerheartAssetManifestLoadErr != nil {
		return DaggerheartAssetManifest{}, daggerheartAssetManifestLoadErr
	}
	return copyDaggerheartAssetManifest(embeddedDaggerheartAssetManifest), nil
}

// ResolveEntityAsset resolves one content image reference with deterministic fallback behavior.
func (m DaggerheartAssetManifest) ResolveEntityAsset(entityType, entityID, assetType string) DaggerheartResolvedAsset {
	return m.resolveEntityAsset(entityType, entityID, assetType, CloudinaryPublicID)
}

func (m DaggerheartAssetManifest) resolveEntityAsset(entityType, entityID, assetType string, lookup cloudinaryAssetLookupFn) DaggerheartResolvedAsset {
	normalizedEntityType := strings.ToLower(strings.TrimSpace(entityType))
	normalizedEntityID := strings.TrimSpace(entityID)
	normalizedAssetType := strings.ToLower(strings.TrimSpace(assetType))
	result := DaggerheartResolvedAsset{
		EntityType: normalizedEntityType,
		EntityID:   normalizedEntityID,
		AssetType:  normalizedAssetType,
		Status:     DaggerheartAssetResolutionStatusUnavailable,
	}
	if normalizedEntityType == "" || normalizedEntityID == "" || normalizedAssetType == "" || lookup == nil {
		return result
	}

	defaultSet, ok := m.defaultSetForAssetType(normalizedAssetType)
	if !ok {
		return result
	}
	result.SetID = defaultSet.ID

	mapped, hasMapped := m.entityAssets[daggerheartEntityAssetLookupKey(normalizedEntityType, normalizedEntityID, normalizedAssetType)]
	if hasMapped {
		mappedSetID := strings.TrimSpace(mapped.SetID)
		if mappedSetID == "" {
			mappedSetID = defaultSet.ID
		}
		if mappedSet, ok := m.setForIDAndAssetType(mappedSetID, normalizedAssetType); ok {
			mappedAssetID := strings.TrimSpace(mapped.AssetID)
			result.SetID = mappedSet.ID
			result.AssetID = mappedAssetID
			if mappedAssetID != "" && assetIDInSet(mappedSet, mappedAssetID) {
				if cdnAssetID, found := lookup(mappedSet.ID, mappedAssetID); found {
					result.CDNAssetID = cdnAssetID
					result.Status = DaggerheartAssetResolutionStatusMapped
					return result
				}
			}

			fallback := m.resolveSetDefault(normalizedEntityType, normalizedEntityID, normalizedAssetType, mappedSet.ID, lookup)
			if fallback.Status == DaggerheartAssetResolutionStatusSetDefault {
				return fallback
			}
			return result
		}
	}

	fallback := m.resolveSetDefault(normalizedEntityType, normalizedEntityID, normalizedAssetType, defaultSet.ID, lookup)
	if fallback.Status == DaggerheartAssetResolutionStatusSetDefault {
		return fallback
	}
	return result
}

func (m DaggerheartAssetManifest) resolveSetDefault(entityType, entityID, assetType, setID string, lookup cloudinaryAssetLookupFn) DaggerheartResolvedAsset {
	resolved := DaggerheartResolvedAsset{
		EntityType: entityType,
		EntityID:   entityID,
		AssetType:  assetType,
		SetID:      strings.TrimSpace(setID),
		Status:     DaggerheartAssetResolutionStatusUnavailable,
	}

	set, ok := m.setForIDAndAssetType(setID, assetType)
	if !ok {
		return resolved
	}
	resolved.SetID = set.ID

	deliverableAssetIDs := make([]string, 0, len(set.AssetIDs))
	for _, candidate := range set.AssetIDs {
		assetID := strings.TrimSpace(candidate)
		if assetID == "" {
			continue
		}
		if _, found := lookup(set.ID, assetID); !found {
			continue
		}
		deliverableAssetIDs = append(deliverableAssetIDs, assetID)
	}
	if len(deliverableAssetIDs) == 0 {
		return resolved
	}

	defaultAssetID, err := Manifest{
		ID:         m.ID,
		DefaultSet: set.ID,
		Sets: map[string]Set{
			set.ID: {
				ID:       set.ID,
				AssetIDs: deliverableAssetIDs,
			},
		},
	}.DeterministicAsset(PickerInput{
		EntityType: entityType,
		EntityID:   entityID,
		SetID:      set.ID,
	})
	if err != nil {
		return resolved
	}
	cdnAssetID, found := lookup(set.ID, defaultAssetID)
	if !found {
		return resolved
	}

	resolved.AssetID = defaultAssetID
	resolved.CDNAssetID = cdnAssetID
	resolved.Status = DaggerheartAssetResolutionStatusSetDefault
	return resolved
}

func (m DaggerheartAssetManifest) defaultSetForAssetType(assetType string) (DaggerheartAssetSet, bool) {
	normalizedAssetType := strings.ToLower(strings.TrimSpace(assetType))
	if normalizedAssetType == "" {
		return DaggerheartAssetSet{}, false
	}
	setID, ok := m.defaultSetIDByAssetType[normalizedAssetType]
	if !ok {
		return DaggerheartAssetSet{}, false
	}
	set, ok := m.setsByID[setID]
	if !ok {
		return DaggerheartAssetSet{}, false
	}
	return set, true
}

func (m DaggerheartAssetManifest) setForIDAndAssetType(setID, assetType string) (DaggerheartAssetSet, bool) {
	normalizedSetID := strings.TrimSpace(setID)
	normalizedAssetType := strings.ToLower(strings.TrimSpace(assetType))
	if normalizedSetID == "" || normalizedAssetType == "" {
		return DaggerheartAssetSet{}, false
	}
	set, ok := m.setsByID[normalizedSetID]
	if !ok {
		return DaggerheartAssetSet{}, false
	}
	if strings.ToLower(strings.TrimSpace(set.AssetType)) != normalizedAssetType {
		return DaggerheartAssetSet{}, false
	}
	return set, true
}

func decodeDaggerheartAssetManifest(raw []byte) (DaggerheartAssetManifest, error) {
	var payload daggerheartAssetManifestJSONDocument
	if err := json.Unmarshal(raw, &payload); err != nil {
		return DaggerheartAssetManifest{}, fmt.Errorf("decode daggerheart asset manifest: %w", err)
	}

	manifestID := strings.TrimSpace(payload.ID)
	systemID := strings.TrimSpace(payload.SystemID)
	systemVersion := strings.TrimSpace(payload.SystemVersion)
	if manifestID == "" || systemID == "" || systemVersion == "" {
		return DaggerheartAssetManifest{}, fmt.Errorf("id/system_id/system_version are required")
	}

	setsByID := make(map[string]DaggerheartAssetSet, len(payload.Sets))
	defaultSetIDByAssetType := map[string]string{}
	for _, rawSet := range payload.Sets {
		setID := strings.TrimSpace(rawSet.ID)
		assetType := strings.ToLower(strings.TrimSpace(rawSet.AssetType))
		if setID == "" || assetType == "" {
			return DaggerheartAssetManifest{}, fmt.Errorf("set id/asset_type are required")
		}
		if _, exists := setsByID[setID]; exists {
			return DaggerheartAssetManifest{}, fmt.Errorf("duplicate daggerheart set id %q", setID)
		}
		setsByID[setID] = DaggerheartAssetSet{
			ID:        setID,
			AssetType: assetType,
			AssetIDs:  normalizeStringList(rawSet.AssetIDs),
		}
		if _, exists := defaultSetIDByAssetType[assetType]; !exists {
			defaultSetIDByAssetType[assetType] = setID
		}
	}

	entityAssets := make(map[string]DaggerheartEntityAsset, len(payload.EntityAssetMap))
	for _, rawEntityAsset := range payload.EntityAssetMap {
		entityType := strings.ToLower(strings.TrimSpace(rawEntityAsset.EntityType))
		entityID := strings.TrimSpace(rawEntityAsset.EntityID)
		assetType := strings.ToLower(strings.TrimSpace(rawEntityAsset.AssetType))
		if entityType == "" || entityID == "" || assetType == "" {
			continue
		}
		entry := DaggerheartEntityAsset{
			EntityType: entityType,
			EntityID:   entityID,
			AssetType:  assetType,
			SetID:      strings.TrimSpace(rawEntityAsset.SetID),
			AssetID:    strings.TrimSpace(rawEntityAsset.AssetID),
		}
		entityAssets[daggerheartEntityAssetLookupKey(entityType, entityID, assetType)] = entry
	}

	return DaggerheartAssetManifest{
		ID:                      manifestID,
		SystemID:                systemID,
		SystemVersion:           systemVersion,
		Locale:                  strings.TrimSpace(payload.Locale),
		Theme:                   strings.TrimSpace(payload.Theme),
		setsByID:                setsByID,
		defaultSetIDByAssetType: defaultSetIDByAssetType,
		entityAssets:            entityAssets,
	}, nil
}

func copyDaggerheartAssetManifest(source DaggerheartAssetManifest) DaggerheartAssetManifest {
	out := DaggerheartAssetManifest{
		ID:                      source.ID,
		SystemID:                source.SystemID,
		SystemVersion:           source.SystemVersion,
		Locale:                  source.Locale,
		Theme:                   source.Theme,
		setsByID:                map[string]DaggerheartAssetSet{},
		defaultSetIDByAssetType: copyStringMap(source.defaultSetIDByAssetType),
		entityAssets:            map[string]DaggerheartEntityAsset{},
	}
	for setID, set := range source.setsByID {
		out.setsByID[setID] = DaggerheartAssetSet{
			ID:        strings.TrimSpace(set.ID),
			AssetType: strings.ToLower(strings.TrimSpace(set.AssetType)),
			AssetIDs:  append([]string(nil), set.AssetIDs...),
		}
	}
	for key, entry := range source.entityAssets {
		out.entityAssets[key] = DaggerheartEntityAsset{
			EntityType: strings.ToLower(strings.TrimSpace(entry.EntityType)),
			EntityID:   strings.TrimSpace(entry.EntityID),
			AssetType:  strings.ToLower(strings.TrimSpace(entry.AssetType)),
			SetID:      strings.TrimSpace(entry.SetID),
			AssetID:    strings.TrimSpace(entry.AssetID),
		}
	}
	return out
}

func assetIDInSet(set DaggerheartAssetSet, assetID string) bool {
	normalizedAssetID := strings.TrimSpace(assetID)
	if normalizedAssetID == "" {
		return false
	}
	for _, candidate := range set.AssetIDs {
		if strings.TrimSpace(candidate) == normalizedAssetID {
			return true
		}
	}
	return false
}

func daggerheartEntityAssetLookupKey(entityType, entityID, assetType string) string {
	return strings.ToLower(strings.TrimSpace(entityType)) +
		"\x00" +
		strings.TrimSpace(entityID) +
		"\x00" +
		strings.ToLower(strings.TrimSpace(assetType))
}
