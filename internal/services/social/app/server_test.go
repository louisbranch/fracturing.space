package server

import (
	"context"
	"testing"
	"time"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestServer_AddListRemoveContactRoundTrip(t *testing.T) {
	client := newSocialClientForTest(t)

	if _, err := client.AddContact(context.Background(), &socialv1.AddContactRequest{
		OwnerUserId:   "user-1",
		ContactUserId: "user-2",
	}); err != nil {
		t.Fatalf("add contact: %v", err)
	}

	listResp, err := client.ListContacts(context.Background(), &socialv1.ListContactsRequest{
		OwnerUserId: "user-1",
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(listResp.GetContacts()) != 1 {
		t.Fatalf("contacts len = %d, want 1", len(listResp.GetContacts()))
	}

	if _, err := client.RemoveContact(context.Background(), &socialv1.RemoveContactRequest{
		OwnerUserId:   "user-1",
		ContactUserId: "user-2",
	}); err != nil {
		t.Fatalf("remove contact: %v", err)
	}

	listResp, err = client.ListContacts(context.Background(), &socialv1.ListContactsRequest{
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
	client := newSocialClientForTest(t)

	setResp, err := client.SetUserProfile(context.Background(), &socialv1.SetUserProfileRequest{
		UserId:        "user-1",
		Name:          "Alice",
		AvatarSetId:   "avatar_set_v1",
		AvatarAssetId: "apothecary_journeyman",
		Bio:           "Campaign manager",
	})
	if err != nil {
		t.Fatalf("set user profile: %v", err)
	}
	if got := setResp.GetUserProfile().GetName(); got != "Alice" {
		t.Fatalf("set name = %q, want Alice", got)
	}

	getResp, err := client.GetUserProfile(context.Background(), &socialv1.GetUserProfileRequest{
		UserId: "user-1",
	})
	if err != nil {
		t.Fatalf("get user profile: %v", err)
	}
	if got := getResp.GetUserProfile().GetBio(); got != "Campaign manager" {
		t.Fatalf("bio = %q, want Campaign manager", got)
	}
}

func TestServer_UserProfileAllowsMissingName(t *testing.T) {
	client := newSocialClientForTest(t)

	setResp, err := client.SetUserProfile(context.Background(), &socialv1.SetUserProfileRequest{
		UserId: "user-2",
	})
	if err != nil {
		t.Fatalf("set user profile: %v", err)
	}
	if got := setResp.GetUserProfile().GetName(); got != "" {
		t.Fatalf("set name = %q, want empty", got)
	}
	if got := setResp.GetUserProfile().GetAvatarSetId(); got != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("set avatar set id = %q, want %q", got, assetcatalog.AvatarSetPeopleV1)
	}
}

func TestServer_UserProfileNotFoundReturnsNotFound(t *testing.T) {
	client := newSocialClientForTest(t)

	_, err := client.GetUserProfile(context.Background(), &socialv1.GetUserProfileRequest{
		UserId: "missing-user",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("get user profile code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func newSocialClientForTest(t *testing.T) socialv1.SocialServiceClient {
	t.Helper()

	dbPath := t.TempDir() + "/social.db"
	t.Setenv("FRACTURING_SPACE_SOCIAL_DB_PATH", dbPath)

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
		t.Fatalf("dial social server: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Fatalf("close gRPC connection: %v", closeErr)
		}
	})

	return socialv1.NewSocialServiceClient(conn)
}
