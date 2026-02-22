package catalog

import (
	"errors"
	"hash/fnv"
	"strings"
)

const (
	DomainAvatars = "avatars"
	AvatarSetV1   = "avatar_set_v1"

	// AvatarRoleUser resolves to portrait slot 1.
	AvatarRoleUser = "user"
	// AvatarRoleParticipant resolves to portrait slot 1.
	AvatarRoleParticipant = "participant"
	// AvatarRoleCharacter resolves to one of portrait slots 2-4.
	AvatarRoleCharacter = "character"
)

var (
	// ErrAvatarRoleInvalid reports unsupported avatar roles for slot selection.
	ErrAvatarRoleInvalid = errors.New("avatar role is invalid")
	// ErrAvatarEntityIDRequired reports missing entity id when selecting a slot.
	ErrAvatarEntityIDRequired = errors.New("avatar entity id is required")
)

// AvatarPortrait defines one crop region inside an avatar sprite sheet.
type AvatarPortrait struct {
	Slot     int
	X        int
	Y        int
	WidthPX  int
	HeightPX int
}

// AvatarSheet defines dimensions and portrait slices for one avatar set.
type AvatarSheet struct {
	WidthPX   int
	HeightPX  int
	Portraits map[int]AvatarPortrait
}

// AvatarManifest returns the canonical built-in avatar set definition.
func AvatarManifest() Manifest {
	_, avatarData, _, err := EmbeddedCatalogManifests()
	if err != nil {
		return Manifest{}
	}
	return copyManifest(avatarData)
}

// AvatarAssetIDs returns the stable ordered avatar asset ids for v1 avatars.
func AvatarAssetIDs() []string {
	_, avatarData, _, err := EmbeddedCatalogManifests()
	if err != nil {
		return []string{}
	}
	avatarSet, ok := avatarData.Sets[AvatarSetV1]
	if !ok {
		return []string{}
	}
	return append([]string(nil), avatarSet.AssetIDs...)
}

// AvatarSheetBySetID returns avatar sprite-sheet metadata for one set id.
func AvatarSheetBySetID(setID string) (AvatarSheet, bool) {
	_, _, sheets, err := EmbeddedCatalogManifests()
	if err != nil {
		return AvatarSheet{}, false
	}
	sheet, ok := sheets[strings.TrimSpace(setID)]
	if !ok {
		return AvatarSheet{}, false
	}
	return copyAvatarSheet(sheet), true
}

// ResolveAvatarPortraitSlot returns a deterministic portrait slot by avatar role.
//
// User and participant roles always use slot 1. Character role resolves to a
// deterministic slot in [2,3,4] from entity identity.
func ResolveAvatarPortraitSlot(role, entityID string) (int, error) {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	switch normalizedRole {
	case AvatarRoleUser, AvatarRoleParticipant:
		return 1, nil
	case AvatarRoleCharacter:
		trimmedEntityID := strings.TrimSpace(entityID)
		if trimmedEntityID == "" {
			return 0, ErrAvatarEntityIDRequired
		}
		return deterministicAvatarSlot(
			normalizedRole,
			trimmedEntityID,
			[]int{2, 3, 4},
		), nil
	default:
		return 0, ErrAvatarRoleInvalid
	}
}

func deterministicAvatarSlot(role, entityID string, candidates []int) int {
	if len(candidates) == 0 {
		return 0
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte("avatar-slot-v1"))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(role))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(entityID))
	index := hasher.Sum64() % uint64(len(candidates))
	return candidates[index]
}

func copyAvatarSheet(source AvatarSheet) AvatarSheet {
	out := AvatarSheet{
		WidthPX:   source.WidthPX,
		HeightPX:  source.HeightPX,
		Portraits: map[int]AvatarPortrait{},
	}
	for slot, portrait := range source.Portraits {
		out.Portraits[slot] = portrait
	}
	return out
}
