package sessiontransport

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestListActiveSessionsForUser_NilRequest(t *testing.T) {
	t.Parallel()

	svc := NewSessionService(Deps{})
	_, err := svc.ListActiveSessionsForUser(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListActiveSessionsForUser_RequiresUserID(t *testing.T) {
	t.Parallel()

	svc := NewSessionService(Deps{})
	_, err := svc.ListActiveSessionsForUser(context.Background(), &statev1.ListActiveSessionsForUserRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListActiveSessionsForUser_ReturnsSortedPage(t *testing.T) {
	t.Parallel()

	participantStore := gametest.NewFakeParticipantStore()
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()

	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	participantStore.Participants["camp-1"] = map[string]storage.ParticipantRecord{
		"seat-1": {ID: "seat-1", CampaignID: "camp-1", UserID: "user-1"},
	}
	participantStore.Participants["camp-2"] = map[string]storage.ParticipantRecord{
		"seat-2": {ID: "seat-2", CampaignID: "camp-2", UserID: "user-1"},
	}
	participantStore.Participants["camp-3"] = map[string]storage.ParticipantRecord{
		"seat-3": {ID: "seat-3", CampaignID: "camp-3", UserID: "user-1"},
	}
	participantStore.Participants["camp-4"] = map[string]storage.ParticipantRecord{
		"seat-4": {ID: "seat-4", CampaignID: "camp-4", UserID: "user-1"},
	}

	campaignStore.Campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", Name: "Amber Keep"}
	campaignStore.Campaigns["camp-2"] = storage.CampaignRecord{ID: "camp-2", Name: "Brass Harbor"}
	campaignStore.Campaigns["camp-3"] = storage.CampaignRecord{ID: "camp-3", Name: "Cinder Vale"}
	campaignStore.Campaigns["camp-4"] = storage.CampaignRecord{ID: "camp-4", Name: "Dormant Hollow"}

	sessionStore.Sessions["camp-1"] = map[string]storage.SessionRecord{
		"session-1": {ID: "session-1", CampaignID: "camp-1", Name: "Night Watch", Status: session.StatusActive, StartedAt: now.Add(-2 * time.Hour)},
	}
	sessionStore.ActiveSession["camp-1"] = "session-1"
	sessionStore.Sessions["camp-2"] = map[string]storage.SessionRecord{
		"session-2": {ID: "session-2", CampaignID: "camp-2", Name: "Harbor Rush", Status: session.StatusActive, StartedAt: now.Add(-30 * time.Minute)},
	}
	sessionStore.ActiveSession["camp-2"] = "session-2"
	sessionStore.Sessions["camp-3"] = map[string]storage.SessionRecord{
		"session-3": {ID: "session-3", CampaignID: "camp-3", Name: "Ashfall", Status: session.StatusActive, StartedAt: now.Add(-30 * time.Minute)},
	}
	sessionStore.ActiveSession["camp-3"] = "session-3"

	svc := NewSessionService(Deps{
		Campaign:    campaignStore,
		Participant: participantStore,
		Session:     sessionStore,
	})

	resp, err := svc.ListActiveSessionsForUser(gametest.ContextWithUserID("user-1"), &statev1.ListActiveSessionsForUserRequest{PageSize: 2})
	if err != nil {
		t.Fatalf("ListActiveSessionsForUser() error = %v", err)
	}
	if !resp.GetHasMore() {
		t.Fatalf("HasMore = false, want true")
	}
	if len(resp.GetSessions()) != 2 {
		t.Fatalf("len(Sessions) = %d, want 2", len(resp.GetSessions()))
	}

	if got := resp.GetSessions()[0].GetCampaignId(); got != "camp-2" {
		t.Fatalf("Sessions[0].CampaignId = %q, want %q", got, "camp-2")
	}
	if got := resp.GetSessions()[1].GetCampaignId(); got != "camp-3" {
		t.Fatalf("Sessions[1].CampaignId = %q, want %q", got, "camp-3")
	}
	if got := resp.GetSessions()[0].GetCampaignName(); got != "Brass Harbor" {
		t.Fatalf("Sessions[0].CampaignName = %q, want %q", got, "Brass Harbor")
	}
	if got := resp.GetSessions()[0].GetSessionName(); got != "Harbor Rush" {
		t.Fatalf("Sessions[0].SessionName = %q, want %q", got, "Harbor Rush")
	}
}

func TestListActiveSessionsForUser_PropagatesParticipantLookupFailure(t *testing.T) {
	t.Parallel()

	participantStore := gametest.NewFakeParticipantStore()
	participantStore.ListCampaignIDsByUserErr = errors.New("boom")

	svc := NewSessionService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Participant: participantStore,
		Session:     gametest.NewFakeSessionStore(),
	})

	_, err := svc.ListActiveSessionsForUser(gametest.ContextWithUserID("user-1"), &statev1.ListActiveSessionsForUserRequest{})
	assertStatusCode(t, err, codes.Internal)
}
