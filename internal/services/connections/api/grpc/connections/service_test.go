package connections

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"github.com/louisbranch/fracturing.space/internal/services/connections/storage"
)

func TestAddContact_SuccessAndIdempotent(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)
	now := time.Date(2026, time.February, 22, 12, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		resp, err := svc.AddContact(context.Background(), &connectionsv1.AddContactRequest{
			OwnerUserId:   "user-1",
			ContactUserId: "user-2",
		})
		if err != nil {
			t.Fatalf("add contact attempt %d: %v", i+1, err)
		}
		if resp.GetContact() == nil {
			t.Fatal("expected contact response")
		}
	}

	listResp, err := svc.ListContacts(context.Background(), &connectionsv1.ListContactsRequest{
		OwnerUserId: "user-1",
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(listResp.GetContacts()) != 1 {
		t.Fatalf("contacts len = %d, want 1", len(listResp.GetContacts()))
	}
	contact := listResp.GetContacts()[0]
	if contact.GetOwnerUserId() != "user-1" || contact.GetContactUserId() != "user-2" {
		t.Fatalf("unexpected contact: %+v", contact)
	}
	if contact.GetCreatedAt() == nil || contact.GetUpdatedAt() == nil {
		t.Fatal("expected contact timestamps")
	}
}

type fakeContactStore struct {
	contacts map[string]map[string]storage.Contact
}

func newFakeContactStore() *fakeContactStore {
	return &fakeContactStore{contacts: make(map[string]map[string]storage.Contact)}
}

func (s *fakeContactStore) PutContact(_ context.Context, contact storage.Contact) error {
	if _, ok := s.contacts[contact.OwnerUserID]; !ok {
		s.contacts[contact.OwnerUserID] = make(map[string]storage.Contact)
	}
	if existing, ok := s.contacts[contact.OwnerUserID][contact.ContactUserID]; ok {
		contact.CreatedAt = existing.CreatedAt
	}
	s.contacts[contact.OwnerUserID][contact.ContactUserID] = contact
	return nil
}

func (s *fakeContactStore) GetContact(_ context.Context, ownerUserID string, contactUserID string) (storage.Contact, error) {
	byOwner, ok := s.contacts[ownerUserID]
	if !ok {
		return storage.Contact{}, storage.ErrNotFound
	}
	contact, ok := byOwner[contactUserID]
	if !ok {
		return storage.Contact{}, storage.ErrNotFound
	}
	return contact, nil
}

func (s *fakeContactStore) DeleteContact(_ context.Context, ownerUserID string, contactUserID string) error {
	if byOwner, ok := s.contacts[ownerUserID]; ok {
		delete(byOwner, contactUserID)
	}
	return nil
}

func (s *fakeContactStore) ListContacts(_ context.Context, ownerUserID string, pageSize int, pageToken string) (storage.ContactPage, error) {
	if pageSize <= 0 {
		return storage.ContactPage{}, errors.New("page size must be greater than zero")
	}
	byOwner := s.contacts[ownerUserID]
	ids := make([]string, 0, len(byOwner))
	for id := range byOwner {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start := 0
	if pageToken != "" {
		found := false
		for i, id := range ids {
			if id == pageToken {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return storage.ContactPage{}, storage.ErrNotFound
		}
	}

	page := storage.ContactPage{Contacts: make([]storage.Contact, 0, pageSize)}
	for i := start; i < len(ids); i++ {
		if len(page.Contacts) >= pageSize {
			page.NextPageToken = ids[i-1]
			break
		}
		page.Contacts = append(page.Contacts, byOwner[ids[i]])
	}
	return page, nil
}
