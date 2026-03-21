// Package invite implements the invite.v1.InviteService gRPC transport.
package invite

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
)

// Deps holds the explicit dependencies for the invite transport.
type Deps struct {
	Store        storage.InviteStore
	Outbox       storage.OutboxStore
	Game         gamev1.ParticipantServiceClient
	GameCampaign gamev1.CampaignServiceClient
	Auth         authv1.AuthServiceClient
	IDGenerator  func() (string, error)
	Clock        func() time.Time
	Verifier     joingrant.Verifier
}

// Service implements the invite.v1.InviteServiceServer gRPC API.
type Service struct {
	invitev1.UnimplementedInviteServiceServer
	store        storage.InviteStore
	outbox       storage.OutboxStore
	game         gamev1.ParticipantServiceClient
	gameCampaign gamev1.CampaignServiceClient
	auth         authv1.AuthServiceClient
	idGenerator  func() (string, error)
	clock        func() time.Time
	verifier     joingrant.Verifier
}

// NewService creates an InviteService with the provided dependencies.
func NewService(deps Deps) *Service {
	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}
	return &Service{
		store:        deps.Store,
		outbox:       deps.Outbox,
		game:         deps.Game,
		gameCampaign: deps.GameCampaign,
		auth:         deps.Auth,
		idGenerator:  deps.IDGenerator,
		clock:        clock,
		verifier:     deps.Verifier,
	}
}
