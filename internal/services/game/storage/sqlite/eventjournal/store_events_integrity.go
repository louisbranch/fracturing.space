package eventjournal

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
)

func (s *Store) VerifyEventIntegrity(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if s.keyring == nil {
		return fmt.Errorf("event integrity keyring is required")
	}

	campaignIDs, err := s.listEventCampaignIDs(ctx)
	if err != nil {
		return err
	}
	for _, campaignID := range campaignIDs {
		if err := s.verifyCampaignEvents(ctx, campaignID); err != nil {
			return err
		}
	}

	return nil
}

// ListEventCampaignIDs returns campaign IDs that have at least one stored event.
func (s *Store) ListEventCampaignIDs(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	return s.listEventCampaignIDs(ctx)
}

func (s *Store) listEventCampaignIDs(ctx context.Context) ([]string, error) {
	rows, err := s.sqlDB.QueryContext(ctx, "SELECT DISTINCT campaign_id FROM events ORDER BY campaign_id")
	if err != nil {
		return nil, fmt.Errorf("list campaign ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan campaign id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate campaign ids: %w", err)
	}
	return ids, nil
}

func (s *Store) verifyCampaignEvents(ctx context.Context, campaignID string) error {
	var lastSeq uint64
	prevChainHash := ""
	for {
		events, err := s.ListEvents(ctx, campaignID, lastSeq, 200)
		if err != nil {
			return fmt.Errorf("list events campaign_id=%s: %w", campaignID, err)
		}
		if len(events) == 0 {
			return nil
		}
		for _, evt := range events {
			if evt.Seq != lastSeq+1 {
				return fmt.Errorf("event sequence gap campaign_id=%s expected=%d got=%d", campaignID, lastSeq+1, evt.Seq)
			}
			if evt.Seq == 1 && evt.PrevHash != "" {
				return fmt.Errorf("first event prev hash must be empty campaign_id=%s", campaignID)
			}
			if evt.Seq > 1 && evt.PrevHash != prevChainHash {
				return fmt.Errorf("prev hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			hash, err := integrity.EventHash(evt)
			if err != nil {
				return fmt.Errorf("compute event hash campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}
			if hash != evt.Hash {
				return fmt.Errorf("event hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			chainHash, err := integrity.ChainHash(evt, prevChainHash)
			if err != nil {
				return fmt.Errorf("compute chain hash campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}
			if chainHash != evt.ChainHash {
				return fmt.Errorf("chain hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			if err := s.keyring.VerifyChainHash(campaignID, chainHash, evt.Signature, evt.SignatureKeyID); err != nil {
				return fmt.Errorf("signature mismatch campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}

			prevChainHash = evt.ChainHash
			lastSeq = evt.Seq
		}
	}
}
