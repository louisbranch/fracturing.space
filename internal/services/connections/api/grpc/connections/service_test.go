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

func TestSetUsername_SuccessAndIdempotent(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)
	now := time.Date(2026, time.February, 22, 13, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		resp, err := svc.SetUsername(context.Background(), &connectionsv1.SetUsernameRequest{
			UserId:   "user-1",
			Username: "Alice_One",
		})
		if err != nil {
			t.Fatalf("set username attempt %d: %v", i+1, err)
		}
		if resp.GetUsernameRecord() == nil {
			t.Fatal("expected username record response")
		}
		if got := resp.GetUsernameRecord().GetUsername(); got != "alice_one" {
			t.Fatalf("username = %q, want alice_one", got)
		}
	}

	getResp, err := svc.GetUsername(context.Background(), &connectionsv1.GetUsernameRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("get username: %v", err)
	}
	if got := getResp.GetUsernameRecord().GetUsername(); got != "alice_one" {
		t.Fatalf("get username = %q, want alice_one", got)
	}

	lookupResp, err := svc.LookupUsername(context.Background(), &connectionsv1.LookupUsernameRequest{
		Username: "ALICE_ONE",
	})
	if err != nil {
		t.Fatalf("lookup username: %v", err)
	}
	if got := lookupResp.GetUsernameRecord().GetUserId(); got != "user-1" {
		t.Fatalf("lookup user_id = %q, want user-1", got)
	}
}

func TestSetUsername_SameCanonicalValueDoesNotChangeTimestamps(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)
	initial := time.Date(2026, time.February, 22, 13, 0, 0, 0, time.UTC)
	retryAt := initial.Add(2 * time.Hour)

	svc.clock = func() time.Time { return initial }
	first, err := svc.SetUsername(context.Background(), &connectionsv1.SetUsernameRequest{
		UserId:   "user-1",
		Username: "Alice_One",
	})
	if err != nil {
		t.Fatalf("set initial username: %v", err)
	}

	svc.clock = func() time.Time { return retryAt }
	second, err := svc.SetUsername(context.Background(), &connectionsv1.SetUsernameRequest{
		UserId:   "user-1",
		Username: "ALICE_ONE",
	})
	if err != nil {
		t.Fatalf("set repeated username: %v", err)
	}

	firstRecord := first.GetUsernameRecord()
	secondRecord := second.GetUsernameRecord()
	if firstRecord == nil || secondRecord == nil {
		t.Fatal("expected username record in both responses")
	}
	if !secondRecord.GetCreatedAt().AsTime().Equal(firstRecord.GetCreatedAt().AsTime()) {
		t.Fatalf("created_at changed: got %v want %v", secondRecord.GetCreatedAt().AsTime(), firstRecord.GetCreatedAt().AsTime())
	}
	if !secondRecord.GetUpdatedAt().AsTime().Equal(firstRecord.GetUpdatedAt().AsTime()) {
		t.Fatalf("updated_at changed: got %v want %v", secondRecord.GetUpdatedAt().AsTime(), firstRecord.GetUpdatedAt().AsTime())
	}
}

func TestSetUsername_InvalidUsernameReturnsInvalidArgument(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.SetUsername(context.Background(), &connectionsv1.SetUsernameRequest{
		UserId:   "user-1",
		Username: "__",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestSetUsername_ConflictReturnsAlreadyExists(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.SetUsername(context.Background(), &connectionsv1.SetUsernameRequest{
		UserId:   "user-1",
		Username: "conflict",
	})
	if err != nil {
		t.Fatalf("set username user-1: %v", err)
	}

	_, err = svc.SetUsername(context.Background(), &connectionsv1.SetUsernameRequest{
		UserId:   "user-2",
		Username: "Conflict",
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.AlreadyExists)
	}
}

func TestGetUsername_NotFoundReturnsNotFound(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.GetUsername(context.Background(), &connectionsv1.GetUsernameRequest{UserId: "missing-user"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestLookupUsername_NotFoundReturnsNotFound(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.LookupUsername(context.Background(), &connectionsv1.LookupUsernameRequest{Username: "missing-user"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestSetPublicProfile_SuccessAndLookupByUsername(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)
	now := time.Date(2026, time.February, 22, 14, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	_, err := svc.SetUsername(context.Background(), &connectionsv1.SetUsernameRequest{
		UserId:   "user-1",
		Username: "alice_one",
	})
	if err != nil {
		t.Fatalf("set username: %v", err)
	}

	setResp, err := svc.SetPublicProfile(context.Background(), &connectionsv1.SetPublicProfileRequest{
		UserId:        "user-1",
		Name:          "Alice",
		AvatarSetId:   "avatar_set_v1",
		AvatarAssetId: "001",
		Bio:           "Campaign manager",
	})
	if err != nil {
		t.Fatalf("set public profile: %v", err)
	}
	if setResp.GetPublicProfileRecord() == nil {
		t.Fatal("expected public profile record response")
	}

	getResp, err := svc.GetPublicProfile(context.Background(), &connectionsv1.GetPublicProfileRequest{
		UserId: "user-1",
	})
	if err != nil {
		t.Fatalf("get public profile: %v", err)
	}
	if got := getResp.GetPublicProfileRecord().GetName(); got != "Alice" {
		t.Fatalf("name = %q, want Alice", got)
	}

	lookupResp, err := svc.LookupPublicProfile(context.Background(), &connectionsv1.LookupPublicProfileRequest{
		Username: "ALICE_ONE",
	})
	if err != nil {
		t.Fatalf("lookup public profile: %v", err)
	}
	if got := lookupResp.GetUsernameRecord().GetUserId(); got != "user-1" {
		t.Fatalf("lookup user_id = %q, want user-1", got)
	}
	if got := lookupResp.GetPublicProfileRecord().GetBio(); got != "Campaign manager" {
		t.Fatalf("lookup bio = %q, want Campaign manager", got)
	}
}

func TestSetPublicProfile_InvalidAvatarReturnsInvalidArgument(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.SetPublicProfile(context.Background(), &connectionsv1.SetPublicProfileRequest{
		UserId:        "user-1",
		Name:          "Alice",
		AvatarSetId:   "missing-set",
		AvatarAssetId: "001",
		Bio:           "Campaign manager",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestGetPublicProfile_NotFoundReturnsNotFound(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.GetPublicProfile(context.Background(), &connectionsv1.GetPublicProfileRequest{
		UserId: "missing-user",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestLookupPublicProfile_UsernameFoundWithoutProfileReturnsUsernameOnly(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.SetUsername(context.Background(), &connectionsv1.SetUsernameRequest{
		UserId:   "user-1",
		Username: "alice_one",
	})
	if err != nil {
		t.Fatalf("set username: %v", err)
	}

	resp, err := svc.LookupPublicProfile(context.Background(), &connectionsv1.LookupPublicProfileRequest{
		Username: "ALICE_ONE",
	})
	if err != nil {
		t.Fatalf("lookup public profile: %v", err)
	}
	if resp.GetUsernameRecord() == nil {
		t.Fatal("expected username record")
	}
	if got := resp.GetUsernameRecord().GetUserId(); got != "user-1" {
		t.Fatalf("lookup user_id = %q, want user-1", got)
	}
	if resp.GetPublicProfileRecord() != nil {
		t.Fatalf("public profile = %+v, want nil when no profile exists", resp.GetPublicProfileRecord())
	}
}

func TestLookupPublicProfile_InvalidUsernameReturnsInvalidArgument(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.LookupPublicProfile(context.Background(), &connectionsv1.LookupPublicProfileRequest{
		Username: "bad username",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

type fakeContactStore struct {
	contacts         map[string]map[string]storage.Contact
	usernamesByUser  map[string]storage.UsernameRecord
	usernamesByValue map[string]string
	publicProfiles   map[string]storage.PublicProfileRecord
}

func newFakeContactStore() *fakeContactStore {
	return &fakeContactStore{
		contacts:         make(map[string]map[string]storage.Contact),
		usernamesByUser:  make(map[string]storage.UsernameRecord),
		usernamesByValue: make(map[string]string),
		publicProfiles:   make(map[string]storage.PublicProfileRecord),
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

func (s *fakeContactStore) PutUsername(_ context.Context, username storage.UsernameRecord) error {
	canonical := strings.TrimSpace(strings.ToLower(username.Username))
	if canonical == "" {
		return errors.New("username is required")
	}
	if owner, ok := s.usernamesByValue[canonical]; ok && owner != username.UserID {
		return storage.ErrAlreadyExists
	}
	if existing, ok := s.usernamesByUser[username.UserID]; ok {
		if existing.Username == canonical {
			username.CreatedAt = existing.CreatedAt
			username.UpdatedAt = existing.UpdatedAt
		} else {
			delete(s.usernamesByValue, existing.Username)
			username.CreatedAt = existing.CreatedAt
		}
	}
	username.Username = canonical
	s.usernamesByUser[username.UserID] = username
	s.usernamesByValue[canonical] = username.UserID
	return nil
}

func (s *fakeContactStore) GetUsernameByUserID(_ context.Context, userID string) (storage.UsernameRecord, error) {
	record, ok := s.usernamesByUser[userID]
	if !ok {
		return storage.UsernameRecord{}, storage.ErrNotFound
	}
	return record, nil
}

func (s *fakeContactStore) GetUsernameByUsername(_ context.Context, username string) (storage.UsernameRecord, error) {
	canonical := strings.TrimSpace(strings.ToLower(username))
	userID, ok := s.usernamesByValue[canonical]
	if !ok {
		return storage.UsernameRecord{}, storage.ErrNotFound
	}
	record, ok := s.usernamesByUser[userID]
	if !ok {
		return storage.UsernameRecord{}, storage.ErrNotFound
	}
	return record, nil
}

func (s *fakeContactStore) PutPublicProfile(_ context.Context, profile storage.PublicProfileRecord) error {
	userID := strings.TrimSpace(profile.UserID)
	if userID == "" {
		return errors.New("user id is required")
	}
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		return errors.New("name is required")
	}
	if existing, ok := s.publicProfiles[userID]; ok {
		profile.CreatedAt = existing.CreatedAt
	}
	if profile.CreatedAt.IsZero() && !profile.UpdatedAt.IsZero() {
		profile.CreatedAt = profile.UpdatedAt
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = profile.CreatedAt
	}
	s.publicProfiles[userID] = storage.PublicProfileRecord{
		UserID:        userID,
		Name:          name,
		AvatarSetID:   strings.TrimSpace(profile.AvatarSetID),
		AvatarAssetID: strings.TrimSpace(profile.AvatarAssetID),
		Bio:           strings.TrimSpace(profile.Bio),
		CreatedAt:     profile.CreatedAt,
		UpdatedAt:     profile.UpdatedAt,
	}
	return nil
}

func (s *fakeContactStore) GetPublicProfileByUserID(_ context.Context, userID string) (storage.PublicProfileRecord, error) {
	record, ok := s.publicProfiles[strings.TrimSpace(userID)]
	if !ok {
		return storage.PublicProfileRecord{}, storage.ErrNotFound
	}
	return record, nil
}
