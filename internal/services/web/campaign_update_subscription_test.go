package web

import (
	"context"
	"testing"

	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
)

func TestCampaignProjectionInitialAfterSeqUsesCursor(t *testing.T) {
	store := &fakeInvalidationCacheStore{
		cursors: map[string]webstorage.CampaignEventCursor{
			"camp-1": {CampaignID: "camp-1", LatestSeq: 42},
		},
	}
	eventClient := &fakeEventHeadClient{
		headByCampaign: map[string]uint64{"camp-1": 99},
	}

	afterSeq := campaignProjectionInitialAfterSeq(context.Background(), store, eventClient, "camp-1")
	if afterSeq != 42 {
		t.Fatalf("after seq = %d, want %d", afterSeq, 42)
	}
	if len(eventClient.listRequests) != 0 {
		t.Fatalf("expected no list requests when cursor exists, got %d", len(eventClient.listRequests))
	}
}

func TestCampaignProjectionInitialAfterSeqFallsBackToHead(t *testing.T) {
	store := &fakeInvalidationCacheStore{}
	eventClient := &fakeEventHeadClient{
		headByCampaign: map[string]uint64{"camp-1": 11},
	}

	afterSeq := campaignProjectionInitialAfterSeq(context.Background(), store, eventClient, "camp-1")
	if afterSeq != 11 {
		t.Fatalf("after seq = %d, want %d", afterSeq, 11)
	}
	if len(eventClient.listRequests) != 1 {
		t.Fatalf("expected one list request, got %d", len(eventClient.listRequests))
	}
}

func TestSelectCampaignIDsForSubscriptionCapsAndRotates(t *testing.T) {
	nextStart := 0
	ids := []string{"camp-1", "camp-2", "camp-3", "camp-4"}

	first := selectCampaignIDsForSubscription(ids, 2, &nextStart)
	if len(first) != 2 || first[0] != "camp-1" || first[1] != "camp-2" {
		t.Fatalf("first = %v, want [camp-1 camp-2]", first)
	}

	second := selectCampaignIDsForSubscription(ids, 2, &nextStart)
	if len(second) != 2 || second[0] != "camp-3" || second[1] != "camp-4" {
		t.Fatalf("second = %v, want [camp-3 camp-4]", second)
	}

	third := selectCampaignIDsForSubscription(ids, 2, &nextStart)
	if len(third) != 2 || third[0] != "camp-1" || third[1] != "camp-2" {
		t.Fatalf("third = %v, want [camp-1 camp-2]", third)
	}
}
