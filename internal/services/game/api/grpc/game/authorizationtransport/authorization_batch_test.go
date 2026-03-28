package authorizationtransport

import (
	"context"
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthorizationServiceBatchCan(t *testing.T) {
	svc := newAuthorizationServiceFixture(t)
	resp, err := svc.BatchCan(requestctx.WithParticipantID("member-1"), &campaignv1.BatchCanRequest{
		Checks: []*campaignv1.BatchCanCheck{
			{
				CheckId:    "char-member-1",
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
				Target: &campaignv1.AuthorizationTarget{
					ResourceId: "char-member-1",
				},
			},
			{
				CheckId:    "char-owner-1",
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
				Target: &campaignv1.AuthorizationTarget{
					OwnerParticipantId: "owner-1",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BatchCan returned error: %v", err)
	}
	if got := len(resp.GetResults()); got != 2 {
		t.Fatalf("len(results) = %d, want 2", got)
	}
	if got := resp.GetResults()[0].GetCheckId(); got != "char-member-1" {
		t.Fatalf("result[0].check_id = %q, want %q", got, "char-member-1")
	}
	if got := resp.GetResults()[0].GetAllowed(); !got {
		t.Fatalf("result[0].allowed = %v, want true", got)
	}
	if got := resp.GetResults()[0].GetReasonCode(); got != domainauthz.ReasonAllowResourceOwner {
		t.Fatalf("result[0].reason_code = %q, want %q", got, domainauthz.ReasonAllowResourceOwner)
	}
	if got := resp.GetResults()[1].GetCheckId(); got != "char-owner-1" {
		t.Fatalf("result[1].check_id = %q, want %q", got, "char-owner-1")
	}
	if got := resp.GetResults()[1].GetAllowed(); got {
		t.Fatalf("result[1].allowed = %v, want false", got)
	}
	if got := resp.GetResults()[1].GetReasonCode(); got != domainauthz.ReasonDenyNotResourceOwner {
		t.Fatalf("result[1].reason_code = %q, want %q", got, domainauthz.ReasonDenyNotResourceOwner)
	}
}

func TestAuthorizationServiceBatchCanRejectsInvalidRequests(t *testing.T) {
	svc := newAuthorizationServiceFixture(t)

	_, err := svc.BatchCan(context.Background(), nil)
	if err == nil {
		t.Fatal("expected nil batch request to fail")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}

	_, err = svc.BatchCan(context.Background(), &campaignv1.BatchCanRequest{})
	if err == nil {
		t.Fatal("expected empty batch checks to fail")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}

	_, err = svc.BatchCan(context.Background(), &campaignv1.BatchCanRequest{
		Checks: []*campaignv1.BatchCanCheck{nil},
	})
	if err == nil {
		t.Fatal("expected nil batch check to fail")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}

	_, err = svc.BatchCan(requestctx.WithParticipantID("owner-1"), &campaignv1.BatchCanRequest{
		Checks: []*campaignv1.BatchCanCheck{
			{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
			},
			{
				Action:   campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource: campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
			},
		},
	})
	if err == nil {
		t.Fatal("expected invalid batch item to fail-fast")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
}
