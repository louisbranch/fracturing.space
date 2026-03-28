package eventtransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestListTimelineEntries_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.ListTimelineEntries(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListTimelineEntries_MissingCampaignId(t *testing.T) {
	service := NewService(Deps{Event: gametest.NewFakeEventStore()})
	_, err := service.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListTimelineEntries_RequiresCampaignReadPolicy(t *testing.T) {
	participantStore := gametest.NewFakeParticipantStore()
	service := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: participantStore},
		Event:       gametest.NewFakeEventStore(),
		Participant: participantStore,
	})
	_, err := service.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListTimelineEntries_InvalidFilter(t *testing.T) {
	service := NewService(Deps{Event: gametest.NewFakeEventStore()})
	_, err := service.ListTimelineEntries(context.Background(), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
		Filter:     "invalid filter syntax ===",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListTimelineEntries_MissingParticipantStoreFailsFast(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	eventStore.Events["c1"] = []event.Event{{
		CampaignID: "c1",
		Seq:        1,
		Type:       event.Type("participant.joined"),
		EntityType: "participant",
		EntityID:   "p1",
		Timestamp:  time.Now().UTC(),
	}}

	service := NewService(Deps{Event: eventStore})

	_, err := service.ListTimelineEntries(requestctx.WithAdminOverride(context.Background(), "timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
	grpcassert.StatusMessage(t, err, "resolve timeline entry")
}

func TestListTimelineEntries_MissingCampaignStoreFailsFast(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	eventStore.Events["c1"] = []event.Event{{
		CampaignID: "c1",
		Seq:        1,
		Type:       event.Type("campaign.created"),
		EntityType: "campaign",
		EntityID:   "c1",
		Timestamp:  time.Now().UTC(),
	}}

	service := NewService(Deps{Event: eventStore})

	_, err := service.ListTimelineEntries(requestctx.WithAdminOverride(context.Background(), "timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
	grpcassert.StatusMessage(t, err, "resolve timeline entry")
}

func TestListTimelineEntries_MissingCharacterStoreFailsFast(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	eventStore.Events["c1"] = []event.Event{{
		CampaignID: "c1",
		Seq:        1,
		Type:       event.Type("character.created"),
		EntityType: "character",
		EntityID:   "ch1",
		Timestamp:  time.Now().UTC(),
	}}

	service := NewService(Deps{Event: eventStore})

	_, err := service.ListTimelineEntries(requestctx.WithAdminOverride(context.Background(), "timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
	grpcassert.StatusMessage(t, err, "resolve timeline entry")
}

func TestListTimelineEntries_MissingSessionStoreFailsFast(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	eventStore.Events["c1"] = []event.Event{{
		CampaignID: "c1",
		Seq:        1,
		Type:       event.Type("session.started"),
		EntityType: "session",
		EntityID:   "s1",
		Timestamp:  time.Now().UTC(),
	}}

	service := NewService(Deps{Event: eventStore})

	_, err := service.ListTimelineEntries(requestctx.WithAdminOverride(context.Background(), "timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
	grpcassert.StatusMessage(t, err, "resolve timeline entry")
}
