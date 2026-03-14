package settings

import (
	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	grpc "google.golang.org/grpc"

	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
)

// Dependencies contains settings feature clients.
type Dependencies struct {
	SocialClient     settingsgateway.SocialClient
	AccountClient    settingsgateway.AccountClient
	PasskeyClient    settingsgateway.PasskeyClient
	CredentialClient settingsgateway.CredentialClient
	AgentClient      settingsgateway.AgentClient
}

// BindAuthDependency wires auth-backed clients into the settings dependency
// set.
func BindAuthDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.AccountClient = authv1.NewAccountServiceClient(conn)
	deps.PasskeyClient = authv1.NewAuthServiceClient(conn)
}

// BindSocialDependency wires social-backed clients into the settings
// dependency set.
func BindSocialDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.SocialClient = socialv1.NewSocialServiceClient(conn)
}

// BindAIDependency wires AI-backed clients into the settings dependency set.
func BindAIDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.CredentialClient = aiv1.NewCredentialServiceClient(conn)
	deps.AgentClient = aiv1.NewAgentServiceClient(conn)
}
