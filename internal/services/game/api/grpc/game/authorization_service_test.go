package game

import (
	"context"
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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
			ctx:  contextWithParticipantID("owner-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
			},
			wantAllow:  true,
			wantReason: domainauthz.ReasonAllowAccessLevel,
		},
		{
			name: "manager cannot manage campaign",
			ctx:  contextWithParticipantID("manager-1"),
			request: &campaignv1.CanRequest{
				CampaignId: "c1",
				Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE,
				Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
			},
			wantAllow:  false,
			wantReason: domainauthz.ReasonDenyAccessLevelRequired,
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
			ctx:  contextWithParticipantID("member-1"),
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
			ctx:  contextWithParticipantID("member-1"),
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
			name: "admin override requires reason",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
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
}

func newAuthorizationServiceFixture(t *testing.T) *AuthorizationService {
	t.Helper()

	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1"}

	participantStore := newFakeParticipantStore()
	for _, record := range []storage.ParticipantRecord{
		{ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner, UserID: "owner-user"},
		{ID: "manager-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessManager, UserID: "manager-user"},
		{ID: "member-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessMember, UserID: "member-user"},
	} {
		if err := participantStore.PutParticipant(context.Background(), record); err != nil {
			t.Fatalf("put participant: %v", err)
		}
	}

	stores := Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   newFakeCharacterStore(),
	}
	return NewAuthorizationService(stores)
}
