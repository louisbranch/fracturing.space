package invite

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
)

const (
	outboxEventCreated  = "invite.invite.created.v1"
	outboxEventClaimed  = "invite.invite.claimed.v1"
	outboxEventDeclined = "invite.invite.declined.v1"
)

// outboxPayload is the durable worker-facing integration payload.
type outboxPayload struct {
	InviteID         string `json:"invite_id"`
	CampaignID       string `json:"campaign_id"`
	RecipientUserID  string `json:"recipient_user_id,omitempty"`
	NotificationKind string `json:"notification_kind"`
}

func enqueueInviteEvent(ctx context.Context, outbox storage.OutboxStore, idGen func() (string, error), now time.Time, eventType string, inv storage.InviteRecord) {
	evtID, err := idGen()
	if err != nil {
		log.Printf("invite outbox: generate id: %v", err)
		return
	}
	payload, _ := json.Marshal(outboxPayload{
		InviteID:         inv.ID,
		CampaignID:       inv.CampaignID,
		RecipientUserID:  inv.RecipientUserID,
		NotificationKind: eventType,
	})
	dedupeKey := "invite:" + inv.ID + ":" + eventType
	if err := outbox.Enqueue(ctx, storage.OutboxEvent{
		ID:          evtID,
		EventType:   eventType,
		PayloadJSON: payload,
		DedupeKey:   dedupeKey,
		CreatedAt:   now,
	}); err != nil {
		log.Printf("invite outbox: enqueue %s: %v", eventType, err)
	}
}
