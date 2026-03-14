package scenario

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	authstorage "github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	authuser "github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type storeAuth struct {
	authDBPath string
	authAddr   string
}

func newRunnerAuthProvider() authProvider {
	authDBPath := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AUTH_DB_PATH"))
	if authDBPath == "" {
		return NewMockAuth()
	}
	return &storeAuth{
		authDBPath: authDBPath,
		authAddr:   strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AUTH_ADDR")),
	}
}

func (s *storeAuth) CreateUser(displayName string) (string, error) {
	store, err := authsqlite.Open(s.authDBPath)
	if err != nil {
		return "", fmt.Errorf("open auth store: %w", err)
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	username := scenarioUsername(displayName)
	existing, err := store.GetUserByUsername(ctx, username)
	if err == nil {
		return existing.ID, s.lookupByUsername(ctx, username)
	}
	if !errors.Is(err, authstorage.ErrNotFound) {
		return "", fmt.Errorf("lookup auth user %q: %w", username, err)
	}

	created, err := authuser.CreateUser(authuser.CreateUserInput{Username: username}, nil, nil)
	if err != nil {
		return "", fmt.Errorf("create auth user %q: %w", username, err)
	}
	if err := store.PutUser(ctx, created); err != nil {
		return "", fmt.Errorf("put auth user %q: %w", username, err)
	}
	if strings.TrimSpace(created.ID) == "" {
		return "", errors.New("created auth user is missing id")
	}
	return created.ID, s.lookupByUsername(ctx, created.Username)
}

func (s *storeAuth) lookupByUsername(ctx context.Context, username string) error {
	if s.authAddr == "" {
		return nil
	}

	conn, err := grpc.NewClient(
		s.authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		return fmt.Errorf("dial auth server: %w", err)
	}
	defer conn.Close()

	client := authv1.NewAuthServiceClient(conn)
	if _, err := client.LookupUserByUsername(ctx, &authv1.LookupUserByUsernameRequest{Username: username}); err != nil {
		return fmt.Errorf("lookup auth user via server: %w", err)
	}
	return nil
}

func scenarioUsername(displayName string) string {
	value := strings.ToLower(strings.TrimSpace(displayName))
	if value == "" {
		return "scenario-runner"
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case !lastDash:
			builder.WriteByte('-')
			lastDash = true
		}
	}

	username := strings.Trim(builder.String(), "-")
	if username == "" {
		return "scenario-runner"
	}
	if first := username[0]; first < 'a' || first > 'z' {
		username = "u-" + username
	}
	if len(username) < 3 {
		username = "scenario-runner"
	}
	return username
}
