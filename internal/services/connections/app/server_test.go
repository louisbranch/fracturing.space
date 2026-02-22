package server

import (
	"context"
	"testing"
	"time"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServer_AddListRemoveContactRoundTrip(t *testing.T) {
	dbPath := t.TempDir() + "/connections.db"
	t.Setenv("FRACTURING_SPACE_CONNECTIONS_DB_PATH", dbPath)

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
		t.Fatalf("dial connections server: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Fatalf("close gRPC connection: %v", closeErr)
		}
	})

	client := connectionsv1.NewConnectionsServiceClient(conn)

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

func TestServer_SetAndLookupPublicProfileRoundTrip(t *testing.T) {
	dbPath := t.TempDir() + "/connections.db"
	t.Setenv("FRACTURING_SPACE_CONNECTIONS_DB_PATH", dbPath)

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
		t.Fatalf("dial connections server: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Fatalf("close gRPC connection: %v", closeErr)
		}
	})

	client := connectionsv1.NewConnectionsServiceClient(conn)

	_, err = client.SetUsername(context.Background(), &connectionsv1.SetUsernameRequest{
		UserId:   "user-1",
		Username: "alice_one",
	})
	if err != nil {
		t.Fatalf("set username: %v", err)
	}

	_, err = client.SetPublicProfile(context.Background(), &connectionsv1.SetPublicProfileRequest{
		UserId:      "user-1",
		DisplayName: "Alice",
		AvatarUrl:   "https://cdn.example.com/avatar/alice.png",
		Bio:         "Campaign manager",
	})
	if err != nil {
		t.Fatalf("set public profile: %v", err)
	}

	getResp, err := client.GetPublicProfile(context.Background(), &connectionsv1.GetPublicProfileRequest{
		UserId: "user-1",
	})
	if err != nil {
		t.Fatalf("get public profile: %v", err)
	}
	if got := getResp.GetPublicProfileRecord().GetDisplayName(); got != "Alice" {
		t.Fatalf("display_name = %q, want Alice", got)
	}

	lookupResp, err := client.LookupPublicProfile(context.Background(), &connectionsv1.LookupPublicProfileRequest{
		Username: "ALICE_ONE",
	})
	if err != nil {
		t.Fatalf("lookup public profile: %v", err)
	}
	if got := lookupResp.GetUsernameRecord().GetUserId(); got != "user-1" {
		t.Fatalf("lookup user_id = %q, want user-1", got)
	}
	if got := lookupResp.GetPublicProfileRecord().GetBio(); got != "Campaign manager" {
		t.Fatalf("lookup bio = %q, want Campaign manager", got)
	}
}
