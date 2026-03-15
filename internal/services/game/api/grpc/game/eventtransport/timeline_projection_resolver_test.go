package eventtransport

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type countingCampaignStore struct {
	*gametest.FakeCampaignStore
	getCalls int
}

func (s *countingCampaignStore) Get(ctx context.Context, id string) (storage.CampaignRecord, error) {
	s.getCalls++
	return s.FakeCampaignStore.Get(ctx, id)
}

func TestTimelineProjectionResolverResolve_UsesEventTypeAndCampaignIDFallback(t *testing.T) {
	campaignStore := &countingCampaignStore{FakeCampaignStore: gametest.NewFakeCampaignStore()}
	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Riverfall",
		Status: campaign.StatusDraft,
	}

	resolver := newTimelineProjectionResolver(timelineProjectionStores{
		Campaign: campaignStore,
	})

	evt := event.Event{
		Type:       event.Type("campaign.created"),
		CampaignID: "c1",
	}

	for idx := 0; idx < 2; idx++ {
		iconID, projection, err := resolver.resolve(context.Background(), evt)
		if err != nil {
			t.Fatalf("resolve[%d] returned error: %v", idx, err)
		}
		if iconID != commonv1.IconId_ICON_ID_CAMPAIGN {
			t.Fatalf("resolve[%d] icon = %s, want %s", idx, iconID.String(), commonv1.IconId_ICON_ID_CAMPAIGN.String())
		}
		if projection == nil || projection.GetTitle() != "Riverfall" {
			t.Fatalf("resolve[%d] projection title = %q, want %q", idx, projection.GetTitle(), "Riverfall")
		}
	}
	if campaignStore.getCalls != 1 {
		t.Fatalf("campaign get calls = %d, want 1", campaignStore.getCalls)
	}
}

func TestTimelineProjectionResolverResolve_UsesSessionIDFallback(t *testing.T) {
	sessionStore := gametest.NewFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {
			ID:         "s1",
			CampaignID: "c1",
			Name:       "Session 1",
			Status:     session.StatusActive,
			StartedAt:  now,
			UpdatedAt:  now,
		},
	}

	resolver := newTimelineProjectionResolver(timelineProjectionStores{
		Session: sessionStore,
	})
	evt := event.Event{
		Type:       event.Type("session.started"),
		EntityType: "SESSION",
		CampaignID: "c1",
		SessionID:  "s1",
	}

	iconID, projection, err := resolver.resolve(context.Background(), evt)
	if err != nil {
		t.Fatalf("resolve returned error: %v", err)
	}
	if iconID != commonv1.IconId_ICON_ID_SESSION {
		t.Fatalf("icon = %s, want %s", iconID.String(), commonv1.IconId_ICON_ID_SESSION.String())
	}
	if projection == nil || projection.GetTitle() != "Session 1" {
		t.Fatalf("projection title = %q, want %q", projection.GetTitle(), "Session 1")
	}
}

func TestTimelineProjectionResolverResolve_MissingStoreFailsFast(t *testing.T) {
	resolver := newTimelineProjectionResolver(timelineProjectionStores{})
	_, _, err := resolver.resolve(context.Background(), event.Event{
		Type:       event.Type("participant.joined"),
		EntityType: "participant",
		CampaignID: "c1",
		EntityID:   "p1",
	})
	if err == nil {
		t.Fatal("expected missing participant store error")
	}
	if !strings.Contains(err.Error(), "participant store is not configured") {
		t.Fatalf("error = %v, want participant store configuration message", err)
	}
}

func TestTimelineProjectionResolverResolve_UnknownDomainReturnsGeneric(t *testing.T) {
	resolver := newTimelineProjectionResolver(timelineProjectionStores{})
	iconID, projection, err := resolver.resolve(context.Background(), event.Event{
		Type:       event.Type("custom.event"),
		EntityType: "custom",
	})
	if err != nil {
		t.Fatalf("resolve returned error: %v", err)
	}
	if iconID != commonv1.IconId_ICON_ID_GENERIC {
		t.Fatalf("icon = %s, want %s", iconID.String(), commonv1.IconId_ICON_ID_GENERIC.String())
	}
	if projection != nil {
		t.Fatalf("projection = %#v, want nil", projection)
	}
}
