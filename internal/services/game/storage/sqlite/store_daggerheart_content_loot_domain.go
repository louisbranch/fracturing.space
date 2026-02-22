package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func (s *Store) PutDaggerheartLootEntry(ctx context.Context, entry storage.DaggerheartLootEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("loot entry id is required")
	}

	return s.q.PutDaggerheartLootEntry(ctx, db.PutDaggerheartLootEntryParams{
		ID:          entry.ID,
		Name:        entry.Name,
		Roll:        int64(entry.Roll),
		Description: entry.Description,
		CreatedAt:   toMillis(entry.CreatedAt),
		UpdatedAt:   toMillis(entry.UpdatedAt),
	})
}

// GetDaggerheartLootEntry retrieves a Daggerheart loot catalog entry.
func (s *Store) GetDaggerheartLootEntry(ctx context.Context, id string) (storage.DaggerheartLootEntry, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartLootEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartLootEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartLootEntry{}, fmt.Errorf("loot entry id is required")
	}

	row, err := s.q.GetDaggerheartLootEntry(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartLootEntry{}, storage.ErrNotFound
		}
		return storage.DaggerheartLootEntry{}, fmt.Errorf("get daggerheart loot entry: %w", err)
	}

	return dbDaggerheartLootEntryToStorage(row), nil
}

// ListDaggerheartLootEntries lists all Daggerheart loot catalog entries.
func (s *Store) ListDaggerheartLootEntries(ctx context.Context) ([]storage.DaggerheartLootEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartLootEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart loot entries: %w", err)
	}

	entries := make([]storage.DaggerheartLootEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, dbDaggerheartLootEntryToStorage(row))
	}
	return entries, nil
}

// DeleteDaggerheartLootEntry removes a Daggerheart loot catalog entry.
func (s *Store) DeleteDaggerheartLootEntry(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("loot entry id is required")
	}

	return s.q.DeleteDaggerheartLootEntry(ctx, id)
}

// PutDaggerheartDamageType persists a Daggerheart damage type catalog entry.
func (s *Store) PutDaggerheartDamageType(ctx context.Context, entry storage.DaggerheartDamageTypeEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("damage type id is required")
	}

	return s.q.PutDaggerheartDamageType(ctx, db.PutDaggerheartDamageTypeParams{
		ID:          entry.ID,
		Name:        entry.Name,
		Description: entry.Description,
		CreatedAt:   toMillis(entry.CreatedAt),
		UpdatedAt:   toMillis(entry.UpdatedAt),
	})
}

// GetDaggerheartDamageType retrieves a Daggerheart damage type catalog entry.
func (s *Store) GetDaggerheartDamageType(ctx context.Context, id string) (storage.DaggerheartDamageTypeEntry, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartDamageTypeEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartDamageTypeEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartDamageTypeEntry{}, fmt.Errorf("damage type id is required")
	}

	row, err := s.q.GetDaggerheartDamageType(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartDamageTypeEntry{}, storage.ErrNotFound
		}
		return storage.DaggerheartDamageTypeEntry{}, fmt.Errorf("get daggerheart damage type: %w", err)
	}

	return dbDaggerheartDamageTypeToStorage(row), nil
}

// ListDaggerheartDamageTypes lists all Daggerheart damage type catalog entries.
func (s *Store) ListDaggerheartDamageTypes(ctx context.Context) ([]storage.DaggerheartDamageTypeEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartDamageTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart damage types: %w", err)
	}

	entries := make([]storage.DaggerheartDamageTypeEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, dbDaggerheartDamageTypeToStorage(row))
	}
	return entries, nil
}

// DeleteDaggerheartDamageType removes a Daggerheart damage type catalog entry.
func (s *Store) DeleteDaggerheartDamageType(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("damage type id is required")
	}

	return s.q.DeleteDaggerheartDamageType(ctx, id)
}

// PutDaggerheartDomain persists a Daggerheart domain catalog entry.
func (s *Store) PutDaggerheartDomain(ctx context.Context, domain storage.DaggerheartDomain) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(domain.ID) == "" {
		return fmt.Errorf("domain id is required")
	}

	return s.q.PutDaggerheartDomain(ctx, db.PutDaggerheartDomainParams{
		ID:          domain.ID,
		Name:        domain.Name,
		Description: domain.Description,
		CreatedAt:   toMillis(domain.CreatedAt),
		UpdatedAt:   toMillis(domain.UpdatedAt),
	})
}

// GetDaggerheartDomain retrieves a Daggerheart domain catalog entry.
func (s *Store) GetDaggerheartDomain(ctx context.Context, id string) (storage.DaggerheartDomain, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartDomain{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartDomain{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartDomain{}, fmt.Errorf("domain id is required")
	}

	row, err := s.q.GetDaggerheartDomain(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartDomain{}, storage.ErrNotFound
		}
		return storage.DaggerheartDomain{}, fmt.Errorf("get daggerheart domain: %w", err)
	}

	return dbDaggerheartDomainToStorage(row), nil
}

// ListDaggerheartDomains lists all Daggerheart domain catalog entries.
func (s *Store) ListDaggerheartDomains(ctx context.Context) ([]storage.DaggerheartDomain, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart domains: %w", err)
	}

	domains := make([]storage.DaggerheartDomain, 0, len(rows))
	for _, row := range rows {
		domains = append(domains, dbDaggerheartDomainToStorage(row))
	}
	return domains, nil
}

// DeleteDaggerheartDomain removes a Daggerheart domain catalog entry.
func (s *Store) DeleteDaggerheartDomain(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("domain id is required")
	}

	return s.q.DeleteDaggerheartDomain(ctx, id)
}

// PutDaggerheartDomainCard persists a Daggerheart domain card catalog entry.
func (s *Store) PutDaggerheartDomainCard(ctx context.Context, card storage.DaggerheartDomainCard) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(card.ID) == "" {
		return fmt.Errorf("domain card id is required")
	}

	return s.q.PutDaggerheartDomainCard(ctx, db.PutDaggerheartDomainCardParams{
		ID:          card.ID,
		Name:        card.Name,
		DomainID:    card.DomainID,
		Level:       int64(card.Level),
		Type:        card.Type,
		RecallCost:  int64(card.RecallCost),
		UsageLimit:  card.UsageLimit,
		FeatureText: card.FeatureText,
		CreatedAt:   toMillis(card.CreatedAt),
		UpdatedAt:   toMillis(card.UpdatedAt),
	})
}

// GetDaggerheartDomainCard retrieves a Daggerheart domain card catalog entry.
func (s *Store) GetDaggerheartDomainCard(ctx context.Context, id string) (storage.DaggerheartDomainCard, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartDomainCard{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartDomainCard{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartDomainCard{}, fmt.Errorf("domain card id is required")
	}

	row, err := s.q.GetDaggerheartDomainCard(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartDomainCard{}, storage.ErrNotFound
		}
		return storage.DaggerheartDomainCard{}, fmt.Errorf("get daggerheart domain card: %w", err)
	}

	return dbDaggerheartDomainCardToStorage(row), nil
}

// ListDaggerheartDomainCards lists all Daggerheart domain card catalog entries.
func (s *Store) ListDaggerheartDomainCards(ctx context.Context) ([]storage.DaggerheartDomainCard, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartDomainCards(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart domain cards: %w", err)
	}

	cards := make([]storage.DaggerheartDomainCard, 0, len(rows))
	for _, row := range rows {
		cards = append(cards, dbDaggerheartDomainCardToStorage(row))
	}
	return cards, nil
}

// ListDaggerheartDomainCardsByDomain lists Daggerheart domain cards for a domain.
func (s *Store) ListDaggerheartDomainCardsByDomain(ctx context.Context, domainID string) ([]storage.DaggerheartDomainCard, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(domainID) == "" {
		return nil, fmt.Errorf("domain id is required")
	}

	rows, err := s.q.ListDaggerheartDomainCardsByDomain(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart domain cards by domain: %w", err)
	}

	cards := make([]storage.DaggerheartDomainCard, 0, len(rows))
	for _, row := range rows {
		cards = append(cards, dbDaggerheartDomainCardToStorage(row))
	}
	return cards, nil
}

// DeleteDaggerheartDomainCard removes a Daggerheart domain card catalog entry.
func (s *Store) DeleteDaggerheartDomainCard(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("domain card id is required")
	}

	return s.q.DeleteDaggerheartDomainCard(ctx, id)
}

// PutDaggerheartWeapon persists a Daggerheart weapon catalog entry.
