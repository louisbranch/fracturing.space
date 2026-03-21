package damagetransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

// containsString reports whether the slice contains the target string.
func containsString(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

// stringsToCharacterIDs converts a string slice to character ID values.
func stringsToCharacterIDs(values []string) []ids.CharacterID {
	if len(values) == 0 {
		return nil
	}
	out := make([]ids.CharacterID, 0, len(values))
	for _, v := range values {
		out = append(out, ids.CharacterID(v))
	}
	return out
}

// damageApplyInputFromProto maps a proto damage request into the
// transport-agnostic domain input consumed by ResolveDamageApplication.
func damageApplyInputFromProto(req *pb.DaggerheartDamageRequest) rules.DamageApplyInput {
	if req == nil {
		return rules.DamageApplyInput{}
	}
	input := rules.DamageApplyInput{
		Amount:       int(req.GetAmount()),
		Direct:       req.GetDirect(),
		AllowMassive: req.GetMassiveDamage(),
		Resistance: rules.ResistanceProfile{
			ResistPhysical: req.GetResistPhysical(),
			ResistMagic:    req.GetResistMagic(),
			ImmunePhysical: req.GetImmunePhysical(),
			ImmuneMagic:    req.GetImmuneMagic(),
		},
	}
	switch req.GetDamageType() {
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL:
		input.Types.Physical = true
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		input.Types.Magic = true
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		input.Types.Physical = true
		input.Types.Magic = true
	}
	return input
}
