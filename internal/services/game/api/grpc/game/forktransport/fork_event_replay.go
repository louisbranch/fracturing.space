package forktransport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/journalimport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// forkEventReplay owns list/filter/append/apply replay for fork creation so the
// top-level fork application can stay focused on the campaign fork use-case.
type forkEventReplay struct {
	events   storage.EventStore
	importer journalimport.Importer
}

func (r forkEventReplay) CopyToCampaign(
	ctx context.Context,
	sourceCampaignID string,
	forkCampaignID string,
	forkEventSeq uint64,
	copyParticipants bool,
) (time.Time, error) {
	if forkEventSeq == 0 {
		return time.Time{}, nil
	}

	afterSeq := uint64(0)
	var lastEventAt time.Time
	for {
		events, err := r.events.ListEvents(ctx, sourceCampaignID, afterSeq, forkEventPageSize)
		if err != nil {
			return lastEventAt, fmt.Errorf("list events: %w", err)
		}
		if len(events) == 0 {
			return lastEventAt, nil
		}

		toImport := make([]event.Event, 0, len(events))
		for _, evt := range events {
			if evt.Seq > forkEventSeq {
				if err := r.importer.Import(ctx, toImport); err != nil {
					return lastEventAt, fmt.Errorf("import forked events: %w", err)
				}
				return lastEventAt, nil
			}
			lastEventAt = evt.Timestamp

			shouldCopy, err := shouldCopyForkEvent(evt, copyParticipants)
			if err != nil {
				return lastEventAt, fmt.Errorf("filter forked event: %w", err)
			}
			if !shouldCopy {
				afterSeq = evt.Seq
				continue
			}

			toImport = append(toImport, forkEventForCampaign(evt, forkCampaignID))
			afterSeq = evt.Seq
		}
		if err := r.importer.Import(ctx, toImport); err != nil {
			return lastEventAt, fmt.Errorf("import forked events: %w", err)
		}

		if len(events) < forkEventPageSize {
			return lastEventAt, nil
		}
	}
}

func shouldCopyForkEvent(evt event.Event, copyParticipants bool) (bool, error) {
	switch evt.Type {
	case handler.EventTypeCampaignCreated, handler.EventTypeCampaignForked:
		return false, nil
	case handler.EventTypeCampaignAIBound, handler.EventTypeCampaignAIUnbound:
		return false, nil
	case handler.EventTypeParticipantJoined, handler.EventTypeParticipantUpdated, handler.EventTypeParticipantLeft:
		return copyParticipants, nil
	case handler.EventTypeCharacterUpdated:
		if copyParticipants {
			return true, nil
		}
		var payload character.UpdatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return false, fmt.Errorf("decode character.updated payload: %w", err)
		}
		participantValue, hasParticipant := payload.Fields["participant_id"]
		if !hasParticipant {
			return true, nil
		}
		participantID := strings.TrimSpace(participantValue)
		if participantID == "" {
			return true, nil
		}
		if len(payload.Fields) == 1 {
			return false, nil
		}
		return true, nil
	default:
		return true, nil
	}
}

func forkEventForCampaign(evt event.Event, campaignID string) event.Event {
	forked := evt
	forked.CampaignID = ids.CampaignID(campaignID)
	forked.Seq = 0
	forked.Hash = ""
	forked.PrevHash = ""
	forked.ChainHash = ""
	forked.Signature = ""
	forked.SignatureKeyID = ""
	if strings.EqualFold(evt.EntityType, "campaign") {
		forked.EntityID = campaignID
	}
	return forked
}
