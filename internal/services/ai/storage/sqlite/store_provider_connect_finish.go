package sqlite

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/providerconnect"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
)

// FinishProviderConnect atomically stores a new provider grant and marks the
// pending connect session completed.
func (s *Store) FinishProviderConnect(ctx context.Context, grant providergrant.ProviderGrant, completedSession providerconnect.Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin finish provider connect tx: %w", err)
	}
	defer tx.Rollback()

	if err := putProviderGrant(ctx, tx, grant); err != nil {
		return err
	}
	if err := completeProviderConnectSession(ctx, tx, completedSession); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit finish provider connect tx: %w", err)
	}
	return nil
}
