package authorizationtransport

import (
	"context"
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"google.golang.org/grpc/metadata"
)

func TestAuthorizationServiceCan(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		request    *campaignv1.CanRequest
		wantAllow  bool
		wantReason string
	}{
		{
			name: "owner can manage campaign",
			ctx:  requestctx.WithParticipantID("owner-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
			},
			wantAllow:  true,
			wantReason: domainauthz.ReasonAllowAccessLevel,
		},
		{
			name: "manager can manage campaign",
			ctx:  requestctx.WithParticipantID("manager-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
			},
			wantAllow:  true,
			wantReason: domainauthz.ReasonAllowAccessLevel,
		},
		{
			name: "missing identity denied",
			ctx:  context.Background(),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
			},
			wantAllow:  false,
			wantReason: domainauthz.ReasonDenyMissingIdentity,
		},
		{
			name: "member character mutation requires ownership",
			ctx:  requestctx.WithParticipantID("member-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
				Target: &campaignv1.AuthorizationTarget{
					OwnerParticipantId: "owner-1",
				},
			},
			wantAllow:  false,
			wantReason: domainauthz.ReasonDenyNotResourceOwner,
		},
		{
			name: "member character mutation owned passes",
			ctx:  requestctx.WithParticipantID("member-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
				Target: &campaignv1.AuthorizationTarget{
					OwnerParticipantId: "member-1",
				},
			},
			wantAllow:  true,
			wantReason: domainauthz.ReasonAllowResourceOwner,
		},
		{
			name: "manager cannot mutate owner participant target",
			ctx:  requestctx.WithParticipantID("manager-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
				Target: &campaignv1.AuthorizationTarget{
					TargetParticipantId:  "owner-1",
					TargetCampaignAccess: campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
				},
			},
			wantAllow:  false,
			wantReason: domainauthz.ReasonDenyTargetIsOwner,
		},
		{
			name: "manager cannot assign owner campaign access",
			ctx:  requestctx.WithParticipantID("manager-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
				Target: &campaignv1.AuthorizationTarget{
					TargetParticipantId:     "member-1",
					TargetCampaignAccess:    campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					RequestedCampaignAccess: campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
				},
			},
			wantAllow:  false,
			wantReason: domainauthz.ReasonDenyManagerOwnerMutationForbidden,
		},
		{
			name: "owner cannot demote final owner",
			ctx:  requestctx.WithParticipantID("owner-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
				Target: &campaignv1.AuthorizationTarget{
					TargetParticipantId:     "owner-1",
					TargetCampaignAccess:    campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
					RequestedCampaignAccess: campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
				},
			},
			wantAllow:  false,
			wantReason: domainauthz.ReasonDenyLastOwnerGuard,
		},
		{
			name: "owner remove operation denies final owner",
			ctx:  requestctx.WithParticipantID("owner-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
				Target: &campaignv1.AuthorizationTarget{
					TargetParticipantId:  "owner-1",
					TargetCampaignAccess: campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
					ParticipantOperation: campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_REMOVE,
				},
			},
			wantAllow:  false,
			wantReason: domainauthz.ReasonDenyLastOwnerGuard,
		},
		{
			name: "owner remove operation denies target owning active characters",
			ctx:  requestctx.WithParticipantID("owner-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
				Target: &campaignv1.AuthorizationTarget{
					TargetParticipantId:  "member-1",
					TargetCampaignAccess: campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					ParticipantOperation: campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_REMOVE,
				},
			},
			wantAllow:  false,
			wantReason: domainauthz.ReasonDenyTargetOwnsActiveCharacters,
		},
		{
			name: "owner mutate operation allows owner target",
			ctx:  requestctx.WithParticipantID("owner-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
				Target: &campaignv1.AuthorizationTarget{
					TargetParticipantId:  "owner-1",
					TargetCampaignAccess: campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
					ParticipantOperation: campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_MUTATE,
				},
			},
			wantAllow:  true,
			wantReason: domainauthz.ReasonAllowAccessLevel,
		},
		{
			name: "owner can promote member to manager",
			ctx:  requestctx.WithParticipantID("owner-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
				Target: &campaignv1.AuthorizationTarget{
					TargetParticipantId:     "member-1",
					TargetCampaignAccess:    campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					RequestedCampaignAccess: campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
				},
			},
			wantAllow:  true,
			wantReason: domainauthz.ReasonAllowAccessLevel,
		},
		{
			name: "admin override requires reason",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
				grpcmeta.UserIDHeader, "user-admin-1",
			)),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
			},
			wantAllow:  false,
			wantReason: domainauthz.ReasonDenyOverrideReasonRequired,
		},
		{
			name: "admin override with reason allowed",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
				grpcmeta.AuthzOverrideReasonHeader, "incident-ops",
				grpcmeta.UserIDHeader, "user-admin-1",
			)),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
			},
			wantAllow:  true,
			wantReason: domainauthz.ReasonAllowAdminOverride,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newAuthorizationServiceFixture(t)
			resp, err := svc.Can(tt.ctx, tt.request)
			if err != nil {
				t.Fatalf("Can returned error: %v", err)
			}
			if resp.GetAllowed() != tt.wantAllow {
				t.Fatalf("allowed = %v, want %v", resp.GetAllowed(), tt.wantAllow)
			}
			if resp.GetReasonCode() != tt.wantReason {
				t.Fatalf("reason_code = %q, want %q", resp.GetReasonCode(), tt.wantReason)
			}
		})
	}
}

func TestAuthorizationServiceCanRejectsInvalidRequests(t *testing.T) {
	svc := newAuthorizationServiceFixture(t)
	_, err := svc.Can(context.Background(), nil)
	if err == nil {
		t.Fatal("expected nil request to fail")
	}

	_, err = svc.Can(context.Background(), &campaignv1.CanRequest{CampaignId: "c1"})
	if err == nil {
		t.Fatal("expected missing action/resource to fail")
	}

	_, err = svc.Can(requestctx.WithParticipantID("owner-1"), &campaignv1.CanRequest{
		CampaignId: "c1",
		Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
		Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT,
		Target: &campaignv1.AuthorizationTarget{
			TargetParticipantId:  "member-1",
			ParticipantOperation: campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_ACCESS_CHANGE,
		},
	})
	if err == nil {
		t.Fatal("expected access-change operation without requested access to fail")
	}
}
