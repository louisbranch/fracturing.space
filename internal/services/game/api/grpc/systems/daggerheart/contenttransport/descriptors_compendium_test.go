package contenttransport

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestDomainCardDescriptorFilterHashSeed(t *testing.T) {
	if got := domainCardDescriptor.filterHashSeed(contentListRequest{}); got != "" {
		t.Fatalf("empty request filter hash seed = %q, want empty", got)
	}

	if got := domainCardDescriptor.filterHashSeed(contentListRequest{DomainID: "dom-1"}); got != "domain_id=dom-1" {
		t.Fatalf("domain request filter hash seed = %q, want %q", got, "domain_id=dom-1")
	}
}

func TestListContentEntriesUsesDomainCardRequestScope(t *testing.T) {
	store := newFakeContentStore()

	items, _, err := listContentEntries(
		context.Background(),
		store,
		contentListRequest{DomainID: "dom-1"},
		commonv1.Locale_LOCALE_UNSPECIFIED,
		domainCardDescriptor,
	)
	if err != nil {
		t.Fatalf("listContentEntries error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("listContentEntries returned %d items, want 1", len(items))
	}
	if items[0].GetId() != "card-1" {
		t.Fatalf("listContentEntries returned id %q, want %q", items[0].GetId(), "card-1")
	}
}
