package generator

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/tools/seed/worldbuilder"
	"google.golang.org/grpc"
)

func TestPickGmMode_VaryFalse(t *testing.T) {
	g := &Generator{rng: rand.New(rand.NewSource(1))}
	got := g.pickGmMode(false, 0)
	if got != statev1.GmMode_HUMAN {
		t.Fatalf("vary=false: want HUMAN, got %v", got)
	}
}

func TestPickGmMode_VaryTrue_CyclesThroughModes(t *testing.T) {
	g := &Generator{rng: rand.New(rand.NewSource(1))}
	want := []statev1.GmMode{
		statev1.GmMode_HUMAN,
		statev1.GmMode_AI,
		statev1.GmMode_HYBRID,
		statev1.GmMode_HUMAN, // wraps
	}
	for i, expected := range want {
		got := g.pickGmMode(true, i)
		if got != expected {
			t.Fatalf("index %d: want %v, got %v", i, expected, got)
		}
	}
}

// newTestGen returns a Generator wired to the given fakes with a deterministic RNG.
func newTestGen(seed int64, deps generatorDeps) *Generator {
	rng := rand.New(rand.NewSource(seed))
	return newGenerator(Config{Seed: seed}, rng, worldbuilder.New(rng), deps)
}

// happyAuthCreator returns a fakeAuthProvider that assigns sequential user IDs.
func happyAuthCreator() *fakeAuthProvider {
	seq := 0
	return &fakeAuthProvider{
		createUser: func(_ context.Context, in *authv1.CreateUserRequest, _ ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
			seq++
			return &authv1.CreateUserResponse{
				User: &authv1.User{Id: fmt.Sprintf("user-%d", seq)},
			}, nil
		},
		issueJoinGrant: func(_ context.Context, in *authv1.IssueJoinGrantRequest, _ ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
			return &authv1.IssueJoinGrantResponse{JoinGrant: "grant-token"}, nil
		},
	}
}

func TestCreateCampaign_HappyPath(t *testing.T) {
	auth := happyAuthCreator()
	camp := &fakeCampaignCreator{
		createCampaign: func(_ context.Context, in *statev1.CreateCampaignRequest, _ ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
			return &statev1.CreateCampaignResponse{
				Campaign:         &statev1.Campaign{Id: "camp-1", Name: in.Name},
				OwnerParticipant: &statev1.Participant{Id: "owner-1"},
			}, nil
		},
	}
	g := newTestGen(1, testDeps(camp, nil, nil, nil, nil, nil, auth))

	campaign, ownerID, err := g.createCampaign(context.Background(), statev1.GmMode_HUMAN)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if campaign.Id != "camp-1" {
		t.Fatalf("want campaign id camp-1, got %s", campaign.Id)
	}
	if ownerID != "owner-1" {
		t.Fatalf("want owner id owner-1, got %s", ownerID)
	}
}

func TestCreateCampaign_AuthError(t *testing.T) {
	auth := &fakeAuthProvider{
		createUser: func(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
			return nil, fmt.Errorf("auth unavailable")
		},
	}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, nil, nil, auth))

	_, _, err := g.createCampaign(context.Background(), statev1.GmMode_HUMAN)
	if err == nil {
		t.Fatal("expected error from auth failure")
	}
}

func TestCreateCampaign_EmptyUserID(t *testing.T) {
	auth := &fakeAuthProvider{
		createUser: func(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
			return &authv1.CreateUserResponse{User: &authv1.User{Id: ""}}, nil
		},
	}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, nil, nil, auth))

	_, _, err := g.createCampaign(context.Background(), statev1.GmMode_HUMAN)
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

func TestCreateCampaign_NilResponse(t *testing.T) {
	auth := happyAuthCreator()
	camp := &fakeCampaignCreator{
		createCampaign: func(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
			return nil, nil
		},
	}
	g := newTestGen(1, testDeps(camp, nil, nil, nil, nil, nil, auth))

	_, _, err := g.createCampaign(context.Background(), statev1.GmMode_HUMAN)
	if err == nil {
		t.Fatal("expected error for nil response")
	}
}

func TestCreateCampaign_NilOwnerParticipant(t *testing.T) {
	auth := happyAuthCreator()
	camp := &fakeCampaignCreator{
		createCampaign: func(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
			return &statev1.CreateCampaignResponse{
				Campaign: &statev1.Campaign{Id: "camp-1"},
			}, nil
		},
	}
	g := newTestGen(1, testDeps(camp, nil, nil, nil, nil, nil, auth))

	_, _, err := g.createCampaign(context.Background(), statev1.GmMode_HUMAN)
	if err == nil {
		t.Fatal("expected error for nil owner participant")
	}
}

func TestCreateCampaign_EmptyOwnerID(t *testing.T) {
	auth := happyAuthCreator()
	camp := &fakeCampaignCreator{
		createCampaign: func(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
			return &statev1.CreateCampaignResponse{
				Campaign:         &statev1.Campaign{Id: "camp-1"},
				OwnerParticipant: &statev1.Participant{Id: ""},
			}, nil
		},
	}
	g := newTestGen(1, testDeps(camp, nil, nil, nil, nil, nil, auth))

	_, _, err := g.createCampaign(context.Background(), statev1.GmMode_HUMAN)
	if err == nil {
		t.Fatal("expected error for empty owner ID")
	}
}

func TestTransitionCampaignStatus_DraftNoOp(t *testing.T) {
	g := newTestGen(1, testDeps(&fakeCampaignCreator{}, nil, nil, nil, nil, nil, nil))
	if err := g.transitionCampaignStatus(context.Background(), "camp-1", 0); err != nil {
		t.Fatalf("index 0 (DRAFT): unexpected error: %v", err)
	}
}

func TestTransitionCampaignStatus_ActiveNoOp(t *testing.T) {
	g := newTestGen(1, testDeps(&fakeCampaignCreator{}, nil, nil, nil, nil, nil, nil))
	if err := g.transitionCampaignStatus(context.Background(), "camp-1", 1); err != nil {
		t.Fatalf("index 1 (ACTIVE): unexpected error: %v", err)
	}
}

func TestTransitionCampaignStatus_CompletedEndsAndEnds(t *testing.T) {
	var endedSessions []string
	sess := &fakeSessionManager{
		listSessions: func(_ context.Context, in *statev1.ListSessionsRequest, _ ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
			return &statev1.ListSessionsResponse{
				Sessions: []*statev1.Session{
					{Id: "s1", Status: statev1.SessionStatus_SESSION_ACTIVE},
					{Id: "s2", Status: statev1.SessionStatus_SESSION_ENDED},
				},
			}, nil
		},
		endSession: func(_ context.Context, in *statev1.EndSessionRequest, _ ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
			endedSessions = append(endedSessions, in.SessionId)
			return &statev1.EndSessionResponse{}, nil
		},
	}
	var campaignEnded bool
	camp := &fakeCampaignCreator{
		endCampaign: func(context.Context, *statev1.EndCampaignRequest, ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
			campaignEnded = true
			return &statev1.EndCampaignResponse{}, nil
		},
	}
	g := newTestGen(1, testDeps(camp, nil, nil, nil, sess, nil, nil))

	if err := g.transitionCampaignStatus(context.Background(), "camp-1", 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(endedSessions) != 1 || endedSessions[0] != "s1" {
		t.Fatalf("expected only active session s1 ended, got %v", endedSessions)
	}
	if !campaignEnded {
		t.Fatal("expected campaign to be ended")
	}
}

func TestTransitionCampaignStatus_ArchivedEndsEndsArchives(t *testing.T) {
	sess := &fakeSessionManager{
		listSessions: func(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
			return &statev1.ListSessionsResponse{}, nil // no sessions
		},
	}
	var campaignEnded, campaignArchived bool
	camp := &fakeCampaignCreator{
		endCampaign: func(context.Context, *statev1.EndCampaignRequest, ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
			campaignEnded = true
			return &statev1.EndCampaignResponse{}, nil
		},
		archiveCampaign: func(context.Context, *statev1.ArchiveCampaignRequest, ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
			campaignArchived = true
			return &statev1.ArchiveCampaignResponse{}, nil
		},
	}
	g := newTestGen(1, testDeps(camp, nil, nil, nil, sess, nil, nil))

	if err := g.transitionCampaignStatus(context.Background(), "camp-1", 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !campaignEnded {
		t.Fatal("expected campaign to be ended")
	}
	if !campaignArchived {
		t.Fatal("expected campaign to be archived")
	}
}

func TestEndAllActiveSessions_EmptyList(t *testing.T) {
	sess := &fakeSessionManager{
		listSessions: func(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
			return &statev1.ListSessionsResponse{}, nil
		},
	}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, sess, nil, nil))

	if err := g.endAllActiveSessions(context.Background(), "camp-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEndAllActiveSessions_ActiveEndedNonActive(t *testing.T) {
	var ended []string
	sess := &fakeSessionManager{
		listSessions: func(_ context.Context, in *statev1.ListSessionsRequest, _ ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
			return &statev1.ListSessionsResponse{
				Sessions: []*statev1.Session{
					{Id: "s1", Status: statev1.SessionStatus_SESSION_ACTIVE},
					{Id: "s2", Status: statev1.SessionStatus_SESSION_ENDED},
					{Id: "s3", Status: statev1.SessionStatus_SESSION_ACTIVE},
				},
			}, nil
		},
		endSession: func(_ context.Context, in *statev1.EndSessionRequest, _ ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
			ended = append(ended, in.SessionId)
			return &statev1.EndSessionResponse{}, nil
		},
	}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, sess, nil, nil))

	if err := g.endAllActiveSessions(context.Background(), "camp-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ended) != 2 || ended[0] != "s1" || ended[1] != "s3" {
		t.Fatalf("expected s1,s3 ended, got %v", ended)
	}
}

func TestEndAllActiveSessions_Pagination(t *testing.T) {
	page := 0
	sess := &fakeSessionManager{
		listSessions: func(_ context.Context, in *statev1.ListSessionsRequest, _ ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
			page++
			if page == 1 {
				return &statev1.ListSessionsResponse{
					Sessions:      []*statev1.Session{{Id: "s1", Status: statev1.SessionStatus_SESSION_ACTIVE}},
					NextPageToken: "page2",
				}, nil
			}
			return &statev1.ListSessionsResponse{
				Sessions: []*statev1.Session{{Id: "s2", Status: statev1.SessionStatus_SESSION_ACTIVE}},
			}, nil
		},
		endSession: func(context.Context, *statev1.EndSessionRequest, ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
			return &statev1.EndSessionResponse{}, nil
		},
	}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, sess, nil, nil))

	if err := g.endAllActiveSessions(context.Background(), "camp-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != 2 {
		t.Fatalf("expected 2 pages fetched, got %d", page)
	}
}
