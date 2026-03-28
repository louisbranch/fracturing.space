package authz

import (
	"context"
	"errors"
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResolveCanCharacterOwnerParticipantID(t *testing.T) {
	t.Run("nil target skips ownership resolution", func(t *testing.T) {
		ownerParticipantID, evaluateOwnership, err := ResolveCanCharacterOwnerParticipantIDWithCharacterStore(
			context.Background(),
			nil,
			"camp-1",
			nil,
		)
		if err != nil {
			t.Fatalf("resolve owner: %v", err)
		}
		if evaluateOwnership {
			t.Fatalf("evaluate ownership = true, want false")
		}
		if ownerParticipantID != "" {
			t.Fatalf("owner participant id = %q, want empty", ownerParticipantID)
		}
	})

	t.Run("owner participant id in target bypasses store lookup", func(t *testing.T) {
		ownerParticipantID, evaluateOwnership, err := ResolveCanCharacterOwnerParticipantIDWithCharacterStore(
			context.Background(),
			nil,
			"camp-1",
			&campaignv1.AuthorizationTarget{OwnerParticipantId: "owner-1"},
		)
		if err != nil {
			t.Fatalf("resolve owner: %v", err)
		}
		if !evaluateOwnership {
			t.Fatalf("evaluate ownership = false, want true")
		}
		if ownerParticipantID != "owner-1" {
			t.Fatalf("owner participant id = %q, want %q", ownerParticipantID, "owner-1")
		}
	})

	t.Run("resource id resolves owner from character projection", func(t *testing.T) {
		characters := gametest.NewFakeCharacterStore()
		if err := characters.PutCharacter(context.Background(), storage.CharacterRecord{
			ID:                 "char-1",
			CampaignID:         "camp-1",
			OwnerParticipantID: "owner-2",
		}); err != nil {
			t.Fatalf("put character: %v", err)
		}

		ownerParticipantID, evaluateOwnership, err := ResolveCanCharacterOwnerParticipantIDWithCharacterStore(
			context.Background(),
			characters,
			"camp-1",
			&campaignv1.AuthorizationTarget{ResourceId: "char-1"},
		)
		if err != nil {
			t.Fatalf("resolve owner: %v", err)
		}
		if !evaluateOwnership {
			t.Fatalf("evaluate ownership = false, want true")
		}
		if ownerParticipantID != "owner-2" {
			t.Fatalf("owner participant id = %q, want %q", ownerParticipantID, "owner-2")
		}
	})
}

func TestEvaluateCanParticipantGovernanceTarget(t *testing.T) {
	t.Run("loads target access from participant store when target access missing", func(t *testing.T) {
		participants := gametest.NewFakeParticipantStore()
		if err := participants.PutParticipant(context.Background(), storage.ParticipantRecord{
			ID:             "owner-1",
			CampaignID:     "camp-1",
			CampaignAccess: participant.CampaignAccessOwner,
		}); err != nil {
			t.Fatalf("put participant: %v", err)
		}

		decision, attrs, evaluated, err := EvaluateCanParticipantGovernanceTargetWithStores(
			context.Background(),
			participants,
			nil,
			"camp-1",
			storage.ParticipantRecord{
				ID:             "manager-1",
				CampaignID:     "camp-1",
				CampaignAccess: participant.CampaignAccessManager,
			},
			&campaignv1.AuthorizationTarget{TargetParticipantId: "owner-1"},
		)
		if err != nil {
			t.Fatalf("evaluate target: %v", err)
		}
		if !evaluated {
			t.Fatalf("evaluated = false, want true")
		}
		if decision.Allowed {
			t.Fatalf("decision allowed = true, want false")
		}
		if decision.ReasonCode != domainauthz.ReasonDenyTargetIsOwner {
			t.Fatalf("decision reason = %q, want %q", decision.ReasonCode, domainauthz.ReasonDenyTargetIsOwner)
		}
		if got, ok := attrs["target_campaign_access"].(string); !ok || got != string(participant.CampaignAccessOwner) {
			t.Fatalf("target_campaign_access = %#v, want %q", attrs["target_campaign_access"], participant.CampaignAccessOwner)
		}
	})

	t.Run("access change operation requires requested access", func(t *testing.T) {
		decision, attrs, evaluated, err := EvaluateCanParticipantGovernanceTargetWithStores(
			context.Background(),
			nil,
			nil,
			"camp-1",
			storage.ParticipantRecord{
				ID:             "owner-1",
				CampaignID:     "camp-1",
				CampaignAccess: participant.CampaignAccessOwner,
			},
			&campaignv1.AuthorizationTarget{
				TargetParticipantId:  "member-1",
				TargetCampaignAccess: campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
				ParticipantOperation: campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_ACCESS_CHANGE,
			},
		)
		if err == nil {
			t.Fatal("expected error when requested campaign access is missing")
		}
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("error code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
		if evaluated {
			t.Fatalf("evaluated = true, want false")
		}
		if decision != (domainauthz.PolicyDecision{}) {
			t.Fatalf("decision = %#v, want zero value", decision)
		}
		if got, ok := attrs["participant_operation"].(string); !ok || got != "access_change" {
			t.Fatalf("participant_operation = %#v, want %q", attrs["participant_operation"], "access_change")
		}
	})

	t.Run("remove operation denies participants who still own active characters", func(t *testing.T) {
		participants := gametest.NewFakeParticipantStore()
		if err := participants.PutParticipant(context.Background(), storage.ParticipantRecord{
			ID:             "owner-1",
			CampaignID:     "camp-1",
			CampaignAccess: participant.CampaignAccessOwner,
		}); err != nil {
			t.Fatalf("put participant owner-1: %v", err)
		}
		if err := participants.PutParticipant(context.Background(), storage.ParticipantRecord{
			ID:             "owner-2",
			CampaignID:     "camp-1",
			CampaignAccess: participant.CampaignAccessOwner,
		}); err != nil {
			t.Fatalf("put participant owner-2: %v", err)
		}
		if err := participants.PutParticipant(context.Background(), storage.ParticipantRecord{
			ID:             "member-1",
			CampaignID:     "camp-1",
			CampaignAccess: participant.CampaignAccessMember,
		}); err != nil {
			t.Fatalf("put participant member-1: %v", err)
		}

		characters := gametest.NewFakeCharacterStore()
		if err := characters.PutCharacter(context.Background(), storage.CharacterRecord{
			ID:                 "char-1",
			CampaignID:         "camp-1",
			OwnerParticipantID: "member-1",
		}); err != nil {
			t.Fatalf("put character: %v", err)
		}

		decision, attrs, evaluated, err := EvaluateCanParticipantGovernanceTargetWithStores(
			context.Background(),
			participants,
			characters,
			"camp-1",
			storage.ParticipantRecord{
				ID:             "owner-1",
				CampaignID:     "camp-1",
				CampaignAccess: participant.CampaignAccessOwner,
			},
			&campaignv1.AuthorizationTarget{
				TargetParticipantId:  "member-1",
				TargetCampaignAccess: campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
				ParticipantOperation: campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_REMOVE,
			},
		)
		if err != nil {
			t.Fatalf("evaluate target: %v", err)
		}
		if !evaluated {
			t.Fatalf("evaluated = false, want true")
		}
		if decision.Allowed {
			t.Fatalf("decision allowed = true, want false")
		}
		if decision.ReasonCode != domainauthz.ReasonDenyTargetOwnsActiveCharacters {
			t.Fatalf("decision reason = %q, want %q", decision.ReasonCode, domainauthz.ReasonDenyTargetOwnsActiveCharacters)
		}
		if got, ok := attrs["target_owns_active_characters"].(bool); !ok || !got {
			t.Fatalf("target_owns_active_characters = %#v, want true", attrs["target_owns_active_characters"])
		}
	})

	t.Run("remove operation denies AI participants", func(t *testing.T) {
		participants := gametest.NewFakeParticipantStore()
		if err := participants.PutParticipant(context.Background(), storage.ParticipantRecord{
			ID:             "owner-1",
			CampaignID:     "camp-1",
			CampaignAccess: participant.CampaignAccessOwner,
			Controller:     participant.ControllerHuman,
		}); err != nil {
			t.Fatalf("put participant owner-1: %v", err)
		}
		if err := participants.PutParticipant(context.Background(), storage.ParticipantRecord{
			ID:             "owner-2",
			CampaignID:     "camp-1",
			CampaignAccess: participant.CampaignAccessOwner,
			Controller:     participant.ControllerHuman,
		}); err != nil {
			t.Fatalf("put participant owner-2: %v", err)
		}
		if err := participants.PutParticipant(context.Background(), storage.ParticipantRecord{
			ID:             "ai-1",
			CampaignID:     "camp-1",
			CampaignAccess: participant.CampaignAccessMember,
			Controller:     participant.ControllerAI,
		}); err != nil {
			t.Fatalf("put participant ai-1: %v", err)
		}

		decision, _, evaluated, err := EvaluateCanParticipantGovernanceTargetWithStores(
			context.Background(),
			participants,
			gametest.NewFakeCharacterStore(),
			"camp-1",
			storage.ParticipantRecord{
				ID:             "owner-1",
				CampaignID:     "camp-1",
				CampaignAccess: participant.CampaignAccessOwner,
			},
			&campaignv1.AuthorizationTarget{
				TargetParticipantId:  "ai-1",
				TargetCampaignAccess: campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
				ParticipantOperation: campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_REMOVE,
			},
		)
		if err != nil {
			t.Fatalf("evaluate target: %v", err)
		}
		if !evaluated {
			t.Fatalf("evaluated = false, want true")
		}
		if decision.Allowed {
			t.Fatalf("decision allowed = true, want false")
		}
		if decision.ReasonCode != domainauthz.ReasonDenyTargetIsAIParticipant {
			t.Fatalf("decision reason = %q, want %q", decision.ReasonCode, domainauthz.ReasonDenyTargetIsAIParticipant)
		}
	})

	t.Run("returns internal error when participant store lookup fails", func(t *testing.T) {
		participants := gametest.NewFakeParticipantStore()
		participants.GetErr = errors.New("boom")

		decision, attrs, evaluated, err := EvaluateCanParticipantGovernanceTargetWithStores(
			context.Background(),
			participants,
			nil,
			"camp-1",
			storage.ParticipantRecord{
				ID:             "owner-1",
				CampaignID:     "camp-1",
				CampaignAccess: participant.CampaignAccessOwner,
			},
			&campaignv1.AuthorizationTarget{TargetParticipantId: "member-1"},
		)
		if err == nil {
			t.Fatal("expected participant lookup error")
		}
		if status.Code(err) != codes.Internal {
			t.Fatalf("error code = %v, want %v", status.Code(err), codes.Internal)
		}
		if evaluated {
			t.Fatalf("evaluated = true, want false")
		}
		if decision != (domainauthz.PolicyDecision{}) {
			t.Fatalf("decision = %#v, want zero value", decision)
		}
		if attrs != nil {
			t.Fatalf("attrs = %#v, want nil", attrs)
		}
	})
}
