package server

import (
	"testing"
	"time"
)

func TestCampaignRoomAIRelayReadyWithValidGrant(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setAIBinding("AI", "agent-1")
	room.setAISessionGrant("grant-token", 1, time.Now().UTC().Add(time.Minute))

	if !room.aiRelayReady() {
		t.Fatal("expected ai relay to be ready with a non-expired grant")
	}
}

func TestCampaignRoomAIRelayReadyClearsExpiredGrant(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setAIBinding("AI", "agent-1")
	room.setAISessionGrant("grant-token", 1, time.Now().UTC().Add(time.Second))

	if room.aiRelayReady() {
		t.Fatal("expected ai relay to be not ready for an expired/near-expiry grant")
	}
	if got := room.aiSessionGrantValue(); got != "" {
		t.Fatalf("grant token = %q, want empty", got)
	}
}
