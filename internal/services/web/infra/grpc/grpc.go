package grpc

import (
	"context"
	"time"

	intgrpc "github.com/louisbranch/fracturing.space/internal/services/web/integration/grpcdial"
)

// AuthClients contains auth connections and typed clients used by web handlers.
type AuthClients = intgrpc.AuthClients

// ConnectionsClients contains connections clients used by the web service.
type ConnectionsClients = intgrpc.ConnectionsClients

// GameClients contains game clients used by the web service.
type GameClients = intgrpc.GameClients

// AIClients contains AI clients used by the web service.
type AIClients = intgrpc.AIClients

// NotificationsClients contains notifications clients used by the web service.
type NotificationsClients = intgrpc.NotificationsClients

// ListingClients contains listing clients used by the web service.
type ListingClients = intgrpc.ListingClients

// DialAuth dials auth-gRPC and returns typed clients.
func DialAuth(ctx context.Context, addr string, timeout time.Duration) (AuthClients, error) {
	return intgrpc.DialAuth(ctx, addr, timeout)
}

// DialConnections dials connections-gRPC and returns typed clients.
func DialConnections(ctx context.Context, addr string, timeout time.Duration) (ConnectionsClients, error) {
	return intgrpc.DialConnections(ctx, addr, timeout)
}

// DialGame dials game-gRPC and returns typed clients.
func DialGame(ctx context.Context, addr string, timeout time.Duration) (GameClients, error) {
	return intgrpc.DialGame(ctx, addr, timeout)
}

// DialAI dials AI-gRPC and returns typed clients.
func DialAI(ctx context.Context, addr string, timeout time.Duration) (AIClients, error) {
	return intgrpc.DialAI(ctx, addr, timeout)
}

// DialNotifications dials notifications-gRPC and returns typed clients.
func DialNotifications(ctx context.Context, addr string, timeout time.Duration) (NotificationsClients, error) {
	return intgrpc.DialNotifications(ctx, addr, timeout)
}

// DialListing dials listing-gRPC and returns typed clients.
func DialListing(ctx context.Context, addr string, timeout time.Duration) (ListingClients, error) {
	return intgrpc.DialListing(ctx, addr, timeout)
}
