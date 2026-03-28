package app

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/interceptors"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

// transportBootstrapper owns startup-time gRPC server construction and service
// registration so the root bootstrap only sequences phases.
type transportBootstrapper interface {
	Bootstrap(
		bundle *storageBundle,
		srvEnv serverEnv,
		daggerheartDeps daggerheartRegistrationDeps,
		campaignDeps campaignRegistrationDeps,
		sessionDeps sessionRegistrationDeps,
		infrastructureDeps infrastructureRegistrationDeps,
	) (transportRuntimeState, error)
}

type transportRuntimeState struct {
	grpcServer   *grpc.Server
	healthServer *health.Server
}

type transportServiceRegistrar interface {
	Register(
		grpcServer *grpc.Server,
		healthServer *health.Server,
		daggerheartDeps daggerheartRegistrationDeps,
		campaignDeps campaignRegistrationDeps,
		sessionDeps sessionRegistrationDeps,
		infrastructureDeps infrastructureRegistrationDeps,
		sessionGrantConfig aisessiongrant.Config,
	) error
}

type transportServiceRegistrarFunc func(
	grpcServer *grpc.Server,
	healthServer *health.Server,
	daggerheartDeps daggerheartRegistrationDeps,
	campaignDeps campaignRegistrationDeps,
	sessionDeps sessionRegistrationDeps,
	infrastructureDeps infrastructureRegistrationDeps,
	sessionGrantConfig aisessiongrant.Config,
) error

func (f transportServiceRegistrarFunc) Register(
	grpcServer *grpc.Server,
	healthServer *health.Server,
	daggerheartDeps daggerheartRegistrationDeps,
	campaignDeps campaignRegistrationDeps,
	sessionDeps sessionRegistrationDeps,
	infrastructureDeps infrastructureRegistrationDeps,
	sessionGrantConfig aisessiongrant.Config,
) error {
	return f(grpcServer, healthServer, daggerheartDeps, campaignDeps, sessionDeps, infrastructureDeps, sessionGrantConfig)
}

type defaultTransportBootstrapper struct {
	newGRPCServer            func(*storageBundle, serverEnv) *grpc.Server
	newHealthServer          func() *health.Server
	loadAISessionGrantConfig func(func() time.Time) (aisessiongrant.Config, error)
	registerServices         transportServiceRegistrar
}

func (b defaultTransportBootstrapper) Bootstrap(
	bundle *storageBundle,
	srvEnv serverEnv,
	daggerheartDeps daggerheartRegistrationDeps,
	campaignDeps campaignRegistrationDeps,
	sessionDeps sessionRegistrationDeps,
	infrastructureDeps infrastructureRegistrationDeps,
) (transportRuntimeState, error) {
	grpcServer := b.newGRPCServer(bundle, srvEnv)
	healthServer := b.newHealthServer()
	sessionGrantConfig, err := b.loadAISessionGrantConfig(time.Now)
	if err != nil {
		return transportRuntimeState{}, fmt.Errorf("load ai session grant config: %w", err)
	}
	if err := b.registerServices.Register(
		grpcServer,
		healthServer,
		daggerheartDeps,
		campaignDeps,
		sessionDeps,
		infrastructureDeps,
		sessionGrantConfig,
	); err != nil {
		return transportRuntimeState{}, fmt.Errorf("register gRPC services: %w", err)
	}
	return transportRuntimeState{
		grpcServer:   grpcServer,
		healthServer: healthServer,
	}, nil
}

var _ transportBootstrapper = defaultTransportBootstrapper{}

func startupLogf(format string, args ...any) {
	slog.Info(fmt.Sprintf(format, args...))
}

func newStatusServiceClient(conn grpc.ClientConnInterface) statusv1.StatusServiceClient {
	return statusv1.NewStatusServiceClient(conn)
}

func newAuthServiceClient(conn grpc.ClientConnInterface) authv1.AuthServiceClient {
	return authv1.NewAuthServiceClient(conn)
}

func newAIAgentServiceClient(conn grpc.ClientConnInterface) aiv1.AgentServiceClient {
	return aiv1.NewAgentServiceClient(conn)
}

func loadAISessionGrantConfig(now func() time.Time) (aisessiongrant.Config, error) {
	return aisessiongrant.LoadConfigFromEnv(now)
}

func newDefaultHealthServer() *health.Server {
	return health.NewServer()
}

func newDefaultGRPCServer(bundle *storageBundle, srvEnv serverEnv) *grpc.Server {
	internalIdentity := interceptors.InternalServiceIdentityConfig{
		MethodPrefixes: []string{
			"/game.v1.CampaignAIService/",
			"/game.v1.CampaignAIOrchestrationService/",
			campaignv1.EventService_AppendEvent_FullMethodName,
			"/game.v1.IntegrationService/",
			campaignv1.ParticipantService_BindParticipant_FullMethodName,
		},
		AllowedServiceIDs: parseInternalServiceAllowlist(srvEnv.InternalServiceAllowlist),
	}
	return grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			grpcmeta.UnaryServerInterceptor(nil),
			interceptors.InternalServiceIdentityUnaryInterceptor(internalIdentity),
			interceptors.AuditInterceptor(audit.EnabledPolicy(bundle.events)),
			interceptors.SessionLockInterceptor(bundle.projections),
			interceptors.ErrorConversionUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			grpcmeta.StreamServerInterceptor(nil),
			interceptors.InternalServiceIdentityStreamInterceptor(internalIdentity),
			interceptors.StreamAuditInterceptor(audit.EnabledPolicy(bundle.events)),
			interceptors.SessionLockStreamInterceptor(),
			interceptors.ErrorConversionStreamInterceptor(),
		),
	)
}

func parseInternalServiceAllowlist(raw string) map[string]struct{} {
	values := strings.Split(strings.TrimSpace(raw), ",")
	allowlist := make(map[string]struct{}, len(values))
	for _, value := range values {
		serviceID := strings.ToLower(strings.TrimSpace(value))
		if serviceID == "" {
			continue
		}
		allowlist[serviceID] = struct{}{}
	}
	return allowlist
}
