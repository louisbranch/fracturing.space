package app

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type fakeAuthzGateway struct {
	decision AuthorizationDecision
	err      error
}

func (f fakeAuthzGateway) CanCampaignAction(_ context.Context, _ string, _ AuthorizationAction, _ AuthorizationResource, _ *AuthorizationTarget) (AuthorizationDecision, error) {
	return f.decision, f.err
}

func TestRequirePolicyAllowedWhenEvaluatedAndAllowed(t *testing.T) {
	t.Parallel()

	support := authorizationSupport{
		gateway: fakeAuthzGateway{
			decision: AuthorizationDecision{Evaluated: true, Allowed: true},
		},
	}
	for _, p := range []mutationAuthzPolicy{
		policyManageCampaign,
		policyManageSession,
		policyMutateCharacter,
		policyManageCharacter,
		policyManageInvite,
		policyManageParticipant,
	} {
		if err := support.requirePolicy(context.Background(), "campaign-1", p); err != nil {
			t.Errorf("policy %s/%s: unexpected error: %v", p.action, p.resource, err)
		}
	}
}

func TestRequirePolicyDeniedWhenNotAllowed(t *testing.T) {
	t.Parallel()

	support := authorizationSupport{
		gateway: fakeAuthzGateway{
			decision: AuthorizationDecision{Evaluated: true, Allowed: false},
		},
	}
	err := support.requirePolicy(context.Background(), "campaign-1", policyManageCampaign)
	if err == nil {
		t.Fatal("expected error for denied decision")
	}
	var appErr apperrors.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T, want apperrors.Error", err)
	}
	if appErr.Kind != apperrors.KindForbidden {
		t.Fatalf("error kind = %s, want %s", appErr.Kind, apperrors.KindForbidden)
	}
}

func TestRequirePolicyDeniedWhenNotEvaluated(t *testing.T) {
	t.Parallel()

	support := authorizationSupport{
		gateway: fakeAuthzGateway{
			decision: AuthorizationDecision{Evaluated: false, Allowed: true},
		},
	}
	err := support.requirePolicy(context.Background(), "campaign-1", policyManageSession)
	if err == nil {
		t.Fatal("expected error for unevaluated decision")
	}
}

func TestRequirePolicyDeniedOnGatewayError(t *testing.T) {
	t.Parallel()

	support := authorizationSupport{
		gateway: fakeAuthzGateway{
			err: errors.New("rpc unavailable"),
		},
	}
	err := support.requirePolicy(context.Background(), "campaign-1", policyManageInvite)
	if err == nil {
		t.Fatal("expected error for gateway failure")
	}
	var appErr apperrors.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T, want apperrors.Error", err)
	}
	if appErr.Kind != apperrors.KindForbidden {
		t.Fatalf("error kind = %s, want %s", appErr.Kind, apperrors.KindForbidden)
	}
}

func TestRequirePolicyDeniedWithNilGateway(t *testing.T) {
	t.Parallel()

	support := authorizationSupport{gateway: nil}
	err := support.requirePolicy(context.Background(), "campaign-1", policyManageCampaign)
	if err == nil {
		t.Fatal("expected error for nil gateway")
	}
	var appErr apperrors.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T, want apperrors.Error", err)
	}
	if appErr.Kind != apperrors.KindForbidden {
		t.Fatalf("error kind = %s, want %s", appErr.Kind, apperrors.KindForbidden)
	}
}

func TestRequirePolicyRejectsEmptyCampaignID(t *testing.T) {
	t.Parallel()

	support := authorizationSupport{
		gateway: fakeAuthzGateway{
			decision: AuthorizationDecision{Evaluated: true, Allowed: true},
		},
	}
	for _, id := range []string{"", "  ", "	"} {
		if err := support.requirePolicy(context.Background(), id, policyManageCampaign); err == nil {
			t.Errorf("expected error for campaign id %q", id)
		}
	}
}

func TestRequirePolicyWithTargetPassesResourceID(t *testing.T) {
	t.Parallel()

	var capturedTarget *AuthorizationTarget
	gateway := gatewayCapture{
		decision: AuthorizationDecision{Evaluated: true, Allowed: true},
		capture:  func(target *AuthorizationTarget) { capturedTarget = target },
	}
	support := authorizationSupport{gateway: gateway}
	err := support.requirePolicyWithTarget(context.Background(), "campaign-1", policyManageCharacter, "char-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTarget == nil || capturedTarget.ResourceID != "char-42" {
		t.Fatalf("target = %#v, want ResourceID=char-42", capturedTarget)
	}
}

func TestPolicyDenyKeysAreNonEmpty(t *testing.T) {
	t.Parallel()

	// Invariant: every policy must have a non-empty deny key for i18n error messages.
	for _, p := range []mutationAuthzPolicy{
		policyManageCampaign,
		policyManageSession,
		policyMutateCharacter,
		policyManageCharacter,
		policyManageInvite,
		policyManageParticipant,
	} {
		if p.denyKey == "" {
			t.Errorf("policy %s/%s has empty denyKey", p.action, p.resource)
		}
		if p.denyMsg == "" {
			t.Errorf("policy %s/%s has empty denyMsg", p.action, p.resource)
		}
	}
}

type gatewayCapture struct {
	decision AuthorizationDecision
	capture  func(*AuthorizationTarget)
}

func (g gatewayCapture) CanCampaignAction(_ context.Context, _ string, _ AuthorizationAction, _ AuthorizationResource, target *AuthorizationTarget) (AuthorizationDecision, error) {
	if g.capture != nil {
		g.capture(target)
	}
	return g.decision, nil
}
