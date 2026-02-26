package campaigns

import (
	"context"
	"errors"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
)

func TestBatchCanCampaignActionMapsResults(t *testing.T) {
	t.Parallel()

	gateway := grpcGateway{
		authorizationClient: fakeAuthorizationClient{
			batchCanResponse: &statev1.BatchCanResponse{Results: []*statev1.BatchCanResult{
				{CheckId: "char-a", Allowed: true, ReasonCode: "AUTHZ_ALLOW_RESOURCE_OWNER"},
				{CheckId: "char-b", Allowed: false, ReasonCode: "AUTHZ_DENY_NOT_RESOURCE_OWNER"},
			}},
		},
	}

	decisions, err := gateway.BatchCanCampaignAction(context.Background(), "c1", []campaignAuthorizationCheck{
		{
			CheckID:  "char-a",
			Action:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
			Resource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
			Target:   &statev1.AuthorizationTarget{ResourceId: "char-a"},
		},
		{
			CheckID:  "char-b",
			Action:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
			Resource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
			Target:   &statev1.AuthorizationTarget{ResourceId: "char-b"},
		},
	})
	if err != nil {
		t.Fatalf("BatchCanCampaignAction() error = %v", err)
	}
	if len(decisions) != 2 {
		t.Fatalf("len(decisions) = %d, want 2", len(decisions))
	}
	if decisions[0].CheckID != "char-a" || !decisions[0].Allowed || decisions[0].ReasonCode != "AUTHZ_ALLOW_RESOURCE_OWNER" {
		t.Fatalf("decisions[0] = %#v", decisions[0])
	}
	if decisions[1].CheckID != "char-b" || decisions[1].Allowed || decisions[1].ReasonCode != "AUTHZ_DENY_NOT_RESOURCE_OWNER" {
		t.Fatalf("decisions[1] = %#v", decisions[1])
	}
}

func TestBatchCanCampaignActionFallsBackToRequestCheckID(t *testing.T) {
	t.Parallel()

	gateway := grpcGateway{
		authorizationClient: fakeAuthorizationClient{
			batchCanResponse: &statev1.BatchCanResponse{Results: []*statev1.BatchCanResult{{Allowed: true, ReasonCode: "AUTHZ_ALLOW_RESOURCE_OWNER"}}},
		},
	}

	decisions, err := gateway.BatchCanCampaignAction(context.Background(), "c1", []campaignAuthorizationCheck{
		{
			CheckID:  "char-a",
			Action:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
			Resource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
			Target:   &statev1.AuthorizationTarget{ResourceId: "char-a"},
		},
	})
	if err != nil {
		t.Fatalf("BatchCanCampaignAction() error = %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("len(decisions) = %d, want 1", len(decisions))
	}
	if decisions[0].CheckID != "char-a" {
		t.Fatalf("decisions[0].CheckID = %q, want %q", decisions[0].CheckID, "char-a")
	}
}

func TestBatchCanCampaignActionFailsWithClientError(t *testing.T) {
	t.Parallel()

	gateway := grpcGateway{authorizationClient: fakeAuthorizationClient{batchCanErr: errors.New("auth unavailable")}}
	_, err := gateway.BatchCanCampaignAction(context.Background(), "c1", []campaignAuthorizationCheck{{CheckID: "char-a"}})
	if err == nil {
		t.Fatal("expected BatchCanCampaignAction() error")
	}
}

type fakeAuthorizationClient struct {
	batchCanResponse *statev1.BatchCanResponse
	batchCanErr      error
}

func (f fakeAuthorizationClient) Can(context.Context, *statev1.CanRequest, ...grpc.CallOption) (*statev1.CanResponse, error) {
	return &statev1.CanResponse{}, nil
}

func (f fakeAuthorizationClient) BatchCan(context.Context, *statev1.BatchCanRequest, ...grpc.CallOption) (*statev1.BatchCanResponse, error) {
	if f.batchCanErr != nil {
		return nil, f.batchCanErr
	}
	if f.batchCanResponse != nil {
		return f.batchCanResponse, nil
	}
	return &statev1.BatchCanResponse{}, nil
}
