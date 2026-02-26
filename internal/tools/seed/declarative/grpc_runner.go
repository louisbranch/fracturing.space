package declarative

import (
	"context"
	"fmt"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	seed "github.com/louisbranch/fracturing.space/internal/tools/seed"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DialConfig defines gRPC addresses used by the declarative runner.
type DialConfig struct {
	GameAddr    string
	AuthAddr    string
	SocialAddr  string
	ListingAddr string
}

// GRPCRunner owns client connections and applies one manifest end-to-end.
type GRPCRunner struct {
	runner *Runner
	conns  []*grpc.ClientConn
}

// NewGRPCRunner constructs a declarative runner backed by gRPC clients.
func NewGRPCRunner(cfg Config, dial DialConfig) (*GRPCRunner, error) {
	gameAddr := strings.TrimSpace(dial.GameAddr)
	if gameAddr == "" {
		return nil, fmt.Errorf("game address is required")
	}
	gameAddr = seed.ResolveLocalFallbackAddr(gameAddr)
	authAddr := strings.TrimSpace(dial.AuthAddr)
	if authAddr == "" {
		return nil, fmt.Errorf("auth address is required")
	}
	authAddr = seed.ResolveLocalFallbackAddr(authAddr)
	socialAddr := strings.TrimSpace(dial.SocialAddr)
	if socialAddr == "" {
		return nil, fmt.Errorf("social address is required")
	}
	socialAddr = seed.ResolveLocalFallbackAddr(socialAddr)
	listingAddr := strings.TrimSpace(dial.ListingAddr)
	if listingAddr == "" {
		return nil, fmt.Errorf("listing address is required")
	}
	listingAddr = seed.ResolveLocalFallbackAddr(listingAddr)

	gameConn, err := grpc.NewClient(
		gameAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		return nil, fmt.Errorf("connect game server: %w", err)
	}
	authConn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		_ = gameConn.Close()
		return nil, fmt.Errorf("connect auth server: %w", err)
	}
	socialConn, err := grpc.NewClient(
		socialAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		_ = authConn.Close()
		_ = gameConn.Close()
		return nil, fmt.Errorf("connect social server: %w", err)
	}
	listingConn, err := grpc.NewClient(
		listingAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		_ = socialConn.Close()
		_ = authConn.Close()
		_ = gameConn.Close()
		return nil, fmt.Errorf("connect listing server: %w", err)
	}

	return &GRPCRunner{
		runner: newRunnerWithClients(cfg, runnerDeps{
			auth:         authv1.NewAuthServiceClient(authConn),
			social:       socialv1.NewSocialServiceClient(socialConn),
			campaigns:    gamev1.NewCampaignServiceClient(gameConn),
			participants: gamev1.NewParticipantServiceClient(gameConn),
			characters:   gamev1.NewCharacterServiceClient(gameConn),
			sessions:     gamev1.NewSessionServiceClient(gameConn),
			forks:        gamev1.NewForkServiceClient(gameConn),
			listings:     listingv1.NewCampaignListingServiceClient(listingConn),
		}),
		conns: []*grpc.ClientConn{gameConn, authConn, socialConn, listingConn},
	}, nil
}

// Run loads and applies the configured manifest.
func (r *GRPCRunner) Run(ctx context.Context) error {
	if r == nil || r.runner == nil {
		return fmt.Errorf("runner is not configured")
	}
	return r.runner.Run(ctx)
}

// Close closes all owned gRPC connections.
func (r *GRPCRunner) Close() error {
	if r == nil {
		return nil
	}
	var firstErr error
	for _, conn := range r.conns {
		if conn == nil {
			continue
		}
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
