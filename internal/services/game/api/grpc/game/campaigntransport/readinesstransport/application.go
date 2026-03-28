package readinesstransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Deps holds the explicit dependencies for the readiness transport subpackage.
type Deps struct {
	Auth           authz.PolicyDeps
	Campaign       storage.CampaignStore
	Participant    storage.ParticipantStore
	Character      storage.CharacterStore
	Session        storage.SessionStore
	SystemStores   systemmanifest.ProjectionStores
	SystemMetadata *bridge.MetadataRegistry
	SystemModules  *module.Registry
}

// Application evaluates campaign session-start readiness.
type Application struct {
	auth   authz.PolicyDeps
	stores stores
}

type stores struct {
	Campaign       storage.CampaignStore
	Participant    storage.ParticipantStore
	Character      storage.CharacterStore
	Session        storage.SessionStore
	SystemStores   systemmanifest.ProjectionStores
	SystemMetadata *bridge.MetadataRegistry
	SystemModules  *module.Registry
}

// NewApplication creates a readiness Application from the given deps.
func NewApplication(deps Deps) Application {
	auth := deps.Auth
	if auth.Participant == nil {
		auth = authz.PolicyDeps{Participant: deps.Participant, Character: deps.Character, Audit: auth.Audit}
	}
	return Application{
		auth: auth,
		stores: stores{
			Campaign:       deps.Campaign,
			Participant:    deps.Participant,
			Character:      deps.Character,
			Session:        deps.Session,
			SystemStores:   deps.SystemStores,
			SystemMetadata: deps.SystemMetadata,
			SystemModules:  deps.SystemModules,
		},
	}
}

// GetCampaignSessionReadiness evaluates readiness blockers for starting a
// campaign session and returns a localized report.
func (a Application) GetCampaignSessionReadiness(
	ctx context.Context,
	campaignID string,
	requestedLocale commonv1.Locale,
) (*campaignv1.CampaignSessionReadiness, error) {
	record, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := authz.RequireReadPolicy(ctx, a.auth, record); err != nil {
		return nil, err
	}

	participantsByCampaign, err := a.stores.Participant.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("list participants by campaign", err)
	}

	charactersByCampaign, err := listAllCharactersByCampaign(ctx, a.stores.Character, campaignID)
	if err != nil {
		return nil, err
	}

	hasActiveSession, err := campaignHasActiveSession(ctx, a.stores.Session, campaignID)
	if err != nil {
		return nil, err
	}

	state, err := campaignReadinessAggregateState(record, participantsByCampaign, charactersByCampaign)
	if err != nil {
		return nil, err
	}
	state, err = bridge.ResolveSessionStartReadinessState(
		ctx,
		a.stores.SystemMetadata,
		ids.CampaignID(record.ID),
		handler.SystemIDFromCampaignRecord(record),
		a.stores.SystemStores,
		state,
	)
	if err != nil {
		return nil, grpcerror.Internal("load system readiness state", err)
	}

	report := readiness.EvaluateSessionStartReport(state, readiness.ReportOptions{
		SystemReadiness: systemReadinessChecker(
			a.stores.SystemModules,
			ids.CampaignID(record.ID),
			handler.SystemIDFromCampaignRecord(record),
			state,
		),
		IncludeSessionBoundary: true,
		HasActiveSession:       hasActiveSession,
	})
	locale := resolveReadinessLocale(requestedLocale, record.Locale)
	readinessProto := &campaignv1.CampaignSessionReadiness{
		Ready: report.Ready(),
	}
	if len(report.Blockers) == 0 {
		return readinessProto, nil
	}

	readinessProto.Blockers = make([]*campaignv1.CampaignSessionReadinessBlocker, 0, len(report.Blockers))
	for _, blocker := range report.Blockers {
		readinessProto.Blockers = append(readinessProto.Blockers, readinessBlockerToProto(locale, blocker))
	}
	return readinessProto, nil
}
