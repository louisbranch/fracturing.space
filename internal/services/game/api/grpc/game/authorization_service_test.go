package game

import (
	"context"
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
			name: "manager can manage campaign",
			ctx:  contextWithParticipantID("manager-1"),
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
			name: "manager cannot mutate owner participant target",
			ctx:  contextWithParticipantID("manager-1"),
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
			ctx:  contextWithParticipantID("manager-1"),
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
			ctx:  contextWithParticipantID("owner-1"),
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
			ctx:  contextWithParticipantID("owner-1"),
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
			ctx:  contextWithParticipantID("owner-1"),
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
			ctx:  contextWithParticipantID("owner-1"),
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
			ctx:  contextWithParticipantID("owner-1"),
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

	_, err = svc.Can(contextWithParticipantID("owner-1"), &campaignv1.CanRequest{
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

func TestAuthorizationServiceBatchCan(t *testing.T) {
	svc := newAuthorizationServiceFixture(t)
	resp, err := svc.BatchCan(contextWithParticipantID("member-1"), &campaignv1.BatchCanRequest{
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

	_, err = svc.BatchCan(contextWithParticipantID("owner-1"), &campaignv1.BatchCanRequest{
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

	eventStore := newFakeEventStore()
	if _, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "c1",
		Type:        eventTypeCharacterCreated,
		EntityType:  "character",
		EntityID:    "char-member-1",
		ActorType:   event.ActorTypeParticipant,
		ActorID:     "member-1",
		PayloadJSON: []byte(`{"character_id":"char-member-1","owner_participant_id":"member-1","name":"Member Hero","kind":"pc"}`),
	}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	characterStore := newFakeCharacterStore()
	if err := characterStore.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-member-1",
		CampaignID:         "c1",
		OwnerParticipantID: "member-1",
		Name:               "Member Hero",
		Kind:               character.KindPC,
	}); err != nil {
		t.Fatalf("put character: %v", err)
	}

	stores := Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Event:       eventStore,
	}
	return NewAuthorizationService(stores)
}
