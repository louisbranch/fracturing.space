package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func (s *Store) PutDaggerheartWeapon(ctx context.Context, weapon storage.DaggerheartWeapon) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(weapon.ID) == "" {
		return fmt.Errorf("weapon id is required")
	}

	damageDiceJSON, err := json.Marshal(weapon.DamageDice)
	if err != nil {
		return fmt.Errorf("marshal weapon damage dice: %w", err)
	}

	return s.q.PutDaggerheartWeapon(ctx, db.PutDaggerheartWeaponParams{
		ID:             weapon.ID,
		Name:           weapon.Name,
		Category:       weapon.Category,
		Tier:           int64(weapon.Tier),
		Trait:          weapon.Trait,
		Range:          weapon.Range,
		DamageDiceJson: string(damageDiceJSON),
		DamageType:     weapon.DamageType,
		Burden:         int64(weapon.Burden),
		Feature:        weapon.Feature,
		CreatedAt:      toMillis(weapon.CreatedAt),
		UpdatedAt:      toMillis(weapon.UpdatedAt),
	})
}

// GetDaggerheartWeapon retrieves a Daggerheart weapon catalog entry.
func (s *Store) GetDaggerheartWeapon(ctx context.Context, id string) (storage.DaggerheartWeapon, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartWeapon{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartWeapon{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartWeapon{}, fmt.Errorf("weapon id is required")
	}

	row, err := s.q.GetDaggerheartWeapon(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartWeapon{}, storage.ErrNotFound
		}
		return storage.DaggerheartWeapon{}, fmt.Errorf("get daggerheart weapon: %w", err)
	}

	weapon, err := dbDaggerheartWeaponToStorage(row)
	if err != nil {
		return storage.DaggerheartWeapon{}, err
	}
	return weapon, nil
}

// ListDaggerheartWeapons lists all Daggerheart weapon catalog entries.
func (s *Store) ListDaggerheartWeapons(ctx context.Context) ([]storage.DaggerheartWeapon, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartWeapons(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart weapons: %w", err)
	}

	weapons := make([]storage.DaggerheartWeapon, 0, len(rows))
	for _, row := range rows {
		weapon, err := dbDaggerheartWeaponToStorage(row)
		if err != nil {
			return nil, err
		}
		weapons = append(weapons, weapon)
	}
	return weapons, nil
}

// DeleteDaggerheartWeapon removes a Daggerheart weapon catalog entry.
func (s *Store) DeleteDaggerheartWeapon(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("weapon id is required")
	}

	return s.q.DeleteDaggerheartWeapon(ctx, id)
}

// PutDaggerheartArmor persists a Daggerheart armor catalog entry.
func (s *Store) PutDaggerheartArmor(ctx context.Context, armor storage.DaggerheartArmor) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(armor.ID) == "" {
		return fmt.Errorf("armor id is required")
	}

	return s.q.PutDaggerheartArmor(ctx, db.PutDaggerheartArmorParams{
		ID:                  armor.ID,
		Name:                armor.Name,
		Tier:                int64(armor.Tier),
		BaseMajorThreshold:  int64(armor.BaseMajorThreshold),
		BaseSevereThreshold: int64(armor.BaseSevereThreshold),
		ArmorScore:          int64(armor.ArmorScore),
		Feature:             armor.Feature,
		CreatedAt:           toMillis(armor.CreatedAt),
		UpdatedAt:           toMillis(armor.UpdatedAt),
	})
}

// GetDaggerheartArmor retrieves a Daggerheart armor catalog entry.
func (s *Store) GetDaggerheartArmor(ctx context.Context, id string) (storage.DaggerheartArmor, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartArmor{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartArmor{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartArmor{}, fmt.Errorf("armor id is required")
	}

	row, err := s.q.GetDaggerheartArmor(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartArmor{}, storage.ErrNotFound
		}
		return storage.DaggerheartArmor{}, fmt.Errorf("get daggerheart armor: %w", err)
	}

	return dbDaggerheartArmorToStorage(row), nil
}

// ListDaggerheartArmor lists all Daggerheart armor catalog entries.
func (s *Store) ListDaggerheartArmor(ctx context.Context) ([]storage.DaggerheartArmor, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartArmor(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart armor: %w", err)
	}

	armor := make([]storage.DaggerheartArmor, 0, len(rows))
	for _, row := range rows {
		armor = append(armor, dbDaggerheartArmorToStorage(row))
	}
	return armor, nil
}

// DeleteDaggerheartArmor removes a Daggerheart armor catalog entry.
func (s *Store) DeleteDaggerheartArmor(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("armor id is required")
	}

	return s.q.DeleteDaggerheartArmor(ctx, id)
}

// PutDaggerheartItem persists a Daggerheart item catalog entry.
func (s *Store) PutDaggerheartItem(ctx context.Context, item storage.DaggerheartItem) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(item.ID) == "" {
		return fmt.Errorf("item id is required")
	}

	return s.q.PutDaggerheartItem(ctx, db.PutDaggerheartItemParams{
		ID:          item.ID,
		Name:        item.Name,
		Rarity:      item.Rarity,
		Kind:        item.Kind,
		StackMax:    int64(item.StackMax),
		Description: item.Description,
		EffectText:  item.EffectText,
		CreatedAt:   toMillis(item.CreatedAt),
		UpdatedAt:   toMillis(item.UpdatedAt),
	})
}

// GetDaggerheartItem retrieves a Daggerheart item catalog entry.
func (s *Store) GetDaggerheartItem(ctx context.Context, id string) (storage.DaggerheartItem, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartItem{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartItem{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartItem{}, fmt.Errorf("item id is required")
	}

	row, err := s.q.GetDaggerheartItem(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartItem{}, storage.ErrNotFound
		}
		return storage.DaggerheartItem{}, fmt.Errorf("get daggerheart item: %w", err)
	}

	return dbDaggerheartItemToStorage(row), nil
}

// ListDaggerheartItems lists all Daggerheart item catalog entries.
func (s *Store) ListDaggerheartItems(ctx context.Context) ([]storage.DaggerheartItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart items: %w", err)
	}

	items := make([]storage.DaggerheartItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dbDaggerheartItemToStorage(row))
	}
	return items, nil
}

// DeleteDaggerheartItem removes a Daggerheart item catalog entry.
func (s *Store) DeleteDaggerheartItem(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("item id is required")
	}

	return s.q.DeleteDaggerheartItem(ctx, id)
}

// PutDaggerheartEnvironment persists a Daggerheart environment catalog entry.
