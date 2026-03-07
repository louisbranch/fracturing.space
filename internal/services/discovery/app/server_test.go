package server

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServer_CreateGetAndListDiscoveryEntriesRoundTrip(t *testing.T) {
	dbPath := t.TempDir() + "/discovery.db"
	t.Setenv("FRACTURING_SPACE_DISCOVERY_DB_PATH", dbPath)

	srv, err := NewWithAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve(runCtx)
	}()
	t.Cleanup(func() {
		runCancel()
		select {
		case serveErr := <-serveDone:
			if serveErr != nil {
				t.Fatalf("serve: %v", serveErr)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for server shutdown")
		}
	})

	conn, err := grpc.NewClient(srv.Addr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial discovery server: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := discoveryv1.NewDiscoveryServiceClient(conn)

	createResp, err := client.CreateDiscoveryEntry(context.Background(), &discoveryv1.CreateDiscoveryEntryRequest{
		Entry: &discoveryv1.DiscoveryEntry{
			EntryId:                    "starter:camp-1",
			Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
			SourceId:                   "camp-1",
			Title:                      "Sunfall",
			Description:                "A haunted valley campaign",
			RecommendedParticipantsMin: 3,
			RecommendedParticipantsMax: 5,
			DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
			ExpectedDurationLabel:      "2-3 sessions",
			System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		},
	})
	if err != nil {
		t.Fatalf("create discovery entry: %v", err)
	}
	if got := createResp.GetEntry().GetEntryId(); got != "starter:camp-1" {
		t.Fatalf("entry_id = %q, want starter:camp-1", got)
	}

	getResp, err := client.GetDiscoveryEntry(context.Background(), &discoveryv1.GetDiscoveryEntryRequest{EntryId: "starter:camp-1"})
	if err != nil {
		t.Fatalf("get discovery entry: %v", err)
	}
	if got := getResp.GetEntry().GetTitle(); got != "Sunfall" {
		t.Fatalf("title = %q, want Sunfall", got)
	}

	listResp, err := client.ListDiscoveryEntries(context.Background(), &discoveryv1.ListDiscoveryEntriesRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list discovery entries: %v", err)
	}
	if len(listResp.GetEntries()) < 4 {
		t.Fatalf("entries len = %d, want at least 4 (3 builtin + 1 created)", len(listResp.GetEntries()))
	}
}
