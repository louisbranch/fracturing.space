package gametools

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/shared/mcpbridge"
)

func TestDirectDialerDialUsesBridgeContext(t *testing.T) {
	dialer := NewDirectDialer(Clients{})
	ctx := mcpbridge.WithSessionContext(context.Background(), mcpbridge.SessionContext{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		ParticipantID: "part-1",
	})

	session, err := dialer.Dial(ctx)
	if err != nil {
		t.Fatalf("Dial error = %v", err)
	}

	direct, ok := session.(*DirectSession)
	if !ok {
		t.Fatalf("Dial session = %T, want *DirectSession", session)
	}
	if direct.sc.CampaignID != "camp-1" || direct.sc.SessionID != "sess-1" || direct.sc.ParticipantID != "part-1" {
		t.Fatalf("Dial session context = %#v", direct.sc)
	}
}

func TestDirectDialerDialNormalizesBridgeContext(t *testing.T) {
	dialer := DirectDialer{
		clients:  Clients{},
		registry: productionToolRegistry{},
	}
	ctx := mcpbridge.WithSessionContext(context.Background(), mcpbridge.SessionContext{
		CampaignID:    " camp-1 ",
		SessionID:     " sess-1 ",
		ParticipantID: " part-1 ",
	})

	session, err := dialer.Dial(ctx)
	if err != nil {
		t.Fatalf("Dial error = %v", err)
	}

	direct, ok := session.(*DirectSession)
	if !ok {
		t.Fatalf("Dial session = %T, want *DirectSession", session)
	}
	if direct.sc.CampaignID != "camp-1" || direct.sc.SessionID != "sess-1" || direct.sc.ParticipantID != "part-1" {
		t.Fatalf("Dial normalized session context = %#v", direct.sc)
	}
}
