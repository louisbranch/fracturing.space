package daggerheart

import (
	"fmt"
	"strings"

	domainerrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
)

// Resource identifiers for character-facing state.
const (
	ResourceHope   = "hope"
	ResourceStress = "stress"
	ResourceGMFear = "gm_fear"
	ResourceArmor  = "armor"
)

// CharacterStateConfig contains the configuration for creating a CharacterState.
type CharacterStateConfig struct {
	CampaignID  string
	CharacterID string
	HP          int
	HPMax       int
	Hope        int
	HopeMax     int
	Stress      int
	StressMax   int
	Armor       int
	ArmorMax    int
	LifeState   string
}

var (
	ErrUnknownResource      = domainerrors.New(domainerrors.CodeDaggerheartUnknownResource, "unknown resource")
	ErrInsufficientResource = domainerrors.New(domainerrors.CodeDaggerheartInsufficientResource, "insufficient resource")
	ErrResourceAtCap        = domainerrors.New(domainerrors.CodeDaggerheartResourceAtCap, "resource at cap")
)

// NewCharacterState creates a character state with clamped values.
func NewCharacterState(cfg CharacterStateConfig) *CharacterState {
	cfg = clampCharacterConfig(cfg)
	cfg = normalizeLifeState(cfg)
	return &CharacterState{
		CampaignID:  cfg.CampaignID,
		CharacterID: cfg.CharacterID,
		HP:          cfg.HP,
		HPMax:       clamp(cfg.HPMax, HPMin, HPMaxCap),
		Hope:        cfg.Hope,
		HopeMax:     clamp(cfg.HopeMax, HopeMin, HopeMax),
		Stress:      cfg.Stress,
		StressMax:   clamp(cfg.StressMax, StressMin, StressMaxCap),
		Armor:       cfg.Armor,
		ArmorMax:    clamp(cfg.ArmorMax, ArmorMin, ArmorMaxCap),
		LifeState:   cfg.LifeState,
	}
}

// CampaignIDValue returns the campaign ID.
func (s *CharacterState) CampaignIDValue() string {
	return s.CampaignID
}

// CharacterIDValue returns the character ID.
func (s *CharacterState) CharacterIDValue() string {
	return s.CharacterID
}

// Heal increases HP by amount up to HPMax.
func (s *CharacterState) Heal(amount int) (before, after int) {
	before = s.HP
	s.HP = clamp(s.HP+amount, 0, s.HPMax)
	return before, s.HP
}

// TakeDamage decreases HP to a floor of zero.
func (s *CharacterState) TakeDamage(amount int) (before, after int) {
	before = s.HP
	s.HP = max(s.HP-amount, 0)
	return before, s.HP
}

// MaxHP returns the maximum HP.
func (s *CharacterState) MaxHP() int {
	return s.HPMax
}

// CurrentHP returns the current HP.
func (s *CharacterState) CurrentHP() int {
	return s.HP
}

// GainResource modifies a resource by name.
func (s *CharacterState) GainResource(name string, amount int) (before, after int, err error) {
	if amount < 0 {
		amount = 0
	}
	switch name {
	case ResourceHope:
		before = s.Hope
		after = min(s.Hope+amount, s.HopeMax)
		s.Hope = after
		return before, after, nil
	case ResourceStress:
		result, err := s.GainStress(amount)
		if err != nil {
			return 0, 0, err
		}
		return result.StressBefore, result.StressAfter, nil
	case ResourceArmor:
		before = s.Armor
		after = min(s.Armor+amount, s.ResourceCap(ResourceArmor))
		s.Armor = after
		return before, after, nil
	default:
		return 0, 0, unknownResourceError(name)
	}
}

// ApplyTemporaryArmor adds or replaces a temporary armor tracker and applies the
// immediate armor gain from that source.
func (s *CharacterState) ApplyTemporaryArmor(bucket TemporaryArmorBucket) {
	bucket.Source = strings.TrimSpace(bucket.Source)
	bucket.Duration = strings.TrimSpace(bucket.Duration)
	bucket.SourceID = strings.TrimSpace(bucket.SourceID)
	bucket.Amount = max(bucket.Amount, 0)
	if bucket.Source == "" || bucket.Duration == "" || bucket.Amount <= 0 {
		return
	}

	for i, existing := range s.ArmorBonus {
		existing.Source = strings.TrimSpace(existing.Source)
		existing.Duration = strings.TrimSpace(existing.Duration)
		existing.SourceID = strings.TrimSpace(existing.SourceID)
		if existing.Source == bucket.Source && existing.Duration == bucket.Duration && existing.SourceID == bucket.SourceID {
			existingAmount := existing.Amount
			if existingAmount < 0 {
				existingAmount = 0
			}
			s.ArmorBonus[i].Amount = bucket.Amount
			s.SetArmor(s.Armor - existingAmount + bucket.Amount)
			return
		}
	}

	s.ArmorBonus = append(s.ArmorBonus, bucket)
	s.SetArmor(s.Armor + bucket.Amount)
}

// ClearTemporaryArmorByDuration removes temporary armor buckets for the given
// duration and reduces current armor by the removed amount.
// It returns the removed total amount.
func (s *CharacterState) ClearTemporaryArmorByDuration(duration string) int {
	duration = strings.TrimSpace(duration)
	if len(s.ArmorBonus) == 0 {
		return 0
	}

	kept := make([]TemporaryArmorBucket, 0, len(s.ArmorBonus))
	removed := 0
	for _, bucket := range s.ArmorBonus {
		if bucket.Duration == duration {
			removed += bucket.Amount
			continue
		}
		kept = append(kept, bucket)
	}
	if removed > 0 {
		s.ArmorBonus = kept
		s.SetArmor(s.Armor - removed)
	}
	return removed
}

// TemporaryArmorAmount returns the sum of all active temporary armor bonuses.
func (s *CharacterState) TemporaryArmorAmount() int {
	total := 0
	for _, bucket := range s.ArmorBonus {
		if bucket.Amount > 0 {
			total += bucket.Amount
		}
	}
	return total
}

// StressGainResult captures stress overflow results.
type StressGainResult struct {
	StressBefore     int
	StressAfter      int
	HPBefore         int
	HPAfter          int
	Overflow         int
	LastStressMarked bool
}

// GainStress applies stress and converts overflow to HP damage.
func (s *CharacterState) GainStress(amount int) (StressGainResult, error) {
	result := StressGainResult{
		StressBefore: s.Stress,
		HPBefore:     s.HP,
	}
	if amount <= 0 {
		result.StressAfter = s.Stress
		result.HPAfter = s.HP
		return result, nil
	}
	if s.Stress >= s.StressMax {
		result.Overflow = amount
		result.StressAfter = s.StressMax
		s.HP = max(s.HP-amount, 0)
		result.HPAfter = s.HP
		return result, nil
	}
	needed := s.StressMax - s.Stress
	if amount >= needed {
		result.LastStressMarked = true
		result.Overflow = amount - needed
		s.Stress = s.StressMax
		if result.Overflow > 0 {
			s.HP = max(s.HP-result.Overflow, 0)
		}
	} else {
		s.Stress += amount
	}

	result.StressAfter = s.Stress
	result.HPAfter = s.HP
	return result, nil
}

// SpendResource decreases a named resource.
func (s *CharacterState) SpendResource(name string, amount int) (before, after int, err error) {
	if amount < 0 {
		amount = 0
	}
	switch name {
	case ResourceHope:
		if s.Hope < amount {
			return 0, 0, insufficientResourceError(name, s.Hope, amount)
		}
		before = s.Hope
		s.Hope -= amount
		return before, s.Hope, nil
	case ResourceStress:
		if s.Stress < amount {
			return 0, 0, insufficientResourceError(name, s.Stress, amount)
		}
		before = s.Stress
		s.Stress -= amount
		return before, s.Stress, nil
	case ResourceArmor:
		if s.Armor < amount {
			return 0, 0, insufficientResourceError(name, s.Armor, amount)
		}
		before = s.Armor
		s.Armor -= amount
		return before, s.Armor, nil
	default:
		return 0, 0, unknownResourceError(name)
	}
}

// ResourceValue returns a resource current value.
func (s *CharacterState) ResourceValue(name string) int {
	switch name {
	case ResourceHope:
		return s.Hope
	case ResourceStress:
		return s.Stress
	case ResourceArmor:
		return s.Armor
	default:
		return 0
	}
}

// ResourceCap returns a resource cap.
func (s *CharacterState) ResourceCap(name string) int {
	switch name {
	case ResourceHope:
		return s.HopeMax
	case ResourceStress:
		return s.StressMax
	case ResourceArmor:
		return s.ArmorMax + s.TemporaryArmorAmount()
	default:
		return 0
	}
}

// ResourceNames returns all resources managed by the state.
func (s *CharacterState) ResourceNames() []string {
	return []string{ResourceHope, ResourceStress, ResourceArmor}
}

// SetHope sets a bounded hope value.
func (s *CharacterState) SetHope(value int) {
	s.Hope = clamp(value, 0, s.HopeMax)
}

// SetHopeMax updates hope cap.
func (s *CharacterState) SetHopeMax(value int) {
	s.HopeMax = clamp(value, HopeMin, HopeMax)
	if s.Hope > s.HopeMax {
		s.Hope = s.HopeMax
	}
}

// SetStress sets a bounded stress value.
func (s *CharacterState) SetStress(value int) {
	s.Stress = clamp(value, 0, s.StressMax)
}

// SetArmor sets a bounded armor value.
func (s *CharacterState) SetArmor(value int) {
	s.Armor = clamp(value, 0, s.ResourceCap(ResourceArmor))
}

func clampCharacterConfig(cfg CharacterStateConfig) CharacterStateConfig {
	if cfg.HPMax <= 0 {
		cfg.HPMax = HPMaxDefault
	}
	cfg.HP = clamp(cfg.HP, 0, cfg.HPMax)
	cfg.HPMax = clamp(cfg.HPMax, HPMin, HPMaxCap)
	cfg.StressMax = clamp(cfg.StressMax, 0, StressMaxCap)
	cfg.Stress = clamp(cfg.Stress, 0, cfg.StressMax)
	cfg.HopeMax = clamp(cfg.HopeMax, 0, HopeMax)
	cfg.Hope = clamp(cfg.Hope, 0, cfg.HopeMax)
	cfg.ArmorMax = clamp(cfg.ArmorMax, 0, ArmorMaxCap)
	cfg.Armor = clamp(cfg.Armor, 0, cfg.ArmorMax)
	return cfg
}

func normalizeLifeState(cfg CharacterStateConfig) CharacterStateConfig {
	if cfg.LifeState == "" {
		cfg.LifeState = LifeStateAlive
	}
	return cfg
}

func unknownResourceError(name string) error {
	return domainerrors.WithMetadata(
		domainerrors.CodeDaggerheartUnknownResource,
		fmt.Sprintf("unknown resource: %s", name),
		map[string]string{"Resource": name},
	)
}

func insufficientResourceError(name string, have, need int) error {
	return domainerrors.WithMetadata(
		domainerrors.CodeDaggerheartInsufficientResource,
		fmt.Sprintf("insufficient %s: have %d, need %d", name, have, need),
		map[string]string{
			"Resource": name,
			"Have":     fmt.Sprintf("%d", have),
			"Need":     fmt.Sprintf("%d", need),
		},
	)
}

func clamp(value, minValue, maxValue int) int {
	if minValue > maxValue {
		return minValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
