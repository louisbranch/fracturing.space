package testkit

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	authstorage "github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	authuser "github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CreateAuthUser persists a test auth user directly into the auth store and
// optionally verifies the running auth server can resolve it.
func CreateAuthUser(t *testing.T, authAddr, username string) string {
	t.Helper()

	authDBPath := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AUTH_DB_PATH"))
	if authDBPath == "" {
		t.Fatal("FRACTURING_SPACE_AUTH_DB_PATH is required")
	}

	store, err := authsqlite.Open(authDBPath)
	if err != nil {
		t.Fatalf("open auth store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	existing, err := store.GetUserByUsername(ctx, username)
	if err == nil {
		return existing.ID
	}
	if !errors.Is(err, authstorage.ErrNotFound) {
		t.Fatalf("lookup auth user %q: %v", username, err)
	}

	created, err := authuser.CreateUser(authuser.CreateUserInput{Username: username}, nil, nil)
	if err != nil {
		t.Fatalf("create auth user: %v", err)
	}
	if err := store.PutUser(ctx, created); err != nil {
		t.Fatalf("put auth user: %v", err)
	}
	if strings.TrimSpace(created.ID) == "" {
		t.Fatal("create auth user: missing user id")
	}

	if strings.TrimSpace(authAddr) != "" {
		conn, dialErr := grpc.NewClient(
			authAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		)
		if dialErr != nil {
			t.Fatalf("dial auth server: %v", dialErr)
		}
		defer conn.Close()

		client := authv1.NewAuthServiceClient(conn)
		if _, lookupErr := client.LookupUserByUsername(ctx, &authv1.LookupUserByUsernameRequest{Username: created.Username}); lookupErr != nil {
			t.Fatalf("lookup created auth user: %v", lookupErr)
		}
	}

	return created.ID
}
