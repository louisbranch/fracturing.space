package social

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"github.com/louisbranch/fracturing.space/internal/services/social/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestUserProfileResponsesUseUserProfileNaming(t *testing.T) {
	assertUserProfileField := func(message protoreflect.ProtoMessage, responseName string) {
		t.Helper()
		fields := message.ProtoReflect().Descriptor().Fields()
		field := fields.ByName(protoreflect.Name("user_profile"))
		if field == nil {
			t.Fatalf("%s missing user_profile field", responseName)
		}
		if got := string(field.Message().Name()); got != "UserProfile" {
			t.Fatalf("%s.user_profile message = %q, want %q", responseName, got, "UserProfile")
		}
	}

	assertUserProfileField(&socialv1.SetUserProfileResponse{}, "SetUserProfileResponse")
	assertUserProfileField(&socialv1.GetUserProfileResponse{}, "GetUserProfileResponse")
}

func TestAddContact_SuccessAndIdempotent(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)
	now := time.Date(2026, time.February, 22, 12, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		resp, err := svc.AddContact(context.Background(), &socialv1.AddContactRequest{
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

	listResp, err := svc.ListContacts(context.Background(), &socialv1.ListContactsRequest{
		OwnerUserId: "user-1",
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(listResp.GetContacts()) != 1 {
		t.Fatalf("contacts len = %d, want 1", len(listResp.GetContacts()))
	}
}

func TestSetUserProfile_SuccessAndGet(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)
	now := time.Date(2026, time.February, 22, 13, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		setResp, err := svc.SetUserProfile(context.Background(), &socialv1.SetUserProfileRequest{
			UserId:        "user-1",
			Name:          "Alice",
			AvatarSetId:   "avatar_set_v1",
			AvatarAssetId: "apothecary_journeyman",
			Bio:           "Campaign manager",
			Pronouns:      sharedpronouns.ToProto("she/her"),
		})
		if err != nil {
			t.Fatalf("set user profile attempt %d: %v", i+1, err)
		}
		if setResp.GetUserProfile() == nil {
			t.Fatal("expected user profile record")
		}
		if got := setResp.GetUserProfile().GetName(); got != "Alice" {
			t.Fatalf("name = %q, want Alice", got)
		}
	}

	getResp, err := svc.GetUserProfile(context.Background(), &socialv1.GetUserProfileRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("get user profile: %v", err)
	}
	if got := getResp.GetUserProfile().GetBio(); got != "Campaign manager" {
		t.Fatalf("bio = %q, want Campaign manager", got)
	}
}

func TestSetUserProfile_SameValueDoesNotChangeTimestamps(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)
	initial := time.Date(2026, time.February, 22, 13, 0, 0, 0, time.UTC)
	retryAt := initial.Add(2 * time.Hour)

	svc.clock = func() time.Time { return initial }
	first, err := svc.SetUserProfile(context.Background(), &socialv1.SetUserProfileRequest{
		UserId:        "user-1",
		Name:          "Alice",
		AvatarSetId:   "avatar_set_v1",
		AvatarAssetId: "apothecary_journeyman",
		Bio:           "Campaign manager",
		Pronouns:      sharedpronouns.ToProto("she/her"),
	})
	if err != nil {
		t.Fatalf("set initial user profile: %v", err)
	}

	svc.clock = func() time.Time { return retryAt }
	second, err := svc.SetUserProfile(context.Background(), &socialv1.SetUserProfileRequest{
		UserId:        "user-1",
		Name:          "Alice",
		AvatarSetId:   "avatar_set_v1",
		AvatarAssetId: "apothecary_journeyman",
		Bio:           "Campaign manager",
		Pronouns:      sharedpronouns.ToProto("she/her"),
	})
	if err != nil {
		t.Fatalf("set repeated user profile: %v", err)
	}

	firstRecord := first.GetUserProfile()
	secondRecord := second.GetUserProfile()
	if !secondRecord.GetCreatedAt().AsTime().Equal(firstRecord.GetCreatedAt().AsTime()) {
		t.Fatalf("created_at changed: got %v want %v", secondRecord.GetCreatedAt().AsTime(), firstRecord.GetCreatedAt().AsTime())
	}
	if !secondRecord.GetUpdatedAt().AsTime().Equal(firstRecord.GetUpdatedAt().AsTime()) {
		t.Fatalf("updated_at changed: got %v want %v", secondRecord.GetUpdatedAt().AsTime(), firstRecord.GetUpdatedAt().AsTime())
	}
}

func TestSetUserProfile_AllowsMissingName(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	resp, err := svc.SetUserProfile(context.Background(), &socialv1.SetUserProfileRequest{
		UserId: "user-1",
	})
	if err != nil {
		t.Fatalf("set user profile: %v", err)
	}
	if got := resp.GetUserProfile().GetName(); got != "" {
		t.Fatalf("name = %q, want empty", got)
	}
	if got := resp.GetUserProfile().GetAvatarSetId(); got != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", got, assetcatalog.AvatarSetPeopleV1)
	}
}

func TestSetUserProfile_InvalidAvatarReturnsInvalidArgument(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.SetUserProfile(context.Background(), &socialv1.SetUserProfileRequest{
		UserId:        "user-1",
		Name:          "Alice",
		AvatarSetId:   "missing-set",
		AvatarAssetId: "apothecary_journeyman",
		Bio:           "Campaign manager",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestGetUserProfile_NotFoundReturnsNotFound(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	_, err := svc.GetUserProfile(context.Background(), &socialv1.GetUserProfileRequest{UserId: "missing-user"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestSyncDirectoryUser_StoresCanonicalUsername(t *testing.T) {
	store := newFakeContactStore()
	svc := NewService(store)

	if _, err := svc.SyncDirectoryUser(context.Background(), &socialv1.SyncDirectoryUserRequest{
		UserId:   "user-1",
		Username: "  ALIce  ",
	}); err != nil {
		t.Fatalf("sync directory user: %v", err)
	}
	if got := store.directory["user-1"].Username; got != "alice" {
		t.Fatalf("username = %q, want %q", got, "alice")
	}
}

func TestSearchUsers_RanksContactsFirst(t *testing.T) {
	store := newFakeContactStore()
	_ = store.PutDirectoryUser(context.Background(), storage.DirectoryUser{UserID: "user-2", Username: "alice"})
	_ = store.PutDirectoryUser(context.Background(), storage.DirectoryUser{UserID: "user-3", Username: "alina"})
	_ = store.PutDirectoryUser(context.Background(), storage.DirectoryUser{UserID: "user-4", Username: "alfred"})
	_ = store.PutUserProfile(context.Background(), storage.UserProfile{UserID: "user-2", Name: "Alice"})
	_ = store.PutUserProfile(context.Background(), storage.UserProfile{UserID: "user-3", Name: "Alina"})
	_ = store.PutContact(context.Background(), storage.Contact{OwnerUserID: "viewer-1", ContactUserID: "user-3"})

	svc := NewService(store)
	resp, err := svc.SearchUsers(context.Background(), &socialv1.SearchUsersRequest{
		ViewerUserId: "viewer-1",
		Query:        "al",
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("search users: %v", err)
	}
	if len(resp.GetUsers()) != 3 {
		t.Fatalf("users len = %d, want 3", len(resp.GetUsers()))
	}
	if got := resp.GetUsers()[0].GetUsername(); got != "alina" {
		t.Fatalf("first username = %q, want %q", got, "alina")
	}
	if !resp.GetUsers()[0].GetIsContact() {
		t.Fatal("expected first result to be a contact")
	}
}

func TestSearchUsers_ShortQueryReturnsEmpty(t *testing.T) {
	svc := NewService(newFakeContactStore())
	resp, err := svc.SearchUsers(context.Background(), &socialv1.SearchUsersRequest{
		ViewerUserId: "viewer-1",
		Query:        "a",
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("search users: %v", err)
	}
	if len(resp.GetUsers()) != 0 {
		t.Fatalf("users len = %d, want 0", len(resp.GetUsers()))
	}
}

type fakeContactStore struct {
	contacts  map[string]storage.Contact
	profiles  map[string]storage.UserProfile
	directory map[string]storage.DirectoryUser
}

func newFakeContactStore() *fakeContactStore {
	return &fakeContactStore{
		contacts:  map[string]storage.Contact{},
		profiles:  map[string]storage.UserProfile{},
		directory: map[string]storage.DirectoryUser{},
	}
}

func (s *fakeContactStore) PutContact(_ context.Context, contact storage.Contact) error {
	key := contact.OwnerUserID + "|" + contact.ContactUserID
	if existing, ok := s.contacts[key]; ok {
		contact.CreatedAt = existing.CreatedAt
	}
	s.contacts[key] = contact
	return nil
}

func (s *fakeContactStore) GetContact(_ context.Context, ownerUserID string, contactUserID string) (storage.Contact, error) {
	contact, ok := s.contacts[ownerUserID+"|"+contactUserID]
	if !ok {
		return storage.Contact{}, storage.ErrNotFound
	}
	return contact, nil
}

func (s *fakeContactStore) DeleteContact(_ context.Context, ownerUserID string, contactUserID string) error {
	delete(s.contacts, ownerUserID+"|"+contactUserID)
	return nil
}

func (s *fakeContactStore) ListContacts(_ context.Context, ownerUserID string, pageSize int, pageToken string) (storage.ContactPage, error) {
	records := make([]storage.Contact, 0, len(s.contacts))
	for _, contact := range s.contacts {
		if contact.OwnerUserID == ownerUserID {
			records = append(records, contact)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].ContactUserID < records[j].ContactUserID
	})

	start := 0
	if pageToken != "" {
		for i, contact := range records {
			if contact.ContactUserID == pageToken {
				start = i + 1
				break
			}
		}
	}

	if pageSize <= 0 {
		pageSize = len(records)
	}
	end := start + pageSize
	page := storage.ContactPage{}
	if end < len(records) {
		page.NextPageToken = records[end-1].ContactUserID
		page.Contacts = append(page.Contacts, records[start:end]...)
		return page, nil
	}
	page.Contacts = append(page.Contacts, records[start:]...)
	return page, nil
}

func (s *fakeContactStore) PutUserProfile(_ context.Context, profile storage.UserProfile) error {
	existing, ok := s.profiles[profile.UserID]
	if ok &&
		strings.TrimSpace(existing.Name) == strings.TrimSpace(profile.Name) &&
		strings.TrimSpace(existing.AvatarSetID) == strings.TrimSpace(profile.AvatarSetID) &&
		strings.TrimSpace(existing.AvatarAssetID) == strings.TrimSpace(profile.AvatarAssetID) &&
		strings.TrimSpace(existing.Bio) == strings.TrimSpace(profile.Bio) &&
		strings.TrimSpace(existing.Pronouns) == strings.TrimSpace(profile.Pronouns) {
		return nil
	}
	if ok {
		profile.CreatedAt = existing.CreatedAt
	}
	s.profiles[profile.UserID] = profile
	return nil
}

func (s *fakeContactStore) PutDirectoryUser(_ context.Context, user storage.DirectoryUser) error {
	if existing, ok := s.directory[user.UserID]; ok && strings.TrimSpace(existing.Username) == strings.TrimSpace(user.Username) {
		user.CreatedAt = existing.CreatedAt
		user.UpdatedAt = existing.UpdatedAt
	}
	s.directory[user.UserID] = user
	return nil
}

func (s *fakeContactStore) SearchUsers(_ context.Context, viewerUserID string, query string, limit int) ([]storage.SearchUser, error) {
	results := make([]storage.SearchUser, 0, len(s.directory))
	query = strings.ToLower(strings.TrimSpace(query))
	for _, entry := range s.directory {
		name := ""
		avatarSetID := ""
		avatarAssetID := ""
		if profile, ok := s.profiles[entry.UserID]; ok {
			name = profile.Name
			avatarSetID = profile.AvatarSetID
			avatarAssetID = profile.AvatarAssetID
		}
		usernameMatch := strings.HasPrefix(strings.ToLower(entry.Username), query)
		nameMatch := strings.HasPrefix(strings.ToLower(strings.TrimSpace(name)), query)
		if !usernameMatch && !nameMatch {
			continue
		}
		_, isContact := s.contacts[viewerUserID+"|"+entry.UserID]
		results = append(results, storage.SearchUser{
			UserID:        entry.UserID,
			Username:      entry.Username,
			Name:          name,
			AvatarSetID:   avatarSetID,
			AvatarAssetID: avatarAssetID,
			IsContact:     isContact,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].IsContact != results[j].IsContact {
			return results[i].IsContact
		}
		return results[i].Username < results[j].Username
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (s *fakeContactStore) GetUserProfileByUserID(_ context.Context, userID string) (storage.UserProfile, error) {
	profile, ok := s.profiles[userID]
	if !ok {
		return storage.UserProfile{}, storage.ErrNotFound
	}
	return profile, nil
}
