package authtest

import (
	"context"
	"strings"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CreateWebSession issues a durable auth-owned web session for one existing
// user so browser-surface tests can authenticate through the auth service
// boundary.
func CreateWebSession(t *testing.T, authAddr, userID string) string {
	t.Helper()

	authAddr = strings.TrimSpace(authAddr)
	if authAddr == "" {
		t.Fatal("auth address is required")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		t.Fatal("user id is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial auth server: %v", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Fatalf("close auth connection: %v", closeErr)
		}
	}()

	client := authv1.NewAuthServiceClient(conn)
	resp, err := client.CreateWebSession(ctx, &authv1.CreateWebSessionRequest{UserId: userID})
	if err != nil {
		t.Fatalf("create auth web session: %v", err)
	}

	sessionID := strings.TrimSpace(resp.GetSession().GetId())
	if sessionID == "" {
		t.Fatal("create auth web session: missing session id")
	}
	if gotUserID := strings.TrimSpace(resp.GetUser().GetId()); gotUserID != userID {
		t.Fatalf("create auth web session returned user %q, want %q", gotUserID, userID)
	}

	verifyResp, err := client.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil {
		t.Fatalf("verify auth web session: %v", err)
	}
	if gotUserID := strings.TrimSpace(verifyResp.GetUser().GetId()); gotUserID != userID {
		t.Fatalf("verified auth web session user %q, want %q", gotUserID, userID)
	}

	return sessionID
}
