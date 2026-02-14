package auth

import (
	"context"
	"net/url"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
)

func TestGenerateMagicLinkAndConsume(t *testing.T) {
	store := openTempAuthStore(t)
	oauthStore := oauth.NewStore(store.DB())

	userRecord := user.User{
		ID:          "user-1",
		DisplayName: "Alpha",
		Locale:      platformi18n.DefaultLocale(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.PutUser(context.Background(), userRecord); err != nil {
		t.Fatalf("put user: %v", err)
	}

	svc := NewAuthService(store, store, oauthStore)
	svc.magicLinkConfig.BaseURL = "http://web.local/magic"
	svc.magicLinkConfig.TTL = 10 * time.Minute
	fixed := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return fixed }

	resp, err := svc.GenerateMagicLink(context.Background(), &authv1.GenerateMagicLinkRequest{
		UserId: "user-1",
		Email:  "alpha@example.com",
	})
	if err != nil {
		t.Fatalf("generate magic link: %v", err)
	}
	if resp.GetMagicLinkUrl() == "" {
		t.Fatalf("expected magic link url")
	}

	parsed, err := url.Parse(resp.GetMagicLinkUrl())
	if err != nil {
		t.Fatalf("parse magic url: %v", err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatalf("expected token in url")
	}

	consumeResp, err := svc.ConsumeMagicLink(context.Background(), &authv1.ConsumeMagicLinkRequest{Token: token})
	if err != nil {
		t.Fatalf("consume magic link: %v", err)
	}
	if consumeResp.GetUser() == nil || consumeResp.GetUser().GetId() != "user-1" {
		t.Fatalf("expected user in response")
	}

	storedEmail, err := store.GetUserEmailByEmail(context.Background(), "alpha@example.com")
	if err != nil {
		t.Fatalf("get email: %v", err)
	}
	if storedEmail.VerifiedAt == nil {
		t.Fatalf("expected verified email")
	}
}

func TestConsumeMagicLinkExpired(t *testing.T) {
	store := openTempAuthStore(t)
	svc := NewAuthService(store, store, nil)
	fixed := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return fixed }

	if err := store.PutMagicLink(context.Background(), storage.MagicLink{
		Token:     "token-1",
		UserID:    "user-1",
		Email:     "alpha@example.com",
		CreatedAt: fixed.Add(-2 * time.Hour),
		ExpiresAt: fixed.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("put magic link: %v", err)
	}

	_, err := svc.ConsumeMagicLink(context.Background(), &authv1.ConsumeMagicLinkRequest{Token: "token-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListUserEmails(t *testing.T) {
	store := openTempAuthStore(t)
	userRecord := user.User{
		ID:          "user-1",
		DisplayName: "Alpha",
		Locale:      platformi18n.DefaultLocale(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.PutUser(context.Background(), userRecord); err != nil {
		t.Fatalf("put user: %v", err)
	}
	if err := store.PutUserEmail(context.Background(), storage.UserEmail{
		ID:        "email-1",
		UserID:    "user-1",
		Email:     "alpha@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("put email: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	resp, err := svc.ListUserEmails(context.Background(), &authv1.ListUserEmailsRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("list user emails: %v", err)
	}
	if len(resp.GetEmails()) != 1 {
		t.Fatalf("expected 1 email, got %d", len(resp.GetEmails()))
	}
}

func openTempAuthStore(t *testing.T) *authsqlite.Store {
	t.Helper()
	path := t.TempDir() + "/auth.db"
	store, err := authsqlite.Open(path)
	if err != nil {
		t.Fatalf("open auth store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}
