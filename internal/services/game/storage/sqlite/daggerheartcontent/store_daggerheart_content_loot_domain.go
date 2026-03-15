package daggerheartcontent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func (s *Store) PutDaggerheartLootEntry(ctx context.Context, entry contentstore.DaggerheartLootEntry) error {
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
func (s *Store) GetDaggerheartLootEntry(ctx context.Context, id string) (contentstore.DaggerheartLootEntry, error) {
	if err := ctx.Err(); err != nil {
		return contentstore.DaggerheartLootEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return contentstore.DaggerheartLootEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return contentstore.DaggerheartLootEntry{}, fmt.Errorf("loot entry id is required")
	}

	row, err := s.q.GetDaggerheartLootEntry(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartLootEntry{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartLootEntry{}, fmt.Errorf("get daggerheart loot entry: %w", err)
	}

	return dbDaggerheartLootEntryToStorage(row), nil
}

// ListDaggerheartLootEntries lists all Daggerheart loot catalog entries.
func (s *Store) ListDaggerheartLootEntries(ctx context.Context) ([]contentstore.DaggerheartLootEntry, error) {
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

	entries := make([]contentstore.DaggerheartLootEntry, 0, len(rows))
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
func (s *Store) PutDaggerheartDamageType(ctx context.Context, entry contentstore.DaggerheartDamageTypeEntry) error {
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
func (s *Store) GetDaggerheartDamageType(ctx context.Context, id string) (contentstore.DaggerheartDamageTypeEntry, error) {
	if err := ctx.Err(); err != nil {
		return contentstore.DaggerheartDamageTypeEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return contentstore.DaggerheartDamageTypeEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return contentstore.DaggerheartDamageTypeEntry{}, fmt.Errorf("damage type id is required")
	}

	row, err := s.q.GetDaggerheartDamageType(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartDamageTypeEntry{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartDamageTypeEntry{}, fmt.Errorf("get daggerheart damage type: %w", err)
	}

	return dbDaggerheartDamageTypeToStorage(row), nil
}

// ListDaggerheartDamageTypes lists all Daggerheart damage type catalog entries.
func (s *Store) ListDaggerheartDamageTypes(ctx context.Context) ([]contentstore.DaggerheartDamageTypeEntry, error) {
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

	entries := make([]contentstore.DaggerheartDamageTypeEntry, 0, len(rows))
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
func (s *Store) PutDaggerheartDomain(ctx context.Context, domain contentstore.DaggerheartDomain) error {
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
func (s *Store) GetDaggerheartDomain(ctx context.Context, id string) (contentstore.DaggerheartDomain, error) {
	if err := ctx.Err(); err != nil {
		return contentstore.DaggerheartDomain{}, err
	}
	if s == nil || s.sqlDB == nil {
		return contentstore.DaggerheartDomain{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return contentstore.DaggerheartDomain{}, fmt.Errorf("domain id is required")
	}

	row, err := s.q.GetDaggerheartDomain(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartDomain{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartDomain{}, fmt.Errorf("get daggerheart domain: %w", err)
	}

	return dbDaggerheartDomainToStorage(row), nil
}

// ListDaggerheartDomains lists all Daggerheart domain catalog entries.
func (s *Store) ListDaggerheartDomains(ctx context.Context) ([]contentstore.DaggerheartDomain, error) {
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

	domains := make([]contentstore.DaggerheartDomain, 0, len(rows))
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
func (s *Store) PutDaggerheartDomainCard(ctx context.Context, card contentstore.DaggerheartDomainCard) error {
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
func (s *Store) GetDaggerheartDomainCard(ctx context.Context, id string) (contentstore.DaggerheartDomainCard, error) {
	if err := ctx.Err(); err != nil {
		return contentstore.DaggerheartDomainCard{}, err
	}
	if s == nil || s.sqlDB == nil {
		return contentstore.DaggerheartDomainCard{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return contentstore.DaggerheartDomainCard{}, fmt.Errorf("domain card id is required")
	}

	row, err := s.q.GetDaggerheartDomainCard(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartDomainCard{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartDomainCard{}, fmt.Errorf("get daggerheart domain card: %w", err)
	}

	return dbDaggerheartDomainCardToStorage(row), nil
}

// ListDaggerheartDomainCards lists all Daggerheart domain card catalog entries.
func (s *Store) ListDaggerheartDomainCards(ctx context.Context) ([]contentstore.DaggerheartDomainCard, error) {
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

	cards := make([]contentstore.DaggerheartDomainCard, 0, len(rows))
	for _, row := range rows {
		cards = append(cards, dbDaggerheartDomainCardToStorage(row))
	}
	return cards, nil
}

// ListDaggerheartDomainCardsByDomain lists Daggerheart domain cards for a domain.
func (s *Store) ListDaggerheartDomainCardsByDomain(ctx context.Context, domainID string) ([]contentstore.DaggerheartDomainCard, error) {
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

	cards := make([]contentstore.DaggerheartDomainCard, 0, len(rows))
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
