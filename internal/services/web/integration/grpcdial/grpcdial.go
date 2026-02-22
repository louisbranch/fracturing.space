package grpcdial

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"google.golang.org/grpc"
)

// AuthClients contains auth connections and typed clients used by web transport handlers.
type AuthClients struct {
	Conn          *grpc.ClientConn
	AuthClient    authv1.AuthServiceClient
	AccountClient authv1.AccountServiceClient
}

// ConnectionsClients contains connections gRPC connection and typed clients.
type ConnectionsClients struct {
	Conn              *grpc.ClientConn
	ConnectionsClient connectionsv1.ConnectionsServiceClient
}

// GameClients contains game gRPC connection and typed clients used by web handlers.
type GameClients struct {
	Conn              *grpc.ClientConn
	ParticipantClient statev1.ParticipantServiceClient
	CampaignClient    statev1.CampaignServiceClient
	EventClient       statev1.EventServiceClient
	SessionClient     statev1.SessionServiceClient
	CharacterClient   statev1.CharacterServiceClient
	InviteClient      statev1.InviteServiceClient
}

// AIClients contains ai gRPC connection and typed clients used by web handlers.
type AIClients struct {
	Conn             *grpc.ClientConn
	CredentialClient aiv1.CredentialServiceClient
}

// NotificationsClients contains notifications gRPC connection and typed clients.
type NotificationsClients struct {
	Conn               *grpc.ClientConn
	NotificationClient notificationsv1.NotificationServiceClient
}

// ListingClients contains listing gRPC connection and typed clients.
type ListingClients struct {
	Conn          *grpc.ClientConn
	ListingClient listingv1.CampaignListingServiceClient
}

func normalizeTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return timeouts.GRPCDial
	}
	return timeout
}

func dialWithHealth(ctx context.Context, addr string, timeout time.Duration, serviceName string) (*grpc.ClientConn, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, nil
	}
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	timeout = normalizeTimeout(timeout)
	logf := func(format string, args ...any) {
		log.Printf("%s %s", serviceName, fmt.Sprintf(format, args...))
	}
	conn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		addr,
		timeout,
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		var dialErr *platformgrpc.DialError
		if errors.As(err, &dialErr) {
			if dialErr.Stage == platformgrpc.DialStageHealth {
				return nil, fmt.Errorf("%s gRPC health check failed for %s: %w", serviceName, addr, dialErr.Err)
			}
			return nil, fmt.Errorf("dial %s gRPC %s: %w", serviceName, addr, dialErr.Err)
		}
		return nil, fmt.Errorf("dial %s gRPC %s: %w", serviceName, addr, err)
	}
	return conn, nil
}

// DialAuth returns gRPC clients for auth-backed login and account profile operations.
func DialAuth(ctx context.Context, addr string, timeout time.Duration) (AuthClients, error) {
	conn, err := dialWithHealth(ctx, addr, timeout, "auth")
	if err != nil {
		return AuthClients{}, err
	}
	if conn == nil {
		return AuthClients{}, nil
	}
	return AuthClients{
		Conn:          conn,
		AuthClient:    authv1.NewAuthServiceClient(conn),
		AccountClient: authv1.NewAccountServiceClient(conn),
	}, nil
}

// DialConnections returns gRPC clients for contact/discovery operations.
func DialConnections(ctx context.Context, addr string, timeout time.Duration) (ConnectionsClients, error) {
	conn, err := dialWithHealth(ctx, addr, timeout, "connections")
	if err != nil {
		return ConnectionsClients{}, err
	}
	if conn == nil {
		return ConnectionsClients{}, nil
	}
	return ConnectionsClients{
		Conn:              conn,
		ConnectionsClient: connectionsv1.NewConnectionsServiceClient(conn),
	}, nil
}

// DialGame returns gRPC clients for campaign/session/participant/character/invite operations.
func DialGame(ctx context.Context, addr string, timeout time.Duration) (GameClients, error) {
	conn, err := dialWithHealth(ctx, addr, timeout, "game")
	if err != nil {
		return GameClients{}, err
	}
	if conn == nil {
		return GameClients{}, nil
	}
	return GameClients{
		Conn:              conn,
		ParticipantClient: statev1.NewParticipantServiceClient(conn),
		CampaignClient:    statev1.NewCampaignServiceClient(conn),
		EventClient:       statev1.NewEventServiceClient(conn),
		SessionClient:     statev1.NewSessionServiceClient(conn),
		CharacterClient:   statev1.NewCharacterServiceClient(conn),
		InviteClient:      statev1.NewInviteServiceClient(conn),
	}, nil
}

// DialAI returns gRPC clients for ai credential operations.
func DialAI(ctx context.Context, addr string, timeout time.Duration) (AIClients, error) {
	conn, err := dialWithHealth(ctx, addr, timeout, "ai")
	if err != nil {
		return AIClients{}, err
	}
	if conn == nil {
		return AIClients{}, nil
	}
	return AIClients{
		Conn:             conn,
		CredentialClient: aiv1.NewCredentialServiceClient(conn),
	}, nil
}

// DialNotifications returns gRPC clients for notifications inbox operations.
func DialNotifications(ctx context.Context, addr string, timeout time.Duration) (NotificationsClients, error) {
	conn, err := dialWithHealth(ctx, addr, timeout, "notifications")
	if err != nil {
		return NotificationsClients{}, err
	}
	if conn == nil {
		return NotificationsClients{}, nil
	}
	return NotificationsClients{
		Conn:               conn,
		NotificationClient: notificationsv1.NewNotificationServiceClient(conn),
	}, nil
}

// DialListing returns gRPC clients for public discovery listing operations.
func DialListing(ctx context.Context, addr string, timeout time.Duration) (ListingClients, error) {
	conn, err := dialWithHealth(ctx, addr, timeout, "listing")
	if err != nil {
		return ListingClients{}, err
	}
	if conn == nil {
		return ListingClients{}, nil
	}
	return ListingClients{
		Conn:          conn,
		ListingClient: listingv1.NewCampaignListingServiceClient(conn),
	}, nil
}
