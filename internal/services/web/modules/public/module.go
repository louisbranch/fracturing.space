package public

import (
	"context"
	"net/http"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
)

// AuthClient performs passkey and user bootstrap operations.
type AuthClient interface {
	CreateUser(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error)
	BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error)
	FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error)
	BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error)
	FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error)
	CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error)
	GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error)
	RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest, ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error)
}

// Module provides unauthenticated root/auth routes.
type Module struct {
	gateway        AuthGateway
	requestMeta    requestmeta.SchemePolicy
	id             string
	prefix         string
	registerRoutes func(*http.ServeMux, handlers)
}

// New returns a public/auth module with the given auth client.
func New(authClient AuthClient) Module {
	return NewWithGatewayAndPolicy(NewGRPCAuthGateway(authClient), requestmeta.SchemePolicy{})
}

// NewWithGateway returns a public/auth module with an explicit gateway.
func NewWithGateway(gateway AuthGateway) Module {
	return NewWithGatewayAndPolicy(gateway, requestmeta.SchemePolicy{})
}

// NewWithGatewayAndPolicy returns a public/auth module with explicit request metadata policy.
func NewWithGatewayAndPolicy(gateway AuthGateway, policy requestmeta.SchemePolicy) Module {
	return newPublicModule("public", routepath.Root, registerRoutes, gateway, policy)
}

// NewShellWithGateway returns an auth shell module with explicit routes and policy.
func NewShellWithGateway(gateway AuthGateway) Module {
	return NewShellWithGatewayAndPolicy(gateway, requestmeta.SchemePolicy{})
}

// NewShellWithGatewayAndPolicy returns the shell/public entrypoint routes module.
func NewShellWithGatewayAndPolicy(gateway AuthGateway, policy requestmeta.SchemePolicy) Module {
	return newPublicModule("public", routepath.Root, registerShellRoutes, gateway, policy)
}

// NewPasskeysWithGateway returns a module containing passkey JSON endpoints.
func NewPasskeysWithGateway(gateway AuthGateway) Module {
	return NewPasskeysWithGatewayAndPolicy(gateway, requestmeta.SchemePolicy{})
}

// NewPasskeysWithGatewayAndPolicy returns the passkey routes module.
func NewPasskeysWithGatewayAndPolicy(gateway AuthGateway, policy requestmeta.SchemePolicy) Module {
	return newPublicModule("public-passkeys", routepath.PasskeysPrefix, registerPasskeyRoutes, gateway, policy)
}

// NewAuthRedirectWithGateway returns an auth-redirect module.
func NewAuthRedirectWithGateway(gateway AuthGateway) Module {
	return NewAuthRedirectWithGatewayAndPolicy(gateway, requestmeta.SchemePolicy{})
}

// NewAuthRedirectWithGatewayAndPolicy returns the auth-redirect route module.
func NewAuthRedirectWithGatewayAndPolicy(gateway AuthGateway, policy requestmeta.SchemePolicy) Module {
	return newPublicModule("public-auth-redirect", routepath.AuthPrefix, registerAuthRedirectRoutes, gateway, policy)
}

func newPublicModule(id string, prefix string, registerRoutes func(*http.ServeMux, handlers), gateway AuthGateway, policy requestmeta.SchemePolicy) Module {
	return Module{
		gateway:        gateway,
		requestMeta:    policy,
		id:             strings.TrimSpace(id),
		prefix:         strings.TrimSpace(prefix),
		registerRoutes: registerRoutes,
	}
}

// ID returns a stable identifier for diagnostics and startup logs.
func (m Module) ID() string {
	id := strings.TrimSpace(m.id)
	if id == "" {
		return "public"
	}
	return id
}

// Mount wires public routes under the auth/root prefix.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	svc := newServiceWithGateway(m.gateway)
	h := newHandlers(svc, m.requestMeta)
	if m.registerRoutes != nil {
		m.registerRoutes(mux, h)
	} else {
		registerRoutes(mux, h)
	}
	prefix := strings.TrimSpace(m.prefix)
	if prefix == "" {
		prefix = routepath.Root
	}
	return module.Mount{Prefix: prefix, Handler: mux}, nil
}
