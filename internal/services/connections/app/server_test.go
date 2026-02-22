package server

import (
	"context"
	"testing"
	"time"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestServer_AddListRemoveContactRoundTrip(t *testing.T) {
	client := newConnectionsClientForTest(t)

	if _, err := client.AddContact(context.Background(), &connectionsv1.AddContactRequest{
		OwnerUserId:   "user-1",
		ContactUserId: "user-2",
	}); err != nil {
		t.Fatalf("add contact: %v", err)
	}

	listResp, err := client.ListContacts(context.Background(), &connectionsv1.ListContactsRequest{
		OwnerUserId: "user-1",
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(listResp.GetContacts()) != 1 {
		t.Fatalf("contacts len = %d, want 1", len(listResp.GetContacts()))
	}

	if _, err := client.RemoveContact(context.Background(), &connectionsv1.RemoveContactRequest{
		OwnerUserId:   "user-1",
		ContactUserId: "user-2",
	}); err != nil {
		t.Fatalf("remove contact: %v", err)
	}

	listResp, err = client.ListContacts(context.Background(), &connectionsv1.ListContactsRequest{
		OwnerUserId: "user-1",
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("list contacts after remove: %v", err)
	}
	if len(listResp.GetContacts()) != 0 {
		t.Fatalf("contacts len after remove = %d, want 0", len(listResp.GetContacts()))
	}
}

func TestServer_UserProfileRoundTrip(t *testing.T) {
	client := newConnectionsClientForTest(t)

	setResp, err := client.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
		UserId:        "user-1",
		Username:      "Alice_One",
		Name:          "Alice",
		AvatarSetId:   "avatar_set_v1",
		AvatarAssetId: "001",
		Bio:           "Campaign manager",
	})
	if err != nil {
		t.Fatalf("set user profile: %v", err)
	}
	if setResp.GetUserProfileRecord() == nil {
		t.Fatal("expected user profile record from set user profile")
	}
	if got := setResp.GetUserProfileRecord().GetUsername(); got != "alice_one" {
		t.Fatalf("set username = %q, want alice_one", got)
	}

	getResp, err := client.GetUserProfile(context.Background(), &connectionsv1.GetUserProfileRequest{
		UserId: "user-1",
	})
	if err != nil {
		t.Fatalf("get user profile: %v", err)
	}
	if got := getResp.GetUserProfileRecord().GetName(); got != "Alice" {
		t.Fatalf("name = %q, want Alice", got)
	}

	lookupResp, err := client.LookupUserProfile(context.Background(), &connectionsv1.LookupUserProfileRequest{
		Username: "ALICE_ONE",
	})
	if err != nil {
		t.Fatalf("lookup user profile: %v", err)
	}
	if got := lookupResp.GetUserProfileRecord().GetUserId(); got != "user-1" {
		t.Fatalf("lookup user_id = %q, want user-1", got)
	}
	if got := lookupResp.GetUserProfileRecord().GetBio(); got != "Campaign manager" {
		t.Fatalf("lookup bio = %q, want Campaign manager", got)
	}
}

func TestServer_UserProfileConflictReturnsAlreadyExists(t *testing.T) {
	client := newConnectionsClientForTest(t)

	if _, err := client.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
		UserId:   "user-1",
		Username: "taken",
		Name:     "Alice",
	}); err != nil {
		t.Fatalf("set user profile user-1: %v", err)
	}

	_, err := client.SetUserProfile(context.Background(), &connectionsv1.SetUserProfileRequest{
		UserId:   "user-2",
		Username: "Taken",
		Name:     "Bob",
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.AlreadyExists)
	}
}

func TestServer_UserProfileNotFoundReturnsNotFound(t *testing.T) {
	client := newConnectionsClientForTest(t)

	_, err := client.GetUserProfile(context.Background(), &connectionsv1.GetUserProfileRequest{
		UserId: "missing-user",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("get user profile code = %v, want %v", status.Code(err), codes.NotFound)
	}

	_, err = client.LookupUserProfile(context.Background(), &connectionsv1.LookupUserProfileRequest{
		Username: "missing-user",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("lookup user profile code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func newConnectionsClientForTest(t *testing.T) connectionsv1.ConnectionsServiceClient {
	t.Helper()

	dbPath := t.TempDir() + "/connections.db"
	authDBPath := t.TempDir() + "/auth.db"
	t.Setenv("FRACTURING_SPACE_CONNECTIONS_DB_PATH", dbPath)
	t.Setenv("FRACTURING_SPACE_AUTH_DB_PATH", authDBPath)
	t.Setenv("FRACTURING_SPACE_CONNECTIONS_MIGRATE_AUTH_CONTACTS", "false")

	srv, err := NewWithAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	runCtx, runCancel := context.WithCancel(context.Background())
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
		t.Fatalf("dial connections server: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Fatalf("close gRPC connection: %v", closeErr)
		}
	})

	return connectionsv1.NewConnectionsServiceClient(conn)
}
