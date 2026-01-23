package bbolt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"go.etcd.io/bbolt"
)

const campaignBucket = "campaign"

// Store provides a BoltDB-backed campaign store.
type Store struct {
	db *bbolt.DB
}

// Open opens a BoltDB-backed store at the provided path.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	cleanPath := filepath.Clean(path)
	db, err := bbolt.Open(cleanPath, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("open storage db: %w", err)
	}

	store := &Store{db: db}
	if err := store.ensureBuckets(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

// Close closes the underlying BoltDB database.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Put persists a campaign record.
func (s *Store) Put(ctx context.Context, campaign domain.Campaign) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaign.ID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	payload, err := json.Marshal(campaign)
	if err != nil {
		return fmt.Errorf("marshal campaign: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(campaignBucket))
		if bucket == nil {
			return fmt.Errorf("campaign bucket is missing")
		}
		return bucket.Put(campaignKey(campaign.ID), payload)
	})
}

// Get fetches a campaign record by ID.
func (s *Store) Get(ctx context.Context, id string) (domain.Campaign, error) {
	if err := ctx.Err(); err != nil {
		return domain.Campaign{}, err
	}
	if s == nil || s.db == nil {
		return domain.Campaign{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return domain.Campaign{}, fmt.Errorf("campaign id is required")
	}

	var campaign domain.Campaign
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(campaignBucket))
		if bucket == nil {
			return fmt.Errorf("campaign bucket is missing")
		}
		payload := bucket.Get(campaignKey(id))
		if payload == nil {
			return storage.ErrNotFound
		}
		if err := json.Unmarshal(payload, &campaign); err != nil {
			return fmt.Errorf("unmarshal campaign: %w", err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return domain.Campaign{}, err
		}
		return domain.Campaign{}, err
	}

	return campaign, nil
}

func (s *Store) ensureBuckets() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(campaignBucket))
		if err != nil {
			return fmt.Errorf("create campaign bucket: %w", err)
		}
		return nil
	})
}

func campaignKey(id string) []byte {
	return []byte(id)
}

// TODO: Reserve index keys such as idx/creator/{creator_id}/campaign/{campaign_id}.
// TODO: Reserve index keys such as idx/campaign/{campaign_id}/session/{session_id}.
// TODO: Reserve index keys such as idx/session/{campaign_id}/{session_id}/actor/{actor_id}.
// TODO: Reserve session keys such as session/{campaign_id}/{session_id}.
// TODO: Reserve GM state keys such as gm/{campaign_id}/{session_id}.
// TODO: Reserve actor keys such as actor/{campaign_id}/{session_id}/{actor_id}.
// TODO: Reserve event keys such as event/{campaign_id}/{session_id}/{seq}.
// TODO: Add versioning and CAS semantics when multi-writer support is required.
