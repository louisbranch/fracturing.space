package campaigntransport

import (
	"context"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Deps holds the explicit dependencies for the campaign transport subpackage.
type Deps struct {
	Auth               authz.PolicyDeps
	Campaign           storage.CampaignStore
	Participant        storage.ParticipantStore
	Character          storage.CharacterStore
	Session            storage.SessionStore
	SessionInteraction storage.SessionInteractionStore
	SystemStores       systemmanifest.ProjectionStores
	SystemMetadata     *bridge.MetadataRegistry
	SystemModules      *module.Registry
	Social             handler.SocialProfileClient
	Write              domainwrite.WritePath
	Applier            projection.Applier
	AuthClient         handler.AuthUserClient
	AIClient           aiv1.AgentServiceClient
}

// campaignApplication coordinates campaign transport use-cases across focused
// method files (creation, mutation, status transitions, and AI binding).
type campaignApplication struct {
	auth        authz.PolicyDeps
	stores      campaignApplicationStores
	write       domainwrite.WritePath
	applier     projection.Applier
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  handler.AuthUserClient
	aiClient    aiv1.AgentServiceClient
}

type campaignApplicationStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Session     storage.SessionStore
	Social      handler.SocialProfileClient
}

type campaignCommandExecution struct {
	Campaign storage.CampaignStore
	Write    domainwrite.WritePath
	Applier  projection.Applier
}

func newCampaignApplicationWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) campaignApplication {
	auth := deps.Auth
	if auth.Participant == nil {
		auth = authz.PolicyDeps{Participant: deps.Participant, Character: deps.Character, Audit: auth.Audit}
	}
	app := campaignApplication{
		auth: auth,
		stores: campaignApplicationStores{
			Campaign:    deps.Campaign,
			Participant: deps.Participant,
			Session:     deps.Session,
			Social:      deps.Social,
		},
		write:       deps.Write,
		applier:     deps.Applier,
		clock:       clock,
		idGenerator: idGenerator,
		authClient:  deps.AuthClient,
		aiClient:    deps.AIClient,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}

// NewClearCampaignAIBindingFunc returns a callback that clears the AI agent
// binding for a campaign. Used by participanttransport to avoid importing
// campaign-internal types.
func NewClearCampaignAIBindingFunc(
	campaignStore storage.CampaignStore,
	write domainwrite.WritePath,
	applier projection.Applier,
) func(ctx context.Context, campaignID, actorID string, actorType command.ActorType, requestID, invocationID string) (storage.CampaignRecord, error) {
	return func(ctx context.Context, campaignID, actorID string, actorType command.ActorType, requestID, invocationID string) (storage.CampaignRecord, error) {
		return clearCampaignAIBindingByCommand(ctx, campaignCommandExecution{
			Campaign: campaignStore,
			Write:    write,
			Applier:  applier,
		}, campaignID, actorID, actorType, requestID, invocationID)
	}
}

func (c campaignApplication) commandExecution() campaignCommandExecution {
	return campaignCommandExecution{
		Campaign: c.stores.Campaign,
		Write:    c.write,
		Applier:  c.applier,
	}
}
