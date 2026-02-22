package grpcdial

import (
	"context"
	"log"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	sharedgrpcdial "github.com/louisbranch/fracturing.space/internal/services/shared/grpcdial"
	"google.golang.org/grpc"
)

const (
	// DefaultRetryDelay sets the initial wait time between gRPC dial attempts.
	DefaultRetryDelay = 500 * time.Millisecond
	// MaxRetryDelay caps the backoff between gRPC dial attempts.
	MaxRetryDelay = 10 * time.Second
)

// GameClients contains all game clients created by a successful game dial.
type GameClients struct {
	Conn              *grpc.ClientConn
	DaggerheartClient daggerheartv1.DaggerheartServiceClient
	ContentClient     daggerheartv1.DaggerheartContentServiceClient
	CampaignClient    statev1.CampaignServiceClient
	SessionClient     statev1.SessionServiceClient
	CharacterClient   statev1.CharacterServiceClient
	ParticipantClient statev1.ParticipantServiceClient
	InviteClient      statev1.InviteServiceClient
	SnapshotClient    statev1.SnapshotServiceClient
	EventClient       statev1.EventServiceClient
	StatisticsClient  statev1.StatisticsServiceClient
	SystemClient      statev1.SystemServiceClient
}

// AuthClients contains auth service clients created by auth dial.
type AuthClients struct {
	Conn          *grpc.ClientConn
	AuthClient    authv1.AuthServiceClient
	AccountClient authv1.AccountServiceClient
}

func normalizeTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return timeouts.GRPCDial
	}
	return timeout
}

// DialGame dials the game gRPC endpoint and returns typed clients.
func DialGame(ctx context.Context, addr string, timeout time.Duration, authzOverrideReason string) (GameClients, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return GameClients{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout = normalizeTimeout(timeout)

	logf := func(format string, args ...any) {
		log.Printf("admin game "+format, args...)
	}
	dialOpts := append(
		platformgrpc.DefaultClientDialOptions(),
		grpc.WithChainUnaryInterceptor(grpcauthctx.AdminOverrideUnaryClientInterceptor(authzOverrideReason)),
		grpc.WithChainStreamInterceptor(grpcauthctx.AdminOverrideStreamClientInterceptor(authzOverrideReason)),
	)
	conn, err := sharedgrpcdial.DialWithHealth(ctx, addr, timeout, "admin game", logf, dialOpts...)
	if err != nil {
		return GameClients{}, err
	}
	return GameClients{
		Conn:              conn,
		DaggerheartClient: daggerheartv1.NewDaggerheartServiceClient(conn),
		ContentClient:     daggerheartv1.NewDaggerheartContentServiceClient(conn),
		CampaignClient:    statev1.NewCampaignServiceClient(conn),
		SessionClient:     statev1.NewSessionServiceClient(conn),
		CharacterClient:   statev1.NewCharacterServiceClient(conn),
		ParticipantClient: statev1.NewParticipantServiceClient(conn),
		InviteClient:      statev1.NewInviteServiceClient(conn),
		SnapshotClient:    statev1.NewSnapshotServiceClient(conn),
		EventClient:       statev1.NewEventServiceClient(conn),
		StatisticsClient:  statev1.NewStatisticsServiceClient(conn),
		SystemClient:      statev1.NewSystemServiceClient(conn),
	}, nil
}

// DialAuth dials the auth gRPC endpoint and returns typed clients.
func DialAuth(ctx context.Context, addr string, timeout time.Duration) (AuthClients, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return AuthClients{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout = normalizeTimeout(timeout)

	logf := func(format string, args ...any) {
		log.Printf("admin auth "+format, args...)
	}
	conn, err := sharedgrpcdial.DialWithHealth(
		ctx,
		addr,
		timeout,
		"admin auth",
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return AuthClients{}, err
	}
	return AuthClients{
		Conn:          conn,
		AuthClient:    authv1.NewAuthServiceClient(conn),
		AccountClient: authv1.NewAccountServiceClient(conn),
	}, nil
}

// ConnectWithRetry keeps dialing until a connection is established or context ends.
func ConnectWithRetry(
	ctx context.Context,
	address string,
	hasConnection func() bool,
	connect func(context.Context) error,
	successLogFormat string,
	failureLogFormat string,
) {
	if hasConnection == nil || connect == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	retryDelay := DefaultRetryDelay
	for {
		if ctx.Err() != nil {
			return
		}
		if hasConnection() {
			return
		}

		err := connect(ctx)
		if err == nil {
			log.Printf(successLogFormat, address)
			return
		}

		log.Printf(failureLogFormat, err)
		timer := time.NewTimer(retryDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
		if retryDelay < MaxRetryDelay {
			retryDelay *= 2
			if retryDelay > MaxRetryDelay {
				retryDelay = MaxRetryDelay
			}
		}
	}
}
