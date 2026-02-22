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
	listContactsResp        *connectionsv1.ListContactsResponse
	listContactsPages       map[string]*connectionsv1.ListContactsResponse
	listContactsCalls       int
	listContactsReq         *connectionsv1.ListContactsRequest
	setUsernameResp         *connectionsv1.SetUsernameResponse
	setUsernameErr          error
	setUsernameReq          *connectionsv1.SetUsernameRequest
	setUsernameMD           metadata.MD
	getUsernameResp         *connectionsv1.GetUsernameResponse
	getUsernameErr          error
	getUsernameReq          *connectionsv1.GetUsernameRequest
	getUsernameMD           metadata.MD
	setPublicProfileResp    *connectionsv1.SetPublicProfileResponse
	setPublicProfileErr     error
	setPublicProfileReq     *connectionsv1.SetPublicProfileRequest
	setPublicProfileMD      metadata.MD
	getPublicProfileResp    *connectionsv1.GetPublicProfileResponse
	getPublicProfileErr     error
	getPublicProfileReq     *connectionsv1.GetPublicProfileRequest
	getPublicProfileMD      metadata.MD
	lookupPublicProfileResp *connectionsv1.LookupPublicProfileResponse
	lookupPublicProfileErr  error
	lookupPublicProfileReq  *connectionsv1.LookupPublicProfileRequest
	lookupPublicProfileMD   metadata.MD
	lookupUsernameResp      *connectionsv1.LookupUsernameResponse
	lookupUsernameErr       error
	lookupUsernameReq       *connectionsv1.LookupUsernameRequest
	lookupUsernameMD        metadata.MD
	lookupUsernameCalls     int
}

func (f *fakeConnectionsClient) AddContact(context.Context, *connectionsv1.AddContactRequest, ...grpc.CallOption) (*connectionsv1.AddContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "fakeConnectionsClient.AddContact not implemented")
}

func (f *fakeConnectionsClient) RemoveContact(context.Context, *connectionsv1.RemoveContactRequest, ...grpc.CallOption) (*connectionsv1.RemoveContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "fakeConnectionsClient.RemoveContact not implemented")
}

func (f *fakeConnectionsClient) SetUsername(ctx context.Context, req *connectionsv1.SetUsernameRequest, _ ...grpc.CallOption) (*connectionsv1.SetUsernameResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.setUsernameReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	f.setUsernameMD = md
	if f.setUsernameErr != nil {
		return nil, f.setUsernameErr
	}
	if f.setUsernameResp != nil {
		return f.setUsernameResp, nil
	}
	return nil, status.Error(codes.NotFound, "set username not configured")
}

func (f *fakeConnectionsClient) GetUsername(ctx context.Context, req *connectionsv1.GetUsernameRequest, _ ...grpc.CallOption) (*connectionsv1.GetUsernameResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.getUsernameReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	f.getUsernameMD = md
	if f.getUsernameErr != nil {
		return nil, f.getUsernameErr
	}
	if f.getUsernameResp != nil {
		return f.getUsernameResp, nil
	}
	return nil, status.Error(codes.NotFound, "get username not configured")
}

func (f *fakeConnectionsClient) SetPublicProfile(ctx context.Context, req *connectionsv1.SetPublicProfileRequest, _ ...grpc.CallOption) (*connectionsv1.SetPublicProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.setPublicProfileReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	f.setPublicProfileMD = md
	if f.setPublicProfileErr != nil {
		return nil, f.setPublicProfileErr
	}
	if f.setPublicProfileResp != nil {
		return f.setPublicProfileResp, nil
	}
	return nil, status.Error(codes.NotFound, "set public profile not configured")
}

func (f *fakeConnectionsClient) GetPublicProfile(ctx context.Context, req *connectionsv1.GetPublicProfileRequest, _ ...grpc.CallOption) (*connectionsv1.GetPublicProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.getPublicProfileReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	f.getPublicProfileMD = md
	if f.getPublicProfileErr != nil {
		return nil, f.getPublicProfileErr
	}
	if f.getPublicProfileResp != nil {
		return f.getPublicProfileResp, nil
	}
	return nil, status.Error(codes.NotFound, "get public profile not configured")
}

func (f *fakeConnectionsClient) LookupPublicProfile(ctx context.Context, req *connectionsv1.LookupPublicProfileRequest, _ ...grpc.CallOption) (*connectionsv1.LookupPublicProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.lookupPublicProfileReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	f.lookupPublicProfileMD = md
	if f.lookupPublicProfileErr != nil {
		return nil, f.lookupPublicProfileErr
	}
	if f.lookupPublicProfileResp != nil {
		return f.lookupPublicProfileResp, nil
	}
	return nil, status.Error(codes.NotFound, "lookup public profile not configured")
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

func (f *fakeConnectionsClient) LookupUsername(ctx context.Context, req *connectionsv1.LookupUsernameRequest, _ ...grpc.CallOption) (*connectionsv1.LookupUsernameResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.lookupUsernameCalls++
	f.lookupUsernameReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	f.lookupUsernameMD = md
	if f.lookupUsernameErr != nil {
		return nil, f.lookupUsernameErr
	}
	if f.lookupUsernameResp == nil {
		return nil, status.Error(codes.NotFound, "lookup username not configured")
	}
	return f.lookupUsernameResp, nil
}
