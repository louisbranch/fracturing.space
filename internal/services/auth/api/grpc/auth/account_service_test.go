package auth

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestAccountProfileProtoRemovesSocialFields(t *testing.T) {
	assertMissingField := func(message protoreflect.ProtoMessage, messageName string, fieldName string) {
		t.Helper()
		fields := message.ProtoReflect().Descriptor().Fields()
		if fields.ByName(protoreflect.Name(fieldName)) != nil {
			t.Fatalf("%s unexpectedly has field %q", messageName, fieldName)
		}
	}

	assertMissingField(&authv1.AccountProfile{}, "AccountProfile", "name")
	assertMissingField(&authv1.AccountProfile{}, "AccountProfile", "avatar_set_id")
	assertMissingField(&authv1.AccountProfile{}, "AccountProfile", "avatar_asset_id")
	assertMissingField(&authv1.UpdateProfileRequest{}, "UpdateProfileRequest", "name")
	assertMissingField(&authv1.UpdateProfileRequest{}, "UpdateProfileRequest", "avatar_set_id")
	assertMissingField(&authv1.UpdateProfileRequest{}, "UpdateProfileRequest", "avatar_asset_id")
}

func TestGetProfile_Success(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{
		ID:        "user-1",
		Username:  "alice",
		Locale:    commonv1.Locale_LOCALE_PT_BR,
		CreatedAt: now,
		UpdatedAt: now,
	}

	svc := NewAccountService(userStore)
	resp, err := svc.GetProfile(context.Background(), &authv1.GetProfileRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if got := resp.GetProfile().GetUsername(); got != "alice" {
		t.Fatalf("username = %q, want %q", got, "alice")
	}
	if got := resp.GetProfile().GetLocale(); got != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("locale = %v, want %v", got, commonv1.Locale_LOCALE_PT_BR)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	userStore := newFakeUserStore()
	createdAt := time.Date(2026, 1, 23, 8, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{
		ID:        "user-1",
		Username:  "alice",
		Locale:    commonv1.Locale_LOCALE_EN_US,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	now := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	svc := NewAccountService(userStore)
	svc.clock = func() time.Time { return now }

	resp, err := svc.UpdateProfile(context.Background(), &authv1.UpdateProfileRequest{
		UserId: "user-1",
		Locale: commonv1.Locale_LOCALE_PT_BR,
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if got := resp.GetProfile().GetUsername(); got != "alice" {
		t.Fatalf("username = %q, want %q", got, "alice")
	}
	if got := resp.GetProfile().GetLocale(); got != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("locale = %v, want %v", got, commonv1.Locale_LOCALE_PT_BR)
	}
	if got := resp.GetProfile().GetUpdatedAt().AsTime(); got != now {
		t.Fatalf("updated_at = %v, want %v", got, now)
	}
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	svc := NewAccountService(newFakeUserStore())
	_, err := svc.UpdateProfile(context.Background(), &authv1.UpdateProfileRequest{UserId: "missing"})
	assertStatusCode(t, err, codes.NotFound)
}
