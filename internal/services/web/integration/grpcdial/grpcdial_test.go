package grpcdial

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
)

func TestNormalizeTimeoutUsesDefaultForNonPositive(t *testing.T) {
	t.Parallel()

	if got := normalizeTimeout(0); got != timeouts.GRPCDial {
		t.Fatalf("normalizeTimeout(0) = %v, want %v", got, timeouts.GRPCDial)
	}
	if got := normalizeTimeout(-1 * time.Second); got != timeouts.GRPCDial {
		t.Fatalf("normalizeTimeout(-1s) = %v, want %v", got, timeouts.GRPCDial)
	}
}

func TestDialAuthEmptyAddrReturnsNoClients(t *testing.T) {
	t.Parallel()

	clients, err := DialAuth(context.Background(), "   ", 0)
	if err != nil {
		t.Fatalf("DialAuth returned error: %v", err)
	}
	if clients.Conn != nil {
		t.Fatal("DialAuth returned non-nil conn for empty addr")
	}
	if clients.AuthClient != nil || clients.AccountClient != nil {
		t.Fatal("DialAuth returned non-nil clients for empty addr")
	}
}

func TestDialAuthNilContextWithAddrReturnsError(t *testing.T) {
	t.Parallel()

	_, err := DialAuth(nil, "127.0.0.1:1", 50*time.Millisecond)
	if err == nil {
		t.Fatal("DialAuth(nil, addr, timeout) expected error")
	}
	if !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("DialAuth(nil, ...) error = %q, want context-required message", err.Error())
	}
}

func TestDialAuthDialErrorIncludesServiceName(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	_, err := DialAuth(ctx, "127.0.0.1:1", 50*time.Millisecond)
	if err == nil {
		t.Fatal("DialAuth expected dial error")
	}
	if !strings.Contains(err.Error(), "dial auth gRPC 127.0.0.1:1") {
		t.Fatalf("DialAuth error = %q, want auth service dial context", err.Error())
	}
}

func TestDialConnectionsEmptyAddrReturnsNoClients(t *testing.T) {
	t.Parallel()

	clients, err := DialConnections(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("DialConnections returned error: %v", err)
	}
	if clients.Conn != nil || clients.ConnectionsClient != nil {
		t.Fatal("DialConnections returned non-nil values for empty addr")
	}
}

func TestDialGameEmptyAddrReturnsNoClients(t *testing.T) {
	t.Parallel()

	clients, err := DialGame(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("DialGame returned error: %v", err)
	}
	if clients.Conn != nil {
		t.Fatal("DialGame returned non-nil conn for empty addr")
	}
}

func TestDialAIEmptyAddrReturnsNoClients(t *testing.T) {
	t.Parallel()

	clients, err := DialAI(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("DialAI returned error: %v", err)
	}
	if clients.Conn != nil || clients.CredentialClient != nil {
		t.Fatal("DialAI returned non-nil values for empty addr")
	}
}

func TestDialNotificationsEmptyAddrReturnsNoClients(t *testing.T) {
	t.Parallel()

	clients, err := DialNotifications(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("DialNotifications returned error: %v", err)
	}
	if clients.Conn != nil || clients.NotificationClient != nil {
		t.Fatal("DialNotifications returned non-nil values for empty addr")
	}
}

func TestDialListingEmptyAddrReturnsNoClients(t *testing.T) {
	t.Parallel()

	clients, err := DialListing(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("DialListing returned error: %v", err)
	}
	if clients.Conn != nil || clients.ListingClient != nil {
		t.Fatal("DialListing returned non-nil values for empty addr")
	}
}
