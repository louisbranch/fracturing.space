package web

import (
	"context"
	"testing"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
	listContactsResp  *connectionsv1.ListContactsResponse
	listContactsPages map[string]*connectionsv1.ListContactsResponse
	listContactsCalls int
	listContactsReq   *connectionsv1.ListContactsRequest
}

func (f *fakeConnectionsClient) AddContact(context.Context, *connectionsv1.AddContactRequest, ...grpc.CallOption) (*connectionsv1.AddContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "fakeConnectionsClient.AddContact not implemented")
}

func (f *fakeConnectionsClient) RemoveContact(context.Context, *connectionsv1.RemoveContactRequest, ...grpc.CallOption) (*connectionsv1.RemoveContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "fakeConnectionsClient.RemoveContact not implemented")
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
