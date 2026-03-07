package game

import (
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestCanResponseTrimsReasonAndActorID(t *testing.T) {
	resp := canResponse(true, "  allow.read  ", storage.ParticipantRecord{
		ID:             "  p-1  ",
		CampaignAccess: participant.CampaignAccessManager,
	})
	if !resp.GetAllowed() {
		t.Fatal("allowed = false, want true")
	}
	if resp.GetReasonCode() != "allow.read" {
		t.Fatalf("reason_code = %q, want %q", resp.GetReasonCode(), "allow.read")
	}
	if resp.GetActorParticipantId() != "p-1" {
		t.Fatalf("actor_participant_id = %q, want %q", resp.GetActorParticipantId(), "p-1")
	}
	if resp.GetActorCampaignAccess() != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Fatalf("actor_campaign_access = %v, want %v", resp.GetActorCampaignAccess(), campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER)
	}
}

func TestAuthorizationActionFromProto(t *testing.T) {
	tests := []struct {
		name   string
		action campaignv1.AuthorizationAction
		want   domainauthz.Action
		wantOK bool
	}{
		{name: "read", action: campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_READ, want: domainauthz.ActionRead, wantOK: true},
		{name: "manage", action: campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE, want: domainauthz.ActionManage, wantOK: true},
		{name: "mutate", action: campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE, want: domainauthz.ActionMutate, wantOK: true},
		{name: "transfer ownership", action: campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_TRANSFER_OWNERSHIP, want: domainauthz.ActionTransferOwnership, wantOK: true},
		{name: "unspecified", action: campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_UNSPECIFIED, want: domainauthz.ActionUnspecified, wantOK: false},
		{name: "unknown", action: campaignv1.AuthorizationAction(-1), want: domainauthz.ActionUnspecified, wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := authorizationActionFromProto(tc.action)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("authorizationActionFromProto(%v) = (%v,%v), want (%v,%v)", tc.action, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestAuthorizationResourceFromProto(t *testing.T) {
	tests := []struct {
		name     string
		resource campaignv1.AuthorizationResource
		want     domainauthz.Resource
		wantOK   bool
	}{
		{name: "campaign", resource: campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN, want: domainauthz.ResourceCampaign, wantOK: true},
		{name: "participant", resource: campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT, want: domainauthz.ResourceParticipant, wantOK: true},
		{name: "invite", resource: campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_INVITE, want: domainauthz.ResourceInvite, wantOK: true},
		{name: "session", resource: campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION, want: domainauthz.ResourceSession, wantOK: true},
		{name: "character", resource: campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER, want: domainauthz.ResourceCharacter, wantOK: true},
		{name: "unspecified", resource: campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_UNSPECIFIED, want: domainauthz.ResourceUnspecified, wantOK: false},
		{name: "unknown", resource: campaignv1.AuthorizationResource(-1), want: domainauthz.ResourceUnspecified, wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := authorizationResourceFromProto(tc.resource)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("authorizationResourceFromProto(%v) = (%v,%v), want (%v,%v)", tc.resource, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
