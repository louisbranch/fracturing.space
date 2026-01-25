package bbolt

import (
	"bytes"
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

const (
	campaignBucket    = "campaign"
	participantBucket = "participant"
	actorBucket       = "actor"
)

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

// List returns a page of campaign records ordered by storage key.
func (s *Store) List(ctx context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CampaignPage{}, err
	}
	if s == nil || s.db == nil {
		return storage.CampaignPage{}, fmt.Errorf("storage is not configured")
	}
	if pageSize <= 0 {
		return storage.CampaignPage{}, fmt.Errorf("page size must be greater than zero")
	}

	page := storage.CampaignPage{}
	var lastKey string
	viewErr := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(campaignBucket))
		if bucket == nil {
			return fmt.Errorf("campaign bucket is missing")
		}

		cursor := bucket.Cursor()
		var key, payload []byte
		if pageToken == "" {
			key, payload = cursor.First()
		} else {
			key, payload = cursor.Seek([]byte(pageToken))
			if key != nil && string(key) == pageToken {
				key, payload = cursor.Next()
			}
		}

		for key != nil && len(page.Campaigns) < pageSize {
			if err := ctx.Err(); err != nil {
				return err
			}
			var campaign domain.Campaign
			if err := json.Unmarshal(payload, &campaign); err != nil {
				return fmt.Errorf("unmarshal campaign: %w", err)
			}
			page.Campaigns = append(page.Campaigns, campaign)
			lastKey = string(key)
			key, payload = cursor.Next()
		}

		if key != nil && len(page.Campaigns) > 0 {
			page.NextPageToken = lastKey
		}
		return nil
	})
	if viewErr != nil {
		return storage.CampaignPage{}, viewErr
	}

	return page, nil
}

func (s *Store) ensureBuckets() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(campaignBucket))
		if err != nil {
			return fmt.Errorf("create campaign bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(participantBucket))
		if err != nil {
			return fmt.Errorf("create participant bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(actorBucket))
		if err != nil {
			return fmt.Errorf("create actor bucket: %w", err)
		}
		return nil
	})
}

func campaignKey(id string) []byte {
	return []byte(id)
}

func participantKey(campaignID, participantID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", campaignID, participantID))
}

func actorKey(campaignID, actorID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", campaignID, actorID))
}

// Put persists a participant record (implements storage.ParticipantStore).
func (s *Store) PutParticipant(ctx context.Context, participant domain.Participant) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(participant.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(participant.ID) == "" {
		return fmt.Errorf("participant id is required")
	}

	payload, err := json.Marshal(participant)
	if err != nil {
		return fmt.Errorf("marshal participant: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(participantBucket))
		if bucket == nil {
			return fmt.Errorf("participant bucket is missing")
		}
		return bucket.Put(participantKey(participant.CampaignID, participant.ID), payload)
	})
}

// Get fetches a participant record by campaign ID and participant ID (implements storage.ParticipantStore).
func (s *Store) GetParticipant(ctx context.Context, campaignID, participantID string) (domain.Participant, error) {
	if err := ctx.Err(); err != nil {
		return domain.Participant{}, err
	}
	if s == nil || s.db == nil {
		return domain.Participant{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return domain.Participant{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(participantID) == "" {
		return domain.Participant{}, fmt.Errorf("participant id is required")
	}

	var participant domain.Participant
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(participantBucket))
		if bucket == nil {
			return fmt.Errorf("participant bucket is missing")
		}
		payload := bucket.Get(participantKey(campaignID, participantID))
		if payload == nil {
			return storage.ErrNotFound
		}
		if err := json.Unmarshal(payload, &participant); err != nil {
			return fmt.Errorf("unmarshal participant: %w", err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return domain.Participant{}, err
		}
		return domain.Participant{}, err
	}

	return participant, nil
}

// ListByCampaign returns all participants for a campaign (implements storage.ParticipantStore).
func (s *Store) ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]domain.Participant, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}

	prefix := campaignID + "/"
	var participants []domain.Participant
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(participantBucket))
		if bucket == nil {
			return fmt.Errorf("participant bucket is missing")
		}

		cursor := bucket.Cursor()
		prefixBytes := []byte(prefix)
		for key, payload := cursor.Seek(prefixBytes); key != nil && bytes.HasPrefix(key, prefixBytes); key, payload = cursor.Next() {
			if err := ctx.Err(); err != nil {
				return err
			}
			var participant domain.Participant
			if err := json.Unmarshal(payload, &participant); err != nil {
				return fmt.Errorf("unmarshal participant: %w", err)
			}
			participants = append(participants, participant)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return participants, nil
}

// ListParticipants returns a page of participant records for a campaign ordered by storage key (implements storage.ParticipantStore).
func (s *Store) ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.ParticipantPage{}, err
	}
	if s == nil || s.db == nil {
		return storage.ParticipantPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ParticipantPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.ParticipantPage{}, fmt.Errorf("page size must be greater than zero")
	}

	prefix := campaignID + "/"
	page := storage.ParticipantPage{}
	var lastKey string
	viewErr := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(participantBucket))
		if bucket == nil {
			return fmt.Errorf("participant bucket is missing")
		}

		cursor := bucket.Cursor()
		prefixBytes := []byte(prefix)
		var key, payload []byte
		if pageToken == "" {
			key, payload = cursor.Seek(prefixBytes)
			if key != nil && !bytes.HasPrefix(key, prefixBytes) {
				key = nil
			}
		} else {
			key, payload = cursor.Seek([]byte(pageToken))
			if key != nil && string(key) == pageToken && bytes.HasPrefix(key, prefixBytes) {
				key, payload = cursor.Next()
			} else if key != nil && !bytes.HasPrefix(key, prefixBytes) {
				key = nil
			}
		}

		for key != nil && bytes.HasPrefix(key, prefixBytes) && len(page.Participants) < pageSize {
			if err := ctx.Err(); err != nil {
				return err
			}
			var participant domain.Participant
			if err := json.Unmarshal(payload, &participant); err != nil {
				return fmt.Errorf("unmarshal participant: %w", err)
			}
			page.Participants = append(page.Participants, participant)
			lastKey = string(key)
			key, payload = cursor.Next()
		}

		if key != nil && bytes.HasPrefix(key, prefixBytes) && len(page.Participants) > 0 {
			page.NextPageToken = lastKey
		}
		return nil
	})
	if viewErr != nil {
		return storage.ParticipantPage{}, viewErr
	}

	return page, nil
}

// PutActor persists an actor record (implements storage.ActorStore).
func (s *Store) PutActor(ctx context.Context, actor domain.Actor) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(actor.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(actor.ID) == "" {
		return fmt.Errorf("actor id is required")
	}

	payload, err := json.Marshal(actor)
	if err != nil {
		return fmt.Errorf("marshal actor: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(actorBucket))
		if bucket == nil {
			return fmt.Errorf("actor bucket is missing")
		}
		return bucket.Put(actorKey(actor.CampaignID, actor.ID), payload)
	})
}

// GetActor fetches an actor record by campaign ID and actor ID (implements storage.ActorStore).
func (s *Store) GetActor(ctx context.Context, campaignID, actorID string) (domain.Actor, error) {
	if err := ctx.Err(); err != nil {
		return domain.Actor{}, err
	}
	if s == nil || s.db == nil {
		return domain.Actor{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return domain.Actor{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(actorID) == "" {
		return domain.Actor{}, fmt.Errorf("actor id is required")
	}

	var actor domain.Actor
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(actorBucket))
		if bucket == nil {
			return fmt.Errorf("actor bucket is missing")
		}
		payload := bucket.Get(actorKey(campaignID, actorID))
		if payload == nil {
			return storage.ErrNotFound
		}
		if err := json.Unmarshal(payload, &actor); err != nil {
			return fmt.Errorf("unmarshal actor: %w", err)
		}
		return nil
	})
	if err != nil {

		return domain.Actor{}, err
	}

	return actor, nil
}

// ListActors returns a page of actor records for a campaign ordered by storage key (implements storage.ActorStore).
func (s *Store) ListActors(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.ActorPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.ActorPage{}, err
	}
	if s == nil || s.db == nil {
		return storage.ActorPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ActorPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.ActorPage{}, fmt.Errorf("page size must be greater than zero")
	}

	prefix := campaignID + "/"
	page := storage.ActorPage{}
	var lastKey string
	viewErr := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(actorBucket))
		if bucket == nil {
			return fmt.Errorf("actor bucket is missing")
		}

		cursor := bucket.Cursor()
		prefixBytes := []byte(prefix)
		var key, payload []byte
		if pageToken == "" {
			key, payload = cursor.Seek(prefixBytes)
			if key != nil && !bytes.HasPrefix(key, prefixBytes) {
				key = nil
			}
		} else {
			key, payload = cursor.Seek([]byte(pageToken))
			if key != nil && string(key) == pageToken && bytes.HasPrefix(key, prefixBytes) {
				key, payload = cursor.Next()
			} else if key != nil && !bytes.HasPrefix(key, prefixBytes) {
				key = nil
			}
		}

		for key != nil && bytes.HasPrefix(key, prefixBytes) && len(page.Actors) < pageSize {
			if err := ctx.Err(); err != nil {
				return err
			}
			var actor domain.Actor
			if err := json.Unmarshal(payload, &actor); err != nil {
				return fmt.Errorf("unmarshal actor: %w", err)
			}
			page.Actors = append(page.Actors, actor)
			lastKey = string(key)
			key, payload = cursor.Next()
		}

		if key != nil && bytes.HasPrefix(key, prefixBytes) && len(page.Actors) > 0 {
			page.NextPageToken = lastKey
		}
		return nil
	})
	if viewErr != nil {
		return storage.ActorPage{}, viewErr
	}

	return page, nil
}

// TODO: Reserve index keys such as idx/creator/{creator_id}/campaign/{campaign_id}.
// TODO: Reserve index keys such as idx/campaign/{campaign_id}/session/{session_id}.
// TODO: Reserve index keys such as idx/session/{campaign_id}/{session_id}/actor/{actor_id}.
// TODO: Reserve session keys such as session/{campaign_id}/{session_id}.
// TODO: Reserve GM state keys such as gm/{campaign_id}/{session_id}.
// TODO: Reserve actor keys such as actor/{campaign_id}/{session_id}/{actor_id}.
// TODO: Reserve event keys such as event/{campaign_id}/{session_id}/{seq}.
// TODO: Add versioning and CAS semantics when multi-writer support is required.
