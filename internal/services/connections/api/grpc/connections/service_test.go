package connections

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"github.com/louisbranch/fracturing.space/internal/services/connections/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestSetUserProfile_SuccessAndLookup(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)
	now := time.Date(2026, time.February, 22, 13, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		setResp, err := svc.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
			UserId:        "user-1",
			Username:      "Alice_One",
			Name:          "Alice",
			AvatarSetId:   "avatar_set_v1",
			AvatarAssetId: "001",
			Bio:           "Campaign manager",
		})
		if err != nil {
			t.Fatalf("set user profile attempt %d: %v", i+1, err)
		}
		if setResp.GetUserProfileRecord() == nil {
			t.Fatal("expected user profile record")
		}
		if got := setResp.GetUserProfileRecord().GetUsername(); got != "alice_one" {
			t.Fatalf("username = %q, want alice_one", got)
		}
	}

	getResp, err := svc.GetUserProfile(context.Background(), &connectionsv1.GetUserProfileRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("get user profile: %v", err)
	}
	if got := getResp.GetUserProfileRecord().GetName(); got != "Alice" {
		t.Fatalf("name = %q, want Alice", got)
	}

	lookupResp, err := svc.LookupUserProfile(context.Background(), &connectionsv1.LookupUserProfileRequest{
		Username: "ALICE_ONE",
	})
	if err != nil {
		t.Fatalf("lookup user profile: %v", err)
	}
	if got := lookupResp.GetUserProfileRecord().GetUserId(); got != "user-1" {
		t.Fatalf("user_id = %q, want user-1", got)
	}
	if got := lookupResp.GetUserProfileRecord().GetBio(); got != "Campaign manager" {
		t.Fatalf("bio = %q, want Campaign manager", got)
	}
}

func TestSetUserProfile_SameCanonicalValueDoesNotChangeTimestamps(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)
	initial := time.Date(2026, time.February, 22, 13, 0, 0, 0, time.UTC)
	retryAt := initial.Add(2 * time.Hour)

	svc.clock = func() time.Time { return initial }
	first, err := svc.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
		UserId:        "user-1",
		Username:      "Alice_One",
		Name:          "Alice",
		AvatarSetId:   "avatar_set_v1",
		AvatarAssetId: "001",
		Bio:           "Campaign manager",
	})
	if err != nil {
		t.Fatalf("set initial user profile: %v", err)
	}

	svc.clock = func() time.Time { return retryAt }
	second, err := svc.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
		UserId:        "user-1",
		Username:      "ALICE_ONE",
		Name:          "Alice",
		AvatarSetId:   "avatar_set_v1",
		AvatarAssetId: "001",
		Bio:           "Campaign manager",
	})
	if err != nil {
		t.Fatalf("set repeated user profile: %v", err)
	}

	firstRecord := first.GetUserProfileRecord()
	secondRecord := second.GetUserProfileRecord()
	if firstRecord == nil || secondRecord == nil {
		t.Fatal("expected user profile record in both responses")
	}
	if !secondRecord.GetCreatedAt().AsTime().Equal(firstRecord.GetCreatedAt().AsTime()) {
		t.Fatalf("created_at changed: got %v want %v", secondRecord.GetCreatedAt().AsTime(), firstRecord.GetCreatedAt().AsTime())
	}
	if !secondRecord.GetUpdatedAt().AsTime().Equal(firstRecord.GetUpdatedAt().AsTime()) {
		t.Fatalf("updated_at changed: got %v want %v", secondRecord.GetUpdatedAt().AsTime(), firstRecord.GetUpdatedAt().AsTime())
	}
}

func TestSetUserProfile_InvalidUsernameReturnsInvalidArgument(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
		UserId:   "user-1",
		Username: "__",
		Name:     "Alice",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestSetUserProfile_MissingNameReturnsInvalidArgument(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
		UserId:   "user-1",
		Username: "alice_one",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestSetUserProfile_InvalidAvatarReturnsInvalidArgument(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
		UserId:        "user-1",
		Username:      "alice_one",
		Name:          "Alice",
		AvatarSetId:   "missing-set",
		AvatarAssetId: "001",
		Bio:           "Campaign manager",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestSetUserProfile_ConflictReturnsAlreadyExists(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
		UserId:   "user-1",
		Username: "conflict",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("set user profile user-1: %v", err)
	}

	_, err = svc.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
		UserId:   "user-2",
		Username: "Conflict",
		Name:     "Bob",
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.AlreadyExists)
	}
}

func TestGetUserProfile_NotFoundReturnsNotFound(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.GetUserProfile(context.Background(), &connectionsv1.GetUserProfileRequest{UserId: "missing-user"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestLookupUserProfile_NotFoundReturnsNotFound(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.LookupUserProfile(context.Background(), &connectionsv1.LookupUserProfileRequest{Username: "missing-user"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

type fakeContactStore struct {
	contacts               map[string]map[string]storage.Contact
	profilesByUser         map[string]storage.UserProfileRecord
	profileOwnerByUsername map[string]string
}

func newFakeContactStore() *fakeContactStore {
	return &fakeContactStore{
		contacts:               make(map[string]map[string]storage.Contact),
		profilesByUser:         make(map[string]storage.UserProfileRecord),
		profileOwnerByUsername: make(map[string]string),
	}
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

func (s *fakeContactStore) PutUserProfile(_ context.Context, profile storage.UserProfileRecord) error {
	userID := strings.TrimSpace(profile.UserID)
	if userID == "" {
		return errors.New("user id is required")
	}
	canonicalUsername := strings.TrimSpace(strings.ToLower(profile.Username))
	if canonicalUsername == "" {
		return errors.New("username is required")
	}
	if owner, ok := s.profileOwnerByUsername[canonicalUsername]; ok && owner != userID {
		return storage.ErrAlreadyExists
	}
	if existing, ok := s.profilesByUser[userID]; ok {
		if existing.Username == canonicalUsername &&
			existing.Name == strings.TrimSpace(profile.Name) &&
			existing.AvatarSetID == strings.TrimSpace(profile.AvatarSetID) &&
			existing.AvatarAssetID == strings.TrimSpace(profile.AvatarAssetID) &&
			existing.Bio == strings.TrimSpace(profile.Bio) {
			profile.CreatedAt = existing.CreatedAt
			profile.UpdatedAt = existing.UpdatedAt
		} else {
			delete(s.profileOwnerByUsername, existing.Username)
			profile.CreatedAt = existing.CreatedAt
		}
	}
	profile.UserID = userID
	profile.Username = canonicalUsername
	profile.Name = strings.TrimSpace(profile.Name)
	profile.AvatarSetID = strings.TrimSpace(profile.AvatarSetID)
	profile.AvatarAssetID = strings.TrimSpace(profile.AvatarAssetID)
	profile.Bio = strings.TrimSpace(profile.Bio)
	s.profilesByUser[userID] = profile
	s.profileOwnerByUsername[canonicalUsername] = userID
	return nil
}

func (s *fakeContactStore) GetUserProfileByUserID(_ context.Context, userID string) (storage.UserProfileRecord, error) {
	record, ok := s.profilesByUser[strings.TrimSpace(userID)]
	if !ok {
		return storage.UserProfileRecord{}, storage.ErrNotFound
	}
	return record, nil
}

func (s *fakeContactStore) GetUserProfileByUsername(_ context.Context, username string) (storage.UserProfileRecord, error) {
	canonical := strings.TrimSpace(strings.ToLower(username))
	userID, ok := s.profileOwnerByUsername[canonical]
	if !ok {
		return storage.UserProfileRecord{}, storage.ErrNotFound
	}
	record, ok := s.profilesByUser[userID]
	if !ok {
		return storage.UserProfileRecord{}, storage.ErrNotFound
	}
	return record, nil
}
