package forktransport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/journalimport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
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

			toImport = append(toImport, forkEventForCampaign(evt, forkCampaignID, copyParticipants))
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
	case campaign.EventTypeCreated, campaign.EventTypeForked:
		return false, nil
	case campaign.EventTypeAIBound, campaign.EventTypeAIUnbound:
		return false, nil
	case participant.EventTypeJoined, participant.EventTypeUpdated, participant.EventTypeLeft:
		return copyParticipants, nil
	case character.EventTypeUpdated:
		if copyParticipants {
			return true, nil
		}
		var payload character.UpdatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return false, fmt.Errorf("decode character.updated payload: %w", err)
		}
		participantValue, hasParticipant := payload.Fields["owner_participant_id"]
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

func forkEventForCampaign(evt event.Event, campaignID string, copyParticipants bool) event.Event {
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
	if !copyParticipants {
		switch evt.Type {
		case character.EventTypeCreated:
			var payload character.CreatePayload
			if err := json.Unmarshal(forked.PayloadJSON, &payload); err == nil {
				payload.OwnerParticipantID = ""
				if payloadJSON, marshalErr := json.Marshal(payload); marshalErr == nil {
					forked.PayloadJSON = payloadJSON
				}
			}
		case character.EventTypeUpdated:
			var payload character.UpdatePayload
			if err := json.Unmarshal(forked.PayloadJSON, &payload); err == nil {
				if _, ok := payload.Fields["owner_participant_id"]; ok {
					payload.Fields["owner_participant_id"] = ""
					if payloadJSON, marshalErr := json.Marshal(payload); marshalErr == nil {
						forked.PayloadJSON = payloadJSON
					}
				}
			}
		}
	}
	return forked
}
