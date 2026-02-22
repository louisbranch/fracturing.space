package web

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcdial"
	websqlite "github.com/louisbranch/fracturing.space/internal/services/web/storage/sqlite"
)

func openWebCacheStore(path string) (*websqlite.Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create web cache dir: %w", err)
		}
	}
	store, err := websqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open web cache sqlite store: %w", err)
	}
	return store, nil
}

// buildAuthConsentURL resolves the post-magic-link consent callback.
// It keeps OAuth return handling deterministic regardless of deployment prefixing.
func buildAuthConsentURL(base string, pendingID string) string {
	base = strings.TrimSpace(base)
	encoded := url.QueryEscape(pendingID)
	if base == "" {
		return "/authorize/consent?pending_id=" + encoded
	}
	return strings.TrimRight(base, "/") + "/authorize/consent?pending_id=" + encoded
}

// dialAuthGRPC returns a client for auth-backed login/registration operations.
// Auth transport is optional in degraded startup modes so the web package can
// still stand up with limited capability.
func dialAuthGRPC(ctx context.Context, config Config) (authGRPCClients, error) {
	authAddr := strings.TrimSpace(config.AuthAddr)
	if authAddr == "" {
		return authGRPCClients{}, nil
	}
	if ctx == nil {
		return authGRPCClients{}, errors.New("context is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}
	logf := func(format string, args ...any) {
		log.Printf("auth %s", fmt.Sprintf(format, args...))
	}
	conn, err := grpcdial.DialWithHealth(
		ctx,
		authAddr,
		config.GRPCDialTimeout,
		"auth",
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return authGRPCClients{}, err
	}
	return authGRPCClients{
		conn:          conn,
		authClient:    authv1.NewAuthServiceClient(conn),
		accountClient: authv1.NewAccountServiceClient(conn),
	}, nil
}

// dialConnectionsGRPC returns a client for contact/discovery-oriented relationship data.
func dialConnectionsGRPC(ctx context.Context, config Config) (connectionsGRPCClients, error) {
	connectionsAddr := strings.TrimSpace(config.ConnectionsAddr)
	if connectionsAddr == "" {
		return connectionsGRPCClients{}, nil
	}
	if ctx == nil {
		return connectionsGRPCClients{}, errors.New("context is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}
	logf := func(format string, args ...any) {
		log.Printf("connections %s", fmt.Sprintf(format, args...))
	}
	conn, err := grpcdial.DialWithHealth(
		ctx,
		connectionsAddr,
		config.GRPCDialTimeout,
		"connections",
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return connectionsGRPCClients{}, err
	}
	return connectionsGRPCClients{
		conn:              conn,
		connectionsClient: connectionsv1.NewConnectionsServiceClient(conn),
	}, nil
}

// dialGameGRPC returns clients for campaign/character/session/invite operations.
// This dependency is optional by design so campaign routes can degrade gracefully
// during partial service outages.
func dialGameGRPC(ctx context.Context, config Config) (gameGRPCClients, error) {
	gameAddr := strings.TrimSpace(config.GameAddr)
	if gameAddr == "" {
		return gameGRPCClients{}, nil
	}
	if ctx == nil {
		return gameGRPCClients{}, errors.New("context is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}
	logf := func(format string, args ...any) {
		log.Printf("game %s", fmt.Sprintf(format, args...))
	}
	conn, err := grpcdial.DialWithHealth(
		ctx,
		gameAddr,
		config.GRPCDialTimeout,
		"game",
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return gameGRPCClients{}, err
	}
	return gameGRPCClients{
		conn:              conn,
		participantClient: statev1.NewParticipantServiceClient(conn),
		campaignClient:    statev1.NewCampaignServiceClient(conn),
		eventClient:       statev1.NewEventServiceClient(conn),
		sessionClient:     statev1.NewSessionServiceClient(conn),
		characterClient:   statev1.NewCharacterServiceClient(conn),
		inviteClient:      statev1.NewInviteServiceClient(conn),
	}, nil
}

// dialAIGRPC returns clients for settings-owned AI key operations.
func dialAIGRPC(ctx context.Context, config Config) (aiGRPCClients, error) {
	aiAddr := strings.TrimSpace(config.AIAddr)
	if aiAddr == "" {
		return aiGRPCClients{}, nil
	}
	if ctx == nil {
		return aiGRPCClients{}, errors.New("context is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}
	logf := func(format string, args ...any) {
		log.Printf("ai %s", fmt.Sprintf(format, args...))
	}
	conn, err := grpcdial.DialWithHealth(
		ctx,
		aiAddr,
		config.GRPCDialTimeout,
		"ai",
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return aiGRPCClients{}, err
	}
	return aiGRPCClients{
		conn:             conn,
		credentialClient: aiv1.NewCredentialServiceClient(conn),
	}, nil
}

// dialNotificationsGRPC returns clients for inbox notifications operations.
func dialNotificationsGRPC(ctx context.Context, config Config) (notificationsGRPCClients, error) {
	notificationsAddr := strings.TrimSpace(config.NotificationsAddr)
	if notificationsAddr == "" {
		return notificationsGRPCClients{}, nil
	}
	if ctx == nil {
		return notificationsGRPCClients{}, errors.New("context is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}
	logf := func(format string, args ...any) {
		log.Printf("notifications %s", fmt.Sprintf(format, args...))
	}
	conn, err := grpcdial.DialWithHealth(
		ctx,
		notificationsAddr,
		config.GRPCDialTimeout,
		"notifications",
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return notificationsGRPCClients{}, err
	}
	return notificationsGRPCClients{
		conn:               conn,
		notificationClient: notificationsv1.NewNotificationServiceClient(conn),
	}, nil
}
