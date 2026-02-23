package daggerheart

import (
	"fmt"

	bridgeDaggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type adversaryStatsInput struct {
	HP            *wrapperspb.Int32Value
	HPMax         *wrapperspb.Int32Value
	Stress        *wrapperspb.Int32Value
	StressMax     *wrapperspb.Int32Value
	Evasion       *wrapperspb.Int32Value
	Major         *wrapperspb.Int32Value
	Severe        *wrapperspb.Int32Value
	Armor         *wrapperspb.Int32Value
	RequireFields bool
	Current       *storage.DaggerheartAdversary
}

type adversaryStats struct {
	HP        int
	HPMax     int
	Stress    int
	StressMax int
	Evasion   int
	Major     int
	Severe    int
	Armor     int
}

const (
	defaultAdversaryEvasion = 10
	defaultAdversaryMajor   = 8
	defaultAdversarySevere  = 12
)

func normalizeAdversaryStats(input adversaryStatsInput) (adversaryStats, error) {
	stats := adversaryStats{
		HP:        bridgeDaggerheart.HPDefault,
		HPMax:     bridgeDaggerheart.HPMaxDefault,
		Stress:    bridgeDaggerheart.StressDefault,
		StressMax: bridgeDaggerheart.StressMaxDefault,
		Evasion:   defaultAdversaryEvasion,
		Major:     defaultAdversaryMajor,
		Severe:    defaultAdversarySevere,
		Armor:     bridgeDaggerheart.ArmorDefault,
	}
	if input.Current != nil {
		stats = adversaryStats{
			HP:        input.Current.HP,
			HPMax:     input.Current.HPMax,
			Stress:    input.Current.Stress,
			StressMax: input.Current.StressMax,
			Evasion:   input.Current.Evasion,
			Major:     input.Current.Major,
			Severe:    input.Current.Severe,
			Armor:     input.Current.Armor,
		}
	}

	if input.HPMax != nil {
		stats.HPMax = int(input.HPMax.GetValue())
	}
	if input.HP != nil {
		stats.HP = int(input.HP.GetValue())
	} else if input.HPMax != nil && input.Current == nil {
		stats.HP = stats.HPMax
	} else if input.HPMax != nil && input.Current != nil && stats.HP > stats.HPMax {
		stats.HP = stats.HPMax
	}

	if input.StressMax != nil {
		stats.StressMax = int(input.StressMax.GetValue())
	}
	if input.Stress != nil {
		stats.Stress = int(input.Stress.GetValue())
	} else if input.StressMax != nil && input.Current == nil {
		stats.Stress = stats.StressMax
	} else if input.StressMax != nil && input.Current != nil && stats.Stress > stats.StressMax {
		stats.Stress = stats.StressMax
	}

	if input.Evasion != nil {
		stats.Evasion = int(input.Evasion.GetValue())
	}
	if input.Major != nil {
		stats.Major = int(input.Major.GetValue())
	}
	if input.Severe != nil {
		stats.Severe = int(input.Severe.GetValue())
	}
	if input.Armor != nil {
		stats.Armor = int(input.Armor.GetValue())
	}

	if stats.HPMax <= 0 {
		return adversaryStats{}, fmt.Errorf("hp_max must be positive")
	}
	if stats.HP < 0 || stats.HP > stats.HPMax {
		return adversaryStats{}, fmt.Errorf("hp must be in range 0..%d", stats.HPMax)
	}
	if stats.StressMax < 0 {
		return adversaryStats{}, fmt.Errorf("stress_max must be non-negative")
	}
	if stats.Stress < 0 || stats.Stress > stats.StressMax {
		return adversaryStats{}, fmt.Errorf("stress must be in range 0..%d", stats.StressMax)
	}
	if stats.Evasion < 0 {
		return adversaryStats{}, fmt.Errorf("evasion must be non-negative")
	}
	if stats.Major < 0 || stats.Severe < 0 {
		return adversaryStats{}, fmt.Errorf("thresholds must be non-negative")
	}
	if stats.Severe < stats.Major {
		return adversaryStats{}, fmt.Errorf("severe_threshold must be >= major_threshold")
	}
	if stats.Armor < 0 {
		return adversaryStats{}, fmt.Errorf("armor must be non-negative")
	}

	if input.RequireFields && (input.HP == nil || input.HPMax == nil) {
		return adversaryStats{}, fmt.Errorf("hp and hp_max are required")
	}

	return stats, nil
}
