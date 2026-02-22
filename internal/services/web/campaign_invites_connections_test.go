package web

import (
	"context"
	"testing"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestListAllContactsUsesConnectionsClient(t *testing.T) {
	client := &fakeConnectionsClient{
		listContactsResp: &connectionsv1.ListContactsResponse{
			Contacts: []*connectionsv1.Contact{
				{OwnerUserId: "user-1", ContactUserId: "user-2"},
			},
		},
	}
	h := &handler{connectionsClient: client}

	contacts, err := h.listAllContacts(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list all contacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("contacts len = %d, want 1", len(contacts))
	}
	if client.listContactsReq == nil {
		t.Fatal("expected ListContacts request")
	}
	if got := client.listContactsReq.GetOwnerUserId(); got != "user-1" {
		t.Fatalf("owner_user_id = %q, want user-1", got)
	}
}

type fakeConnectionsClient struct {
	listContactsResp       *connectionsv1.ListContactsResponse
	listContactsPages      map[string]*connectionsv1.ListContactsResponse
	listContactsCalls      int
	listContactsReq        *connectionsv1.ListContactsRequest
	setUserProfileResp     *connectionsv1.SetUserProfileResponse
	setUserProfileErr      error
	setUserProfileReq      *connectionsv1.SetUserProfileRequest
	setUserProfileMD       metadata.MD
	getUserProfileResp     *connectionsv1.GetUserProfileResponse
	getUserProfileErr      error
	getUserProfileReq      *connectionsv1.GetUserProfileRequest
	getUserProfileMD       metadata.MD
	lookupUserProfileResp  *connectionsv1.LookupUserProfileResponse
	lookupUserProfileErr   error
	lookupUserProfileReq   *connectionsv1.LookupUserProfileRequest
	lookupUserProfileMD    metadata.MD
	lookupUserProfileCalls int
}

func (f *fakeConnectionsClient) AddContact(context.Context, *connectionsv1.AddContactRequest, ...grpc.CallOption) (*connectionsv1.AddContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "fakeConnectionsClient.AddContact not implemented")
}

func (f *fakeConnectionsClient) RemoveContact(context.Context, *connectionsv1.RemoveContactRequest, ...grpc.CallOption) (*connectionsv1.RemoveContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "fakeConnectionsClient.RemoveContact not implemented")
}

func (f *fakeConnectionsClient) SetUserProfile(ctx context.Context, req *connectionsv1.SetUserProfileRequest, _ ...grpc.CallOption) (*connectionsv1.SetUserProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.setUserProfileReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	f.setUserProfileMD = md
	if f.setUserProfileErr != nil {
		return nil, f.setUserProfileErr
	}
	if f.setUserProfileResp != nil {
		return f.setUserProfileResp, nil
	}
	return nil, status.Error(codes.NotFound, "set user profile not configured")
}

func (f *fakeConnectionsClient) GetUserProfile(ctx context.Context, req *connectionsv1.GetUserProfileRequest, _ ...grpc.CallOption) (*connectionsv1.GetUserProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.getUserProfileReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	f.getUserProfileMD = md
	if f.getUserProfileErr != nil {
		return nil, f.getUserProfileErr
	}
	if f.getUserProfileResp != nil {
		return f.getUserProfileResp, nil
	}
	return nil, status.Error(codes.NotFound, "get user profile not configured")
}

func (f *fakeConnectionsClient) ListContacts(ctx context.Context, req *connectionsv1.ListContactsRequest, _ ...grpc.CallOption) (*connectionsv1.ListContactsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.listContactsCalls++
	f.listContactsReq = req
	if f.listContactsPages != nil {
		if resp, ok := f.listContactsPages[req.GetPageToken()]; ok {
			return resp, nil
		}
	}
	return f.listContactsResp, nil
}

func (f *fakeConnectionsClient) LookupUserProfile(ctx context.Context, req *connectionsv1.LookupUserProfileRequest, _ ...grpc.CallOption) (*connectionsv1.LookupUserProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.lookupUserProfileCalls++
	f.lookupUserProfileReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	f.lookupUserProfileMD = md
	if f.lookupUserProfileErr != nil {
		return nil, f.lookupUserProfileErr
	}
	if f.lookupUserProfileResp == nil {
		return nil, status.Error(codes.NotFound, "lookup user profile not configured")
	}
	return f.lookupUserProfileResp, nil
}
