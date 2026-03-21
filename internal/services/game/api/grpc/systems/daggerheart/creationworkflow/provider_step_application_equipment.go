package creationworkflow

import (
	"context"
	"errors"
	"strings"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// startingWeaponSelection captures the weapon metadata needed to enforce the
// SRD loadout invariant after content lookup has resolved each selected ID.
type startingWeaponSelection struct {
	ID       string
	Category string
	Tier     int
	Burden   int
}

// normalizeStartingWeaponIDs validates the selected starting weapons and
// returns them in canonical primary-then-secondary order for storage.
func normalizeStartingWeaponIDs(weapons []startingWeaponSelection) ([]string, error) {
	if len(weapons) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one weapon_id is required")
	}
	if len(weapons) > 2 {
		return nil, status.Error(codes.InvalidArgument, "at most two weapon_ids are allowed")
	}

	var primary startingWeaponSelection
	var secondary startingWeaponSelection
	hasPrimary := false
	hasSecondary := false
	for idx := range weapons {
		weapon := weapons[idx]
		if weapon.Tier != 1 {
			return nil, status.Errorf(codes.InvalidArgument, "weapon_id %q must be tier 1", weapon.ID)
		}
		switch strings.ToLower(strings.TrimSpace(weapon.Category)) {
		case "primary":
			if hasPrimary {
				return nil, status.Error(codes.InvalidArgument, "starting equipment can include at most one primary weapon")
			}
			primary = weapon
			hasPrimary = true
		case "secondary":
			if hasSecondary {
				return nil, status.Error(codes.InvalidArgument, "starting equipment can include at most one secondary weapon")
			}
			secondary = weapon
			hasSecondary = true
		default:
			return nil, status.Errorf(codes.InvalidArgument, "weapon_id %q has unsupported category %q", weapon.ID, weapon.Category)
		}
	}

	if !hasPrimary {
		return nil, status.Error(codes.InvalidArgument, "starting equipment must include exactly one primary weapon")
	}
	switch primary.Burden {
	case 1:
		if !hasSecondary {
			return nil, status.Error(codes.InvalidArgument, "one-handed primary weapons must include exactly one one-handed secondary weapon")
		}
	case 2:
		if hasSecondary {
			return nil, status.Error(codes.InvalidArgument, "two-handed primary weapons cannot also equip a secondary weapon")
		}
		return []string{primary.ID}, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "primary weapon_id %q must have burden 1 or 2", primary.ID)
	}

	if secondary.Burden != 1 {
		return nil, status.Errorf(codes.InvalidArgument, "secondary weapon_id %q must have burden 1", secondary.ID)
	}
	return []string{primary.ID, secondary.ID}, nil
}

// applyEquipmentInput validates the selected loadout and derives the creation
// profile's starting combat equipment and thresholds.
func applyEquipmentInput(ctx context.Context, content contentstore.DaggerheartContentReadStore, profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepEquipmentInput) error {
	weaponIDs := input.GetWeaponIds()
	seenWeaponIDs := make(map[string]struct{}, len(weaponIDs))
	selectedWeapons := make([]startingWeaponSelection, 0, len(weaponIDs))
	for _, weaponID := range weaponIDs {
		trimmedWeaponID := strings.TrimSpace(weaponID)
		if trimmedWeaponID == "" {
			return status.Error(codes.InvalidArgument, "weapon_ids must not contain empty values")
		}
		if _, seen := seenWeaponIDs[trimmedWeaponID]; seen {
			return status.Errorf(codes.InvalidArgument, "weapon_id %q is duplicated", trimmedWeaponID)
		}
		seenWeaponIDs[trimmedWeaponID] = struct{}{}
		weapon, err := content.GetDaggerheartWeapon(ctx, trimmedWeaponID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "weapon_id %q is not found", trimmedWeaponID)
			}
			return grpcerror.Internal("get weapon", err)
		}
		selectedWeapons = append(selectedWeapons, startingWeaponSelection{
			ID:       trimmedWeaponID,
			Category: weapon.Category,
			Tier:     weapon.Tier,
			Burden:   weapon.Burden,
		})
	}
	normalizedWeaponIDs, err := normalizeStartingWeaponIDs(selectedWeapons)
	if err != nil {
		return err
	}

	armorID, err := validate.RequiredID(input.GetArmorId(), "armor_id")
	if err != nil {
		return err
	}
	armor, err := content.GetDaggerheartArmor(ctx, armorID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return status.Errorf(codes.InvalidArgument, "armor_id %q is not found", armorID)
		}
		return grpcerror.Internal("get armor", err)
	}
	if armor.Tier != 1 {
		return status.Errorf(codes.InvalidArgument, "armor_id %q must be tier 1", armorID)
	}

	potionItemID := strings.TrimSpace(input.GetPotionItemId())
	if !daggerheart.IsValidStartingPotionItemID(potionItemID) {
		return status.Errorf(
			codes.InvalidArgument,
			"potion_item_id must be %q or %q",
			daggerheart.StartingPotionMinorHealthID,
			daggerheart.StartingPotionMinorStaminaID,
		)
	}
	if _, err := content.GetDaggerheartItem(ctx, potionItemID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return status.Errorf(codes.InvalidArgument, "potion_item_id %q is not found", potionItemID)
		}
		return grpcerror.Internal("get potion item", err)
	}

	profile.StartingWeaponIDs = normalizedWeaponIDs
	profile.StartingArmorID = armorID
	profile.StartingPotionItemID = potionItemID
	profile.EquippedArmorID = armorID
	profile.Proficiency = daggerheartprofile.PCProficiency
	profile.ArmorScore = armor.ArmorScore
	profile.ArmorMax = armor.ArmorScore
	if profile.Level == 0 {
		profile.Level = daggerheartprofile.PCLevelDefault
	}
	profile.MajorThreshold, profile.SevereThreshold = daggerheartprofile.DeriveThresholds(
		profile.Level,
		profile.ArmorScore,
		armor.BaseMajorThreshold,
		armor.BaseSevereThreshold,
	)
	return nil
}
