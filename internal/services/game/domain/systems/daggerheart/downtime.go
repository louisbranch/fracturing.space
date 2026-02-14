package daggerheart

// DowntimeMove represents a downtime action.
type DowntimeMove int

const (
	DowntimeClearAllStress DowntimeMove = iota
	DowntimeRepairAllArmor
	DowntimePrepare
	DowntimeWorkOnProject
)

// DowntimeOptions configures downtime behavior.
type DowntimeOptions struct {
	PrepareWithGroup bool
}

// DowntimeResult captures the state changes from a downtime move.
type DowntimeResult struct {
	HopeBefore   int
	HopeAfter    int
	StressBefore int
	StressAfter  int
	ArmorBefore  int
	ArmorAfter   int
}

// ApplyDowntimeMove applies a downtime move to the character state.
func ApplyDowntimeMove(state *CharacterState, move DowntimeMove, opts DowntimeOptions) DowntimeResult {
	result := DowntimeResult{
		HopeBefore:   state.Hope(),
		StressBefore: state.Stress(),
		ArmorBefore:  state.Armor(),
	}

	switch move {
	case DowntimeClearAllStress:
		state.SetStress(0)
	case DowntimeRepairAllArmor:
		state.SetArmor(state.ResourceCap(ResourceArmor))
	case DowntimePrepare:
		gain := 1
		if opts.PrepareWithGroup {
			gain = 2
		}
		_, _, _ = state.GainResource(ResourceHope, gain)
	case DowntimeWorkOnProject:
		// No state changes.
	}

	result.HopeAfter = state.Hope()
	result.StressAfter = state.Stress()
	result.ArmorAfter = state.Armor()
	return result
}
