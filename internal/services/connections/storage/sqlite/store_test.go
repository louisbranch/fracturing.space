package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/connections/storage"
)

func TestContactRoundTripAndOwnerScoping(t *testing.T) {
	store, err := Open(t.TempDir() + "/connections.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, time.February, 22, 12, 0, 0, 0, time.UTC)
	if err := store.PutContact(context.Background(), storage.Contact{
		OwnerUserID:   "user-1",
		ContactUserID: "user-2",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put contact 1->2: %v", err)
	}
	if err := store.PutContact(context.Background(), storage.Contact{
		OwnerUserID:   "user-1",
		ContactUserID: "user-3",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put contact 1->3: %v", err)
	}
	if err := store.PutContact(context.Background(), storage.Contact{
		OwnerUserID:   "user-2",
		ContactUserID: "user-1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put contact 2->1: %v", err)
	}

	page, err := store.ListContacts(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(page.Contacts) != 2 {
		t.Fatalf("contacts len = %d, want 2", len(page.Contacts))
	}
	for _, contact := range page.Contacts {
		if contact.OwnerUserID != "user-1" {
			t.Fatalf("owner_user_id = %q, want user-1", contact.OwnerUserID)
		}
	}
}
