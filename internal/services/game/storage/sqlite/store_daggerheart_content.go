package sqlite

import (
	"context"
	"fmt"
	"strings"
)

// Daggerheart content catalog methods

func (s *Store) validateContentStore(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	return nil
}

func requireCatalogEntryID(id string, label string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s id is required", label)
	}
	return nil
}
