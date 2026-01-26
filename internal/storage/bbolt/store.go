package bbolt

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	sessiondomain "github.com/louisbranch/duality-engine/internal/session/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"go.etcd.io/bbolt"
)

const (
	campaignBucket              = "campaign"
	participantBucket           = "participant"
	characterBucket             = "character"
	characterProfileBucket      = "character_profile"
	characterStateBucket        = "character_state"
	controlDefaultBucket        = "control_default"
	sessionsBucket              = "sessions"
	campaignActiveSessionBucket = "campaign_active_session"
	sessionEventsBucket         = "session_events"
	sessionEventSeqBucket       = "session_event_seq"
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
		_, err = tx.CreateBucketIfNotExists([]byte(characterBucket))
		if err != nil {
			return fmt.Errorf("create character bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(characterProfileBucket))
		if err != nil {
			return fmt.Errorf("create character profile bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(characterStateBucket))
		if err != nil {
			return fmt.Errorf("create character state bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(controlDefaultBucket))
		if err != nil {
			return fmt.Errorf("create control default bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(sessionsBucket))
		if err != nil {
			return fmt.Errorf("create sessions bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(campaignActiveSessionBucket))
		if err != nil {
			return fmt.Errorf("create campaign active session bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(sessionEventsBucket))
		if err != nil {
			return fmt.Errorf("create session events bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(sessionEventSeqBucket))
		if err != nil {
			return fmt.Errorf("create session event seq bucket: %w", err)
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

func characterKey(campaignID, characterID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", campaignID, characterID))
}

func characterProfileKey(campaignID, characterID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", campaignID, characterID))
}

func characterStateKey(campaignID, characterID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", campaignID, characterID))
}

func controlDefaultKey(campaignID, characterID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", campaignID, characterID))
}

func sessionKey(campaignID, sessionID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", campaignID, sessionID))
}

func activeSessionKey(campaignID string) []byte {
	return []byte(campaignID)
}

func sessionEventKey(sessionID string, seq uint64) []byte {
	prefix := []byte(sessionID + "/")
	key := make([]byte, len(prefix)+8)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix):], seq)
	return key
}

// Put persists a participant record (implements storage.ParticipantStore).
// Atomically increments the campaign's participant_count within the same transaction.
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

	participantPayload, err := json.Marshal(participant)
	if err != nil {
		return fmt.Errorf("marshal participant: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		// Load campaign to verify it exists
		campBucket := tx.Bucket([]byte(campaignBucket))
		if campBucket == nil {
			return fmt.Errorf("campaign bucket is missing")
		}
		campaignPayload := campBucket.Get(campaignKey(participant.CampaignID))
		if campaignPayload == nil {
			return storage.ErrNotFound
		}

		var campaign domain.Campaign
		if err := json.Unmarshal(campaignPayload, &campaign); err != nil {
			return fmt.Errorf("unmarshal campaign: %w", err)
		}

		// Check if participant already exists - only increment counter for new records
		partsBucket := tx.Bucket([]byte(participantBucket))
		if partsBucket == nil {
			return fmt.Errorf("participant bucket is missing")
		}
		participantKeyBytes := participantKey(participant.CampaignID, participant.ID)
		isNewParticipant := partsBucket.Get(participantKeyBytes) == nil

		// Store the participant
		if err := partsBucket.Put(participantKeyBytes, participantPayload); err != nil {
			return fmt.Errorf("put participant: %w", err)
		}

		// Increment participant count only for new records and update timestamp
		if isNewParticipant {
			campaign.ParticipantCount++
			campaign.UpdatedAt = time.Now().UTC()

			// Persist updated campaign
			updatedCampaignPayload, err := json.Marshal(campaign)
			if err != nil {
				return fmt.Errorf("marshal campaign: %w", err)
			}
			if err := campBucket.Put(campaignKey(participant.CampaignID), updatedCampaignPayload); err != nil {
				return fmt.Errorf("put campaign: %w", err)
			}
		}

		return nil
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

// PutCharacter persists a character record (implements storage.CharacterStore).
// Atomically increments the campaign's character_count within the same transaction.
func (s *Store) PutCharacter(ctx context.Context, character domain.Character) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(character.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(character.ID) == "" {
		return fmt.Errorf("character id is required")
	}

	characterPayload, err := json.Marshal(character)
	if err != nil {
		return fmt.Errorf("marshal character: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		// Load campaign to verify it exists
		campBucket := tx.Bucket([]byte(campaignBucket))
		if campBucket == nil {
			return fmt.Errorf("campaign bucket is missing")
		}
		campaignPayload := campBucket.Get(campaignKey(character.CampaignID))
		if campaignPayload == nil {
			return storage.ErrNotFound
		}

		var campaign domain.Campaign
		if err := json.Unmarshal(campaignPayload, &campaign); err != nil {
			return fmt.Errorf("unmarshal campaign: %w", err)
		}

		// Check if character already exists - only increment counter for new records
		charsBucket := tx.Bucket([]byte(characterBucket))
		if charsBucket == nil {
			return fmt.Errorf("character bucket is missing")
		}
		characterKeyBytes := characterKey(character.CampaignID, character.ID)
		isNewCharacter := charsBucket.Get(characterKeyBytes) == nil

		// Store the character
		if err := charsBucket.Put(characterKeyBytes, characterPayload); err != nil {
			return fmt.Errorf("put character: %w", err)
		}

		// Increment character count only for new records and update timestamp
		if isNewCharacter {
			campaign.CharacterCount++
			campaign.UpdatedAt = time.Now().UTC()

			// Persist updated campaign
			updatedCampaignPayload, err := json.Marshal(campaign)
			if err != nil {
				return fmt.Errorf("marshal campaign: %w", err)
			}
			if err := campBucket.Put(campaignKey(character.CampaignID), updatedCampaignPayload); err != nil {
				return fmt.Errorf("put campaign: %w", err)
			}
		}

		return nil
	})
}

// GetCharacter fetches a character record by campaign ID and character ID (implements storage.CharacterStore).
func (s *Store) GetCharacter(ctx context.Context, campaignID, characterID string) (domain.Character, error) {
	if err := ctx.Err(); err != nil {
		return domain.Character{}, err
	}
	if s == nil || s.db == nil {
		return domain.Character{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return domain.Character{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return domain.Character{}, fmt.Errorf("character id is required")
	}

	var character domain.Character
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(characterBucket))
		if bucket == nil {
			return fmt.Errorf("character bucket is missing")
		}
		payload := bucket.Get(characterKey(campaignID, characterID))
		if payload == nil {
			return storage.ErrNotFound
		}
		if err := json.Unmarshal(payload, &character); err != nil {
			return fmt.Errorf("unmarshal character: %w", err)
		}
		return nil
	})
	if err != nil {

		return domain.Character{}, err
	}

	return character, nil
}

// ListCharacters returns a page of character records for a campaign ordered by storage key (implements storage.CharacterStore).
func (s *Store) ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CharacterPage{}, err
	}
	if s == nil || s.db == nil {
		return storage.CharacterPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.CharacterPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.CharacterPage{}, fmt.Errorf("page size must be greater than zero")
	}

	prefix := campaignID + "/"
	page := storage.CharacterPage{}
	var lastKey string
	viewErr := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(characterBucket))
		if bucket == nil {
			return fmt.Errorf("character bucket is missing")
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

		for key != nil && bytes.HasPrefix(key, prefixBytes) && len(page.Characters) < pageSize {
			if err := ctx.Err(); err != nil {
				return err
			}
			var character domain.Character
			if err := json.Unmarshal(payload, &character); err != nil {
				return fmt.Errorf("unmarshal character: %w", err)
			}
			page.Characters = append(page.Characters, character)
			lastKey = string(key)
			key, payload = cursor.Next()
		}

		if key != nil && bytes.HasPrefix(key, prefixBytes) && len(page.Characters) > 0 {
			page.NextPageToken = lastKey
		}
		return nil
	})
	if viewErr != nil {
		return storage.CharacterPage{}, viewErr
	}

	return page, nil
}

// PutControlDefault persists a default controller assignment for a character (implements storage.ControlDefaultStore).
func (s *Store) PutControlDefault(ctx context.Context, campaignID, characterID string, controller domain.CharacterController) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return fmt.Errorf("character id is required")
	}
	if err := controller.Validate(); err != nil {
		return fmt.Errorf("validate controller: %w", err)
	}

	payload, err := json.Marshal(controller)
	if err != nil {
		return fmt.Errorf("marshal controller: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(controlDefaultBucket))
		if bucket == nil {
			return fmt.Errorf("control default bucket is missing")
		}
		return bucket.Put(controlDefaultKey(campaignID, characterID), payload)
	})
}

// GetSession fetches a session record by campaign ID and session ID (implements storage.SessionStore).
func (s *Store) GetSession(ctx context.Context, campaignID, sessionID string) (sessiondomain.Session, error) {
	if err := ctx.Err(); err != nil {
		return sessiondomain.Session{}, err
	}
	if s == nil || s.db == nil {
		return sessiondomain.Session{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return sessiondomain.Session{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return sessiondomain.Session{}, fmt.Errorf("session id is required")
	}

	var session sessiondomain.Session
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(sessionsBucket))
		if bucket == nil {
			return fmt.Errorf("sessions bucket is missing")
		}
		payload := bucket.Get(sessionKey(campaignID, sessionID))
		if payload == nil {
			return storage.ErrNotFound
		}
		if err := json.Unmarshal(payload, &session); err != nil {
			return fmt.Errorf("unmarshal session: %w", err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return sessiondomain.Session{}, err
		}
		return sessiondomain.Session{}, err
	}

	return session, nil
}

// GetActiveSession retrieves the active session for a campaign (implements storage.SessionStore).
func (s *Store) GetActiveSession(ctx context.Context, campaignID string) (sessiondomain.Session, error) {
	if err := ctx.Err(); err != nil {
		return sessiondomain.Session{}, err
	}
	if s == nil || s.db == nil {
		return sessiondomain.Session{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return sessiondomain.Session{}, fmt.Errorf("campaign id is required")
	}

	var sessionID string
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(campaignActiveSessionBucket))
		if bucket == nil {
			return fmt.Errorf("campaign active session bucket is missing")
		}
		payload := bucket.Get(activeSessionKey(campaignID))
		if payload == nil {
			return storage.ErrNotFound
		}
		sessionID = string(payload)
		return nil
	})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return sessiondomain.Session{}, err
		}
		return sessiondomain.Session{}, err
	}

	// Fetch the actual session record
	return s.GetSession(ctx, campaignID, sessionID)
}

// PutSession atomically stores a session and sets it as the active session for the campaign.
// This method ensures that only one active session exists per campaign.
// Returns ErrActiveSessionExists if an active session already exists for the campaign.
func (s *Store) PutSession(ctx context.Context, session sessiondomain.Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(session.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(session.ID) == "" {
		return fmt.Errorf("session id is required")
	}
	if session.Status != sessiondomain.SessionStatusActive {
		return fmt.Errorf("session must be ACTIVE to set as active session")
	}

	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		// Check if an active session already exists
		activeBucket := tx.Bucket([]byte(campaignActiveSessionBucket))
		if activeBucket == nil {
			return fmt.Errorf("campaign active session bucket is missing")
		}
		existingActive := activeBucket.Get(activeSessionKey(session.CampaignID))
		if existingActive != nil {
			return storage.ErrActiveSessionExists
		}

		// Store the session
		sessionBucket := tx.Bucket([]byte(sessionsBucket))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket is missing")
		}
		if err := sessionBucket.Put(sessionKey(session.CampaignID, session.ID), payload); err != nil {
			return fmt.Errorf("put session: %w", err)
		}

		// Set as active session
		if err := activeBucket.Put(activeSessionKey(session.CampaignID), []byte(session.ID)); err != nil {
			return fmt.Errorf("put active session pointer: %w", err)
		}

		return nil
	})
}

// ListSessions returns a page of session records for a campaign ordered by storage key (implements storage.SessionStore).
func (s *Store) ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.SessionPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionPage{}, err
	}
	if s == nil || s.db == nil {
		return storage.SessionPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.SessionPage{}, fmt.Errorf("page size must be greater than zero")
	}

	prefix := campaignID + "/"
	page := storage.SessionPage{}
	var lastKey string
	viewErr := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(sessionsBucket))
		if bucket == nil {
			return fmt.Errorf("sessions bucket is missing")
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

		for key != nil && bytes.HasPrefix(key, prefixBytes) && len(page.Sessions) < pageSize {
			if err := ctx.Err(); err != nil {
				return err
			}
			var session sessiondomain.Session
			if err := json.Unmarshal(payload, &session); err != nil {
				return fmt.Errorf("unmarshal session: %w", err)
			}
			page.Sessions = append(page.Sessions, session)
			lastKey = string(key)
			key, payload = cursor.Next()
		}

		if key != nil && bytes.HasPrefix(key, prefixBytes) && len(page.Sessions) > 0 {
			page.NextPageToken = lastKey
		}
		return nil
	})
	if viewErr != nil {
		return storage.SessionPage{}, viewErr
	}

	return page, nil
}

// AppendSessionEvent atomically appends a session event and returns it with seq set.
func (s *Store) AppendSessionEvent(ctx context.Context, event sessiondomain.SessionEvent) (sessiondomain.SessionEvent, error) {
	if err := ctx.Err(); err != nil {
		return sessiondomain.SessionEvent{}, err
	}
	if s == nil || s.db == nil {
		return sessiondomain.SessionEvent{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(event.SessionID) == "" {
		return sessiondomain.SessionEvent{}, fmt.Errorf("session id is required")
	}
	if !event.Type.IsValid() {
		return sessiondomain.SessionEvent{}, fmt.Errorf("session event type is required")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	var stored sessiondomain.SessionEvent
	updateErr := s.db.Update(func(tx *bbolt.Tx) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		seqBucket := tx.Bucket([]byte(sessionEventSeqBucket))
		if seqBucket == nil {
			return fmt.Errorf("session event seq bucket is missing")
		}

		currentSeq := uint64(0)
		if payload := seqBucket.Get([]byte(event.SessionID)); payload != nil {
			if len(payload) != 8 {
				return fmt.Errorf("invalid session event seq value")
			}
			currentSeq = binary.BigEndian.Uint64(payload)
		}

		event.Seq = currentSeq + 1
		seqBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(seqBytes, event.Seq)
		if err := seqBucket.Put([]byte(event.SessionID), seqBytes); err != nil {
			return fmt.Errorf("put session event seq: %w", err)
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal session event: %w", err)
		}

		eventBucket := tx.Bucket([]byte(sessionEventsBucket))
		if eventBucket == nil {
			return fmt.Errorf("session events bucket is missing")
		}
		if err := eventBucket.Put(sessionEventKey(event.SessionID, event.Seq), payload); err != nil {
			return fmt.Errorf("put session event: %w", err)
		}

		stored = event
		return nil
	})
	if updateErr != nil {
		return sessiondomain.SessionEvent{}, updateErr
	}

	return stored, nil
}

// ListSessionEvents returns a slice of session events ordered by sequence ascending.
func (s *Store) ListSessionEvents(ctx context.Context, sessionID string, afterSeq uint64, limit int) ([]sessiondomain.SessionEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("session id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	prefix := []byte(sessionID + "/")
	startKey := sessionEventKey(sessionID, afterSeq+1)
	results := make([]sessiondomain.SessionEvent, 0, limit)

	viewErr := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(sessionEventsBucket))
		if bucket == nil {
			return fmt.Errorf("session events bucket is missing")
		}

		cursor := bucket.Cursor()
		for key, payload := cursor.Seek(startKey); key != nil && bytes.HasPrefix(key, prefix) && len(results) < limit; key, payload = cursor.Next() {
			if err := ctx.Err(); err != nil {
				return err
			}
			var event sessiondomain.SessionEvent
			if err := json.Unmarshal(payload, &event); err != nil {
				return fmt.Errorf("unmarshal session event: %w", err)
			}
			results = append(results, event)
		}
		return nil
	})
	if viewErr != nil {
		return nil, viewErr
	}

	return results, nil
}

// PutCharacterProfile persists a character profile record (implements storage.CharacterProfileStore).
func (s *Store) PutCharacterProfile(ctx context.Context, profile domain.CharacterProfile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(profile.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(profile.CharacterID) == "" {
		return fmt.Errorf("character id is required")
	}

	profilePayload, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("marshal character profile: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(characterProfileBucket))
		if bucket == nil {
			return fmt.Errorf("character profile bucket is missing")
		}
		return bucket.Put(characterProfileKey(profile.CampaignID, profile.CharacterID), profilePayload)
	})
}

// GetCharacterProfile fetches a character profile record by campaign ID and character ID (implements storage.CharacterProfileStore).
func (s *Store) GetCharacterProfile(ctx context.Context, campaignID, characterID string) (domain.CharacterProfile, error) {
	if err := ctx.Err(); err != nil {
		return domain.CharacterProfile{}, err
	}
	if s == nil || s.db == nil {
		return domain.CharacterProfile{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return domain.CharacterProfile{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return domain.CharacterProfile{}, fmt.Errorf("character id is required")
	}

	var profile domain.CharacterProfile
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(characterProfileBucket))
		if bucket == nil {
			return fmt.Errorf("character profile bucket is missing")
		}
		payload := bucket.Get(characterProfileKey(campaignID, characterID))
		if payload == nil {
			return storage.ErrNotFound
		}
		if err := json.Unmarshal(payload, &profile); err != nil {
			return fmt.Errorf("unmarshal character profile: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.CharacterProfile{}, err
	}

	return profile, nil
}

// PutCharacterState persists a character state record (implements storage.CharacterStateStore).
func (s *Store) PutCharacterState(ctx context.Context, state domain.CharacterState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(state.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(state.CharacterID) == "" {
		return fmt.Errorf("character id is required")
	}

	statePayload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal character state: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(characterStateBucket))
		if bucket == nil {
			return fmt.Errorf("character state bucket is missing")
		}
		return bucket.Put(characterStateKey(state.CampaignID, state.CharacterID), statePayload)
	})
}

// GetCharacterState fetches a character state record by campaign ID and character ID (implements storage.CharacterStateStore).
func (s *Store) GetCharacterState(ctx context.Context, campaignID, characterID string) (domain.CharacterState, error) {
	if err := ctx.Err(); err != nil {
		return domain.CharacterState{}, err
	}
	if s == nil || s.db == nil {
		return domain.CharacterState{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return domain.CharacterState{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return domain.CharacterState{}, fmt.Errorf("character id is required")
	}

	var state domain.CharacterState
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(characterStateBucket))
		if bucket == nil {
			return fmt.Errorf("character state bucket is missing")
		}
		payload := bucket.Get(characterStateKey(campaignID, characterID))
		if payload == nil {
			return storage.ErrNotFound
		}
		if err := json.Unmarshal(payload, &state); err != nil {
			return fmt.Errorf("unmarshal character state: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.CharacterState{}, err
	}

	return state, nil
}

// TODO: Reserve index keys such as idx/creator/{creator_id}/campaign/{campaign_id}.
// TODO: Reserve index keys such as idx/campaign/{campaign_id}/session/{session_id}.
// TODO: Reserve index keys such as idx/session/{campaign_id}/{session_id}/character/{character_id}.
// TODO: Reserve session keys such as session/{campaign_id}/{session_id}.
// TODO: Reserve GM state keys such as gm/{campaign_id}/{session_id}.
// TODO: Reserve character keys such as character/{campaign_id}/{session_id}/{character_id}.
// TODO: Reserve event keys such as event/{campaign_id}/{session_id}/{seq}.
// TODO: Add versioning and CAS semantics when multi-writer support is required.
