package auth

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
)

type fakeAccountProfileStore struct {
	profiles map[string]storage.AccountProfile
	putErr   error
	getErr   error
}

func newFakeAccountProfileStore() *fakeAccountProfileStore {
	return &fakeAccountProfileStore{profiles: make(map[string]storage.AccountProfile)}
}

func (s *fakeAccountProfileStore) GetAccountProfile(_ context.Context, userID string) (storage.AccountProfile, error) {
	if s.getErr != nil {
		return storage.AccountProfile{}, s.getErr
	}
	profile, ok := s.profiles[userID]
	if !ok {
		return storage.AccountProfile{}, storage.ErrNotFound
	}
	return profile, nil
}

func (s *fakeAccountProfileStore) PutAccountProfile(_ context.Context, profile storage.AccountProfile) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.profiles == nil {
		s.profiles = make(map[string]storage.AccountProfile)
	}
	s.profiles[profile.UserID] = profile
	return nil
}

func TestGetProfile_NilRequest(t *testing.T) {
	svc := NewAccountService(newFakeAccountProfileStore(), newFakeUserStore())
	_, err := svc.GetProfile(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetProfile_MissingUserID(t *testing.T) {
	svc := NewAccountService(newFakeAccountProfileStore(), newFakeUserStore())
	_, err := svc.GetProfile(context.Background(), &authv1.GetProfileRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetProfile_UserNotFound(t *testing.T) {
	svc := NewAccountService(newFakeAccountProfileStore(), newFakeUserStore())
	_, err := svc.GetProfile(context.Background(), &authv1.GetProfileRequest{UserId: "missing-user"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetProfile_Success(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", Email: "alice"}

	profileStore := newFakeAccountProfileStore()
	profileStore.profiles["user-1"] = storage.AccountProfile{
		UserID:    "user-1",
		Name:      "Alice",
		Locale:    commonv1.Locale_LOCALE_EN_US,
		UpdatedAt: time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
	}
	svc := NewAccountService(profileStore, userStore)

	resp, err := svc.GetProfile(context.Background(), &authv1.GetProfileRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if got := resp.GetProfile(); got == nil {
		t.Fatal("expected profile response")
	} else if got.GetUserId() != "user-1" || got.GetName() != "Alice" {
		t.Fatalf("unexpected profile: %+v", got)
	}
}

func TestUpdateProfile_NilRequest(t *testing.T) {
	svc := NewAccountService(newFakeAccountProfileStore(), newFakeUserStore())
	_, err := svc.UpdateProfile(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateProfile_MissingUserID(t *testing.T) {
	svc := NewAccountService(newFakeAccountProfileStore(), newFakeUserStore())
	_, err := svc.UpdateProfile(context.Background(), &authv1.UpdateProfileRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	svc := NewAccountService(newFakeAccountProfileStore(), newFakeUserStore())
	_, err := svc.UpdateProfile(context.Background(), &authv1.UpdateProfileRequest{UserId: "missing-user"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestUpdateProfile_Success(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", Email: "alice"}
	profileStore := newFakeAccountProfileStore()

	svc := NewAccountService(profileStore, userStore)
	now := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }
	resp, err := svc.UpdateProfile(context.Background(), &authv1.UpdateProfileRequest{
		UserId: "user-1",
		Name:   "  Alice  ",
		Locale: commonv1.Locale_LOCALE_PT_BR,
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if got := resp.GetProfile(); got == nil {
		t.Fatal("expected profile response")
	} else if got.GetName() != "Alice" {
		t.Fatalf("expected normalized name, got %q", got.GetName())
	} else if got.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("expected locale pt-BR, got %v", got.GetLocale())
	} else if got.GetUpdatedAt().AsTime() != now {
		t.Fatalf("expected updated_at %v, got %v", now, got.GetUpdatedAt().AsTime())
	}
}

func TestUpdateProfile_MapsProfileStorageErrors(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", Email: "alice"}
	profileStore := newFakeAccountProfileStore()
	profileStore.putErr = apperrors.Wrap(apperrors.CodeActiveSessionExists, "cannot write account profile", nil)

	svc := NewAccountService(profileStore, userStore)
	_, err := svc.UpdateProfile(context.Background(), &authv1.UpdateProfileRequest{
		UserId: "user-1",
		Name:   "Alice",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}
