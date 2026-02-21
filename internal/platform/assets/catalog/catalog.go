package catalog

import (
	"errors"
	"hash/fnv"
	"net/url"
	"path"
	"strings"
)

const defaultAlgorithm = "asset-default-v1"

var (
	ErrSetNotFound  = errors.New("asset set is not configured")
	ErrSetEmpty     = errors.New("asset set has no assets")
	ErrEntityID     = errors.New("entity id is required")
	ErrEntityType   = errors.New("entity type is required")
	ErrAssetInvalid = errors.New("asset id is invalid for set")
)

// Set defines one image set and its stable ordered assets.
type Set struct {
	ID       string
	AssetIDs []string
}

// Manifest is a service-agnostic catalog definition.
//
// The owner service keeps entity records; the catalog only normalizes and
// validates set/asset identifiers plus deterministic defaults.
type Manifest struct {
	ID           string
	DefaultSet   string
	Sets         map[string]Set
	SetAliases   map[string]string
	AssetAliases map[string]string
}

// PickerInput identifies the entity and set used for deterministic defaults.
type PickerInput struct {
	EntityType string
	EntityID   string
	SetID      string
	Algorithm  string
}

// SelectionInput captures set/asset selection inputs for one entity.
type SelectionInput struct {
	EntityType string
	EntityID   string
	SetID      string
	AssetID    string
	Algorithm  string
}

// NormalizeSetID resolves aliases and verifies configured set membership.
func (m Manifest) NormalizeSetID(raw string) (string, bool) {
	setID := strings.TrimSpace(raw)
	if setID == "" {
		setID = strings.TrimSpace(m.DefaultSet)
	}
	if setID == "" {
		return "", false
	}
	if canonical, ok := m.SetAliases[setID]; ok {
		setID = strings.TrimSpace(canonical)
	}
	_, ok := m.Sets[setID]
	return setID, ok
}

// NormalizeAssetID resolves aliases and trims whitespace.
func (m Manifest) NormalizeAssetID(raw string) string {
	assetID := strings.TrimSpace(raw)
	if assetID == "" {
		return ""
	}
	if canonical, ok := m.AssetAliases[assetID]; ok {
		assetID = strings.TrimSpace(canonical)
	}
	return assetID
}

// ValidateAssetInSet reports whether set and asset identifiers are configured.
func (m Manifest) ValidateAssetInSet(setID, assetID string) bool {
	canonicalSetID, ok := m.NormalizeSetID(setID)
	if !ok {
		return false
	}
	normalizedAssetID := m.NormalizeAssetID(assetID)
	if normalizedAssetID == "" {
		return false
	}
	set := m.Sets[canonicalSetID]
	for _, candidate := range set.AssetIDs {
		if candidate == normalizedAssetID {
			return true
		}
	}
	return false
}

// DeterministicAsset chooses a stable default asset from one set.
func (m Manifest) DeterministicAsset(input PickerInput) (string, error) {
	entityType := strings.TrimSpace(input.EntityType)
	if entityType == "" {
		return "", ErrEntityType
	}
	entityID := strings.TrimSpace(input.EntityID)
	if entityID == "" {
		return "", ErrEntityID
	}
	setID, ok := m.NormalizeSetID(input.SetID)
	if !ok {
		return "", ErrSetNotFound
	}
	set := m.Sets[setID]
	if len(set.AssetIDs) == 0 {
		return "", ErrSetEmpty
	}
	algorithm := strings.TrimSpace(input.Algorithm)
	if algorithm == "" {
		algorithm = defaultAlgorithm
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(entityType))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(entityID))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(setID))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(algorithm))
	index := hasher.Sum64() % uint64(len(set.AssetIDs))
	return set.AssetIDs[index], nil
}

// ResolveSelection returns canonical set/asset identifiers for an entity.
//
// If AssetID is empty, a deterministic default asset is selected for the
// normalized set.
func (m Manifest) ResolveSelection(input SelectionInput) (string, string, error) {
	setID, ok := m.NormalizeSetID(input.SetID)
	if !ok {
		return "", "", ErrSetNotFound
	}

	assetID := m.NormalizeAssetID(input.AssetID)
	if assetID == "" {
		defaultAssetID, err := m.DeterministicAsset(PickerInput{
			EntityType: input.EntityType,
			EntityID:   input.EntityID,
			SetID:      setID,
			Algorithm:  input.Algorithm,
		})
		if err != nil {
			return "", "", err
		}
		return setID, defaultAssetID, nil
	}

	if !m.ValidateAssetInSet(setID, assetID) {
		return "", "", ErrAssetInvalid
	}
	return setID, assetID, nil
}

// BuildVersionedAssetKey returns a normalized key for object storage/CDN.
func BuildVersionedAssetKey(version, domain, setID, assetID, ext string) (string, error) {
	normalizedVersion := strings.TrimSpace(version)
	normalizedDomain := strings.TrimSpace(domain)
	normalizedSetID := strings.TrimSpace(setID)
	normalizedAssetID := strings.TrimSpace(assetID)
	if normalizedVersion == "" || normalizedDomain == "" || normalizedSetID == "" || normalizedAssetID == "" {
		return "", ErrAssetInvalid
	}
	normalizedExt := strings.TrimSpace(ext)
	if normalizedExt == "" {
		normalizedExt = ".png"
	}
	if !strings.HasPrefix(normalizedExt, ".") {
		normalizedExt = "." + normalizedExt
	}
	filename := normalizedAssetID + normalizedExt
	return path.Clean(path.Join(normalizedVersion, normalizedDomain, normalizedSetID, filename)), nil
}

// ResolveAssetURL joins a CDN/object-storage base URL with an asset key.
func ResolveAssetURL(baseURL, assetKey string) (string, error) {
	base := strings.TrimSpace(baseURL)
	key := strings.TrimSpace(assetKey)
	if base == "" || key == "" {
		return "", ErrAssetInvalid
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join(parsed.Path, key)
	return parsed.String(), nil
}
