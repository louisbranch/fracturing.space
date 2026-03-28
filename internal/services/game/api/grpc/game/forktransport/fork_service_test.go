package forktransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestForkCampaign_RequiresDomainEngine(t *testing.T) {
	ctx := requestctx.WithAdminOverride(context.Background(), "fork-test")
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()

	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      bridge.SystemIDDaggerheart,
		GmMode:      campaign.GmModeHuman,
		ThemePrompt: "theme",
	}

	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-10 * time.Hour),
		Type:       event.Type("campaign.created"),
		EntityType: "campaign",
		EntityID:   "source",
		PayloadJSON: mustJSON(t, campaign.CreatePayload{
			Name:        "Source Campaign",
			GameSystem:  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			GmMode:      statev1.GmMode_HUMAN.String(),
			ThemePrompt: "theme",
		}),
	})

	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		Event:        eventStore,
		CampaignFork: forkStore,
	}, runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
		CopyParticipants: false,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestForkCampaign_RejectsWhenSourceCampaignHasActiveSession(t *testing.T) {
	ctx := requestctx.WithAdminOverride(context.Background(), "fork-test")
	now := time.Date(2025, 2, 2, 9, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()
	sessionStore := gametest.NewFakeSessionStore()

	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      bridge.SystemIDDaggerheart,
		GmMode:      campaign.GmModeHuman,
		ThemePrompt: "theme",
	}
	sessionStore.Sessions["source"] = map[string]storage.SessionRecord{
		"sess-1": {
			ID:         "sess-1",
			CampaignID: "source",
			Name:       "Active Session",
			Status:     session.StatusActive,
			StartedAt:  now.Add(-1 * time.Hour),
		},
	}
	sessionStore.ActiveSession["source"] = "sess-1"

	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		Session:      sessionStore,
		Event:        eventStore,
		CampaignFork: forkStore,
		Write:        domainwrite.WritePath{Executor: &fakeDomainEngine{store: eventStore}},
	}, runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}
