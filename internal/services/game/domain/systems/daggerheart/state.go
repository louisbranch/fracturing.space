package daggerheart

import (
	"context"
	"fmt"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
)

// Daggerheart-specific constants for resource ranges.
const (
	HPMin         = 0
	HPMaxCap      = 12
	HopeMin       = 0
	HopeMax       = 6
	HopeDefault   = 2
	StressMin     = 0
	StressMaxCap  = 12
	StressDefault = 0
	ArmorMin      = 0
	GMFearMin     = 0
	GMFearMax     = 12
	GMFearDefault = 0
)

// Resource names for Daggerheart.
const (
	ResourceHope   = "hope"
	ResourceStress = "stress"
	ResourceGMFear = "gm_fear"
	ResourceArmor  = "armor"
)

var (
	// ErrUnknownResource indicates an unknown resource name.
	ErrUnknownResource = apperrors.New(apperrors.CodeDaggerheartUnknownResource, "unknown resource")
	// ErrInsufficientResource indicates insufficient resource to spend.
	ErrInsufficientResource = apperrors.New(apperrors.CodeDaggerheartInsufficientResource, "insufficient resource")
	// ErrResourceAtCap indicates the resource is already at maximum.
	ErrResourceAtCap = apperrors.New(apperrors.CodeDaggerheartResourceAtCap, "resource at cap")
)

// CharacterState implements systems.CharacterStateHandler for Daggerheart.
// It manages HP, Hope, and Stress for a character.
type CharacterState struct {
	campaignID  string
	characterID string
	hp          int
	hpMax       int
	hope        int
	hopeMax     int
	stress      int
	stressMax   int
	armor       int
	armorMax    int
	lifeState   string
}

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

// NewCharacterState creates a new Daggerheart character state from config.
func NewCharacterState(cfg CharacterStateConfig) *CharacterState {
	cfg = clampCharacterConfig(cfg)
	return &CharacterState{
		campaignID:  cfg.CampaignID,
		characterID: cfg.CharacterID,
		hp:          cfg.HP,
		hpMax:       cfg.HPMax,
		hope:        cfg.Hope,
		hopeMax:     cfg.HopeMax,
		stress:      cfg.Stress,
		stressMax:   cfg.StressMax,
		armor:       cfg.Armor,
		armorMax:    cfg.ArmorMax,
		lifeState:   cfg.LifeState,
	}
}

// CampaignID returns the campaign this state belongs to.
func (s *CharacterState) CampaignID() string {
	return s.campaignID
}

// CharacterID returns the character this state belongs to.
func (s *CharacterState) CharacterID() string {
	return s.characterID
}

// Heal increases HP up to the maximum.
func (s *CharacterState) Heal(amount int) (before, after int) {
	before = s.hp
	s.hp = min(s.hp+amount, s.hpMax)
	return before, s.hp
}

// MaxHP returns the maximum HP.
func (s *CharacterState) MaxHP() int {
	return s.hpMax
}

// TakeDamage reduces HP, not below zero.
func (s *CharacterState) TakeDamage(amount int) (before, after int) {
	before = s.hp
	s.hp = max(s.hp-amount, 0)
	return before, s.hp
}

// CurrentHP returns the current HP.
func (s *CharacterState) CurrentHP() int {
	return s.hp
}

// GainResource increases a named resource (hope or stress).
func (s *CharacterState) GainResource(name string, amount int) (before, after int, err error) {
	switch name {
	case ResourceHope:
		before = s.hope
		s.hope = min(s.hope+amount, s.hopeMax)
		return before, s.hope, nil
	case ResourceStress:
		before = s.stress
		s.stress = min(s.stress+amount, s.stressMax)
		return before, s.stress, nil
	case ResourceArmor:
		before = s.armor
		s.armor = min(s.armor+amount, s.armorMax)
		return before, s.armor, nil
	default:
		return 0, 0, unknownResourceError(name)
	}
}

// SpendResource decreases a named resource (hope or stress).
func (s *CharacterState) SpendResource(name string, amount int) (before, after int, err error) {
	switch name {
	case ResourceHope:
		if s.hope < amount {
			return 0, 0, insufficientResourceError(name, s.hope, amount)
		}
		before = s.hope
		s.hope -= amount
		return before, s.hope, nil
	case ResourceStress:
		if s.stress < amount {
			return 0, 0, insufficientResourceError(name, s.stress, amount)
		}
		before = s.stress
		s.stress -= amount
		return before, s.stress, nil
	case ResourceArmor:
		if s.armor < amount {
			return 0, 0, insufficientResourceError(name, s.armor, amount)
		}
		before = s.armor
		s.armor -= amount
		return before, s.armor, nil
	default:
		return 0, 0, unknownResourceError(name)
	}
}

// ResourceValue returns the current value of a named resource.
func (s *CharacterState) ResourceValue(name string) int {
	switch name {
	case ResourceHope:
		return s.hope
	case ResourceStress:
		return s.stress
	case ResourceArmor:
		return s.armor
	default:
		return 0
	}
}

// ResourceCap returns the maximum value of a named resource.
func (s *CharacterState) ResourceCap(name string) int {
	switch name {
	case ResourceHope:
		return s.hopeMax
	case ResourceStress:
		return s.stressMax
	case ResourceArmor:
		return s.armorMax
	default:
		return 0
	}
}

// ResourceNames returns the names of all resources this holder manages.
func (s *CharacterState) ResourceNames() []string {
	return []string{ResourceHope, ResourceStress, ResourceArmor}
}

// Hope returns the current hope value.
func (s *CharacterState) Hope() int {
	return s.hope
}

// HopeMax returns the maximum hope value.
func (s *CharacterState) HopeMax() int {
	return s.hopeMax
}

// Stress returns the current stress value.
func (s *CharacterState) Stress() int {
	return s.stress
}

// Armor returns the current armor slots.
func (s *CharacterState) Armor() int {
	return s.armor
}

// SetHope sets the hope value directly (for storage layer).
func (s *CharacterState) SetHope(v int) {
	s.hope = min(max(v, HopeMin), s.hopeMax)
}

// SetHopeMax sets the hope max value directly (for storage layer).
func (s *CharacterState) SetHopeMax(v int) {
	s.hopeMax = min(max(v, HopeMin), HopeMax)
	if s.hope > s.hopeMax {
		s.hope = s.hopeMax
	}
}

// SetStress sets the stress value directly (for storage layer).
func (s *CharacterState) SetStress(v int) {
	s.stress = min(max(v, StressMin), s.stressMax)
}

// SetArmor sets the armor slots directly (for storage layer).
func (s *CharacterState) SetArmor(v int) {
	s.armor = min(max(v, ArmorMin), s.armorMax)
}

// Ensure CharacterState implements CharacterStateHandler.
var _ systems.CharacterStateHandler = (*CharacterState)(nil)

// SnapshotState implements systems.SnapshotStateHandler for Daggerheart.
// It manages campaign-level state like GM Fear.
type SnapshotState struct {
	campaignID string
	gmFear     int
	shortRests int
}

// SnapshotStateConfig contains the configuration for creating a SnapshotState.
type SnapshotStateConfig struct {
	CampaignID string
	GMFear     int
	ShortRests int
}

// NewSnapshotState creates a new Daggerheart snapshot projection from config.
func NewSnapshotState(cfg SnapshotStateConfig) *SnapshotState {
	return &SnapshotState{
		campaignID: cfg.CampaignID,
		gmFear:     cfg.GMFear,
		shortRests: cfg.ShortRests,
	}
}

// CampaignID returns the campaign this state belongs to.
func (s *SnapshotState) CampaignID() string {
	return s.campaignID
}

// GainResource increases GM Fear.
func (s *SnapshotState) GainResource(name string, amount int) (before, after int, err error) {
	if name != ResourceGMFear {
		return 0, 0, unknownResourceError(name)
	}
	before = s.gmFear
	s.gmFear = min(s.gmFear+amount, GMFearMax)
	return before, s.gmFear, nil
}

// SpendResource decreases GM Fear.
func (s *SnapshotState) SpendResource(name string, amount int) (before, after int, err error) {
	if name != ResourceGMFear {
		return 0, 0, unknownResourceError(name)
	}
	if s.gmFear < amount {
		return 0, 0, insufficientResourceError(name, s.gmFear, amount)
	}
	before = s.gmFear
	s.gmFear -= amount
	return before, s.gmFear, nil
}

// ResourceValue returns the current GM Fear value.
func (s *SnapshotState) ResourceValue(name string) int {
	if name == ResourceGMFear {
		return s.gmFear
	}
	return 0
}

// ResourceCap returns the GM Fear cap.
func (s *SnapshotState) ResourceCap(name string) int {
	if name == ResourceGMFear {
		return GMFearMax
	}
	return 0
}

// ResourceNames returns the names of all resources this holder manages.
func (s *SnapshotState) ResourceNames() []string {
	return []string{ResourceGMFear}
}

// GMFear returns the current GM Fear value.
func (s *SnapshotState) GMFear() int {
	return s.gmFear
}

// ShortRests returns the consecutive short rest count.
func (s *SnapshotState) ShortRests() int {
	return s.shortRests
}

// SetGMFear sets the GM Fear value directly (for storage layer).
func (s *SnapshotState) SetGMFear(v int) {
	s.gmFear = min(max(v, GMFearMin), GMFearMax)
}

// SetShortRests sets the consecutive short rest count directly (for storage layer).
func (s *SnapshotState) SetShortRests(v int) {
	if v < 0 {
		v = 0
	}
	s.shortRests = v
}

// Ensure SnapshotState implements SnapshotStateHandler.
var _ systems.SnapshotStateHandler = (*SnapshotState)(nil)

// StateFactory implements systems.StateFactory for Daggerheart.
type StateFactory struct{}

// NewStateFactory creates a new Daggerheart state factory.
func NewStateFactory() *StateFactory {
	return &StateFactory{}
}

// NewCharacterState creates initial character state for the given character.
func (f *StateFactory) NewCharacterState(campaignID, characterID string, kind systems.CharacterKind) (systems.CharacterStateHandler, error) {
	// Default values for new characters
	cfg := CharacterStateConfig{
		CampaignID:  campaignID,
		CharacterID: characterID,
		HP:          6, // Default HP max for PCs
		HPMax:       6,
		Hope:        HopeDefault,
		HopeMax:     HopeMax,
		Stress:      StressDefault,
		StressMax:   6, // Default stress max
		Armor:       0,
		ArmorMax:    0,
		LifeState:   LifeStateAlive,
	}

	// NPCs may have different defaults
	if kind == systems.CharacterKindNPC {
		cfg.Hope = 0
		cfg.StressMax = 0
	}

	return NewCharacterState(cfg), nil
}

// NewSnapshotState creates an initial snapshot projection for the given campaign.
func (f *StateFactory) NewSnapshotState(campaignID string) (systems.SnapshotStateHandler, error) {
	cfg := SnapshotStateConfig{
		CampaignID: campaignID,
		GMFear:     GMFearDefault,
		ShortRests: 0,
	}
	return NewSnapshotState(cfg), nil
}

// Ensure StateFactory implements systems.StateFactory.
var _ systems.StateFactory = (*StateFactory)(nil)

// OutcomeApplier implements systems.OutcomeApplier for Daggerheart.
type OutcomeApplier struct{}

// NewOutcomeApplier creates a new Daggerheart outcome applier.
func NewOutcomeApplier() *OutcomeApplier {
	return &OutcomeApplier{}
}

// ApplyOutcome applies a Daggerheart roll outcome to game state.
func (a *OutcomeApplier) ApplyOutcome(ctx context.Context, outcome systems.OutcomeContext) ([]systems.StateChange, error) {
	// This is a placeholder - actual implementation will depend on the outcome type
	// and will be integrated with the storage layer in Phase 4.
	return nil, nil
}

// Ensure OutcomeApplier implements systems.OutcomeApplier.
var _ systems.OutcomeApplier = (*OutcomeApplier)(nil)

// unknownResourceError creates a structured error for unknown resource names.
func unknownResourceError(name string) *apperrors.Error {
	return apperrors.WithMetadata(
		apperrors.CodeDaggerheartUnknownResource,
		fmt.Sprintf("unknown resource: %s", name),
		map[string]string{"Resource": name},
	)
}

func clampCharacterConfig(cfg CharacterStateConfig) CharacterStateConfig {
	cfg.HPMax = clamp(cfg.HPMax, HPMin, HPMaxCap)
	cfg.HP = clamp(cfg.HP, HPMin, cfg.HPMax)
	cfg.StressMax = clamp(cfg.StressMax, StressMin, StressMaxCap)
	cfg.Stress = clamp(cfg.Stress, StressMin, cfg.StressMax)
	cfg.HopeMax = clamp(cfg.HopeMax, HopeMin, HopeMax)
	cfg.Hope = clamp(cfg.Hope, HopeMin, cfg.HopeMax)
	cfg.ArmorMax = clamp(cfg.ArmorMax, ArmorMin, ArmorMaxCap)
	cfg.Armor = clamp(cfg.Armor, ArmorMin, cfg.ArmorMax)
	if cfg.LifeState == "" {
		cfg.LifeState = LifeStateAlive
	}
	return cfg
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

// insufficientResourceError creates a structured error for insufficient resources.
func insufficientResourceError(name string, have, need int) *apperrors.Error {
	return apperrors.WithMetadata(
		apperrors.CodeDaggerheartInsufficientResource,
		fmt.Sprintf("insufficient %s: have %d, need %d", name, have, need),
		map[string]string{
			"Resource": name,
			"Have":     fmt.Sprintf("%d", have),
			"Need":     fmt.Sprintf("%d", need),
		},
	)
}
