package eventjournal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
	sqliteintegrationoutbox "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/integrationoutbox"
	sqliteprojectionapplyoutbox "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/projectionapplyoutbox"
)

// AppendEvent atomically appends an event and returns it with sequence and hash set.
func (s *Store) AppendEvent(ctx context.Context, evt event.Event) (event.Event, error) {
	if err := ctx.Err(); err != nil {
		return event.Event{}, err
	}
	if s == nil || s.sqlDB == nil {
		return event.Event{}, fmt.Errorf("storage is not configured")
	}
	if s.eventRegistry == nil {
		return event.Event{}, fmt.Errorf("event registry is required")
	}

	validated, err := s.eventRegistry.ValidateForAppend(evt)
	if err != nil {
		return event.Event{}, err
	}
	evt = validated

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return event.Event{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	evt.Timestamp = evt.Timestamp.UTC().Truncate(time.Millisecond)

	cid := string(evt.CampaignID)

	if err := qtx.InitEventSeq(ctx, cid); err != nil {
		return event.Event{}, fmt.Errorf("init event seq: %w", err)
	}

	seq, err := qtx.GetEventSeq(ctx, cid)
	if err != nil {
		return event.Event{}, fmt.Errorf("get event seq: %w", err)
	}
	evt.Seq = uint64(seq)

	if err := qtx.IncrementEventSeq(ctx, cid); err != nil {
		return event.Event{}, fmt.Errorf("increment event seq: %w", err)
	}

	if s.keyring == nil {
		return event.Event{}, fmt.Errorf("event integrity keyring is required")
	}

	hash, err := integrity.EventHash(evt)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute event hash: %w", err)
	}
	if strings.TrimSpace(hash) == "" {
		return event.Event{}, fmt.Errorf("event hash is required")
	}
	evt.Hash = hash

	prevHash := ""
	if evt.Seq > 1 {
		prevRow, err := qtx.GetEventBySeq(ctx, db.GetEventBySeqParams{
			CampaignID: cid,
			Seq:        int64(evt.Seq - 1),
		})
		if err != nil {
			return event.Event{}, fmt.Errorf("load previous event: %w", err)
		}
		prevHash = prevRow.ChainHash
	}

	chainHash, err := integrity.ChainHash(evt, prevHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute chain hash: %w", err)
	}
	if strings.TrimSpace(chainHash) == "" {
		return event.Event{}, fmt.Errorf("chain hash is required")
	}

	signature, keyID, err := s.keyring.SignChainHash(cid, chainHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("sign chain hash: %w", err)
	}

	evt.PrevHash = prevHash
	evt.ChainHash = chainHash
	evt.Signature = signature
	evt.SignatureKeyID = keyID

	if err := qtx.AppendEvent(ctx, db.AppendEventParams{
		CampaignID:     cid,
		Seq:            int64(evt.Seq),
		EventHash:      evt.Hash,
		PrevEventHash:  prevHash,
		ChainHash:      chainHash,
		SignatureKeyID: keyID,
		EventSignature: signature,
		Timestamp:      sqliteutil.ToMillis(evt.Timestamp),
		EventType:      string(evt.Type),
		SessionID:      evt.SessionID.String(),
		SceneID:        evt.SceneID.String(),
		RequestID:      evt.RequestID,
		InvocationID:   evt.InvocationID,
		ActorType:      string(evt.ActorType),
		ActorID:        evt.ActorID,
		EntityType:     evt.EntityType,
		EntityID:       evt.EntityID,
		SystemID:       evt.SystemID,
		SystemVersion:  evt.SystemVersion,
		CorrelationID:  evt.CorrelationID,
		CausationID:    evt.CausationID,
		PayloadJson:    evt.PayloadJSON,
	}); err != nil {
		// Idempotency contract: if the event already exists (constraint violation on
		// the unique hash), return the previously stored copy. This allows callers to
		// safely retry AppendEvent without generating duplicate journal entries. The
		// lookup by hash confirms the stored event matches the one being appended.
		if isConstraintError(err) {
			stored, lookupErr := s.GetEventByHash(ctx, evt.Hash)
			if lookupErr == nil {
				return stored, nil
			}
		}
		return event.Event{}, fmt.Errorf("append event: %w", err)
	}
	// Outbox enqueue runs inside the same append transaction so the event and
	// its projection-apply work item are committed atomically. This guarantees
	// every persisted event has a corresponding outbox entry when the feature
	// is enabled, preventing silent projection gaps.
	if err := sqliteprojectionapplyoutbox.EnqueueForEvent(ctx, tx, evt, s.projectionApplyOutboxEnabled); err != nil {
		return event.Event{}, err
	}
	if err := sqliteintegrationoutbox.EnqueueForEvent(ctx, tx, evt); err != nil {
		return event.Event{}, err
	}

	if err := tx.Commit(); err != nil {
		return event.Event{}, fmt.Errorf("commit: %w", err)
	}

	return evt, nil
}

// BatchAppendEvents atomically appends multiple events in a single transaction.
//
// All events must belong to the same campaign. Sequence numbers are allocated
// contiguously, and chain hashes link each event to its predecessor — including
// the last previously stored event for the first item in the batch.
func (s *Store) BatchAppendEvents(ctx context.Context, events []event.Event) ([]event.Event, error) {
	if len(events) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if s.eventRegistry == nil {
		return nil, fmt.Errorf("event registry is required")
	}
	if s.keyring == nil {
		return nil, fmt.Errorf("event integrity keyring is required")
	}

	// Validate all events before opening a transaction.
	validated := make([]event.Event, len(events))
	for i, evt := range events {
		v, err := s.eventRegistry.ValidateForAppend(evt)
		if err != nil {
			return nil, fmt.Errorf("event %d: %w", i, err)
		}
		if v.Timestamp.IsZero() {
			v.Timestamp = time.Now().UTC()
		}
		v.Timestamp = v.Timestamp.UTC().Truncate(time.Millisecond)
		validated[i] = v
	}

	campaignID := string(validated[0].CampaignID)

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	if err := qtx.InitEventSeq(ctx, campaignID); err != nil {
		return nil, fmt.Errorf("init event seq: %w", err)
	}

	baseSeq, err := qtx.GetEventSeq(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("get event seq: %w", err)
	}

	// Load previous chain hash for linking the first event in the batch.
	prevChainHash := ""
	if baseSeq > 1 {
		prevRow, err := qtx.GetEventBySeq(ctx, db.GetEventBySeqParams{
			CampaignID: campaignID,
			Seq:        baseSeq - 1,
		})
		if err != nil {
			return nil, fmt.Errorf("load previous event: %w", err)
		}
		prevChainHash = prevRow.ChainHash
	}

	stored := make([]event.Event, len(validated))
	for i, evt := range validated {
		evt.Seq = uint64(baseSeq) + uint64(i)

		hash, err := integrity.EventHash(evt)
		if err != nil {
			return nil, fmt.Errorf("event %d hash: %w", i, err)
		}
		if strings.TrimSpace(hash) == "" {
			return nil, fmt.Errorf("event %d: hash is empty", i)
		}
		evt.Hash = hash

		chainHash, err := integrity.ChainHash(evt, prevChainHash)
		if err != nil {
			return nil, fmt.Errorf("event %d chain hash: %w", i, err)
		}
		if strings.TrimSpace(chainHash) == "" {
			return nil, fmt.Errorf("event %d: chain hash is empty", i)
		}

		signature, keyID, err := s.keyring.SignChainHash(campaignID, chainHash)
		if err != nil {
			return nil, fmt.Errorf("event %d sign: %w", i, err)
		}

		evt.PrevHash = prevChainHash
		evt.ChainHash = chainHash
		evt.Signature = signature
		evt.SignatureKeyID = keyID

		if err := qtx.AppendEvent(ctx, db.AppendEventParams{
			CampaignID:     campaignID,
			Seq:            int64(evt.Seq),
			EventHash:      evt.Hash,
			PrevEventHash:  prevChainHash,
			ChainHash:      chainHash,
			SignatureKeyID: keyID,
			EventSignature: signature,
			Timestamp:      sqliteutil.ToMillis(evt.Timestamp),
			EventType:      string(evt.Type),
			SessionID:      evt.SessionID.String(),
			SceneID:        evt.SceneID.String(),
			RequestID:      evt.RequestID,
			InvocationID:   evt.InvocationID,
			ActorType:      string(evt.ActorType),
			ActorID:        evt.ActorID,
			EntityType:     evt.EntityType,
			EntityID:       evt.EntityID,
			SystemID:       evt.SystemID,
			SystemVersion:  evt.SystemVersion,
			CorrelationID:  evt.CorrelationID,
			CausationID:    evt.CausationID,
			PayloadJson:    evt.PayloadJSON,
		}); err != nil {
			return nil, fmt.Errorf("append event %d: %w", i, err)
		}

		if err := sqliteprojectionapplyoutbox.EnqueueForEvent(ctx, tx, evt, s.projectionApplyOutboxEnabled); err != nil {
			return nil, err
		}
		if err := sqliteintegrationoutbox.EnqueueForEvent(ctx, tx, evt); err != nil {
			return nil, err
		}

		prevChainHash = chainHash
		stored[i] = evt
	}

	// Advance the sequence counter to account for all appended events.
	nextSeq := int64(baseSeq) + int64(len(events))
	if _, err := tx.ExecContext(ctx,
		"UPDATE event_seq SET next_seq = ? WHERE campaign_id = ?",
		nextSeq, campaignID,
	); err != nil {
		return nil, fmt.Errorf("update event seq: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return stored, nil
}
