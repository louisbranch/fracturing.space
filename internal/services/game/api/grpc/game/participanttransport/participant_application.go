package participanttransport

import (
	"context"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// ClearCampaignAIBindingFunc clears the AI agent binding for a campaign.
// Injected from root campaign code to avoid import cycles between participant
// and campaign entity packages.
type ClearCampaignAIBindingFunc func(
	ctx context.Context,
	campaignID string,
	actorID string,
	actorType command.ActorType,
	requestID string,
	invocationID string,
) (storage.CampaignRecord, error)

// Deps holds the explicit dependencies for the participant transport subpackage.
type Deps struct {
	Auth        authz.PolicyDeps
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Social      socialv1.SocialServiceClient
	Write       domainwrite.WritePath
	Applier     projection.Applier

	// ClaimIndex enforces one-user-per-campaign seat uniqueness.
	// Optional — if nil, claim index checks are skipped in BindParticipant.
	ClaimIndex storage.ClaimIndexStore

	// Event provides authoritative event replay for bind-time conflict
	// detection. Optional — if nil, replay checks are skipped.
	Event storage.EventStore

	// ClearCampaignAIBinding is called when participant mutations require
	// clearing the campaign's AI binding (e.g., owner access changes or removal).
	// Optional — if nil, the AI binding clear step is skipped.
	ClearCampaignAIBinding ClearCampaignAIBindingFunc
}

// participantApplication coordinates participant transport use-cases across
// focused method files (create, update, delete, bind, and policy helpers).
type participantApplication struct {
	auth                   authz.PolicyDeps
	stores                 participantApplicationStores
	write                  domainwrite.WritePath
	applier                projection.Applier
	clock                  func() time.Time
	idGenerator            func() (string, error)
	authClient             authv1.AuthServiceClient
	claimIndex             storage.ClaimIndexStore
	eventStore             storage.EventStore
	clearCampaignAIBinding ClearCampaignAIBindingFunc
}

type participantApplicationStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Social      socialv1.SocialServiceClient
}

func newParticipantApplicationFromDeps(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
	authClient authv1.AuthServiceClient,
) participantApplication {
	app := participantApplication{
		auth: deps.Auth,
		stores: participantApplicationStores{
			Campaign:    deps.Campaign,
			Participant: deps.Participant,
			Character:   deps.Character,
			Social:      deps.Social,
		},
		write:                  deps.Write,
		applier:                deps.Applier,
		clock:                  clock,
		idGenerator:            idGenerator,
		authClient:             authClient,
		claimIndex:             deps.ClaimIndex,
		eventStore:             deps.Event,
		clearCampaignAIBinding: deps.ClearCampaignAIBinding,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
