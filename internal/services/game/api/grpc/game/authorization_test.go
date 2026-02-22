package game

import (
	"context"
	"errors"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type authzParticipantStore struct {
	get            func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error)
	listByCampaign func(ctx context.Context, campaignID string) ([]storage.ParticipantRecord, error)
}

type authzTelemetryStore struct {
	events []storage.TelemetryEvent
	err    error
}

func (s *authzTelemetryStore) AppendTelemetryEvent(_ context.Context, evt storage.TelemetryEvent) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, evt)
	return nil
}

func (f authzParticipantStore) PutParticipant(ctx context.Context, p storage.ParticipantRecord) error {
	return nil
}

func (f authzParticipantStore) GetParticipant(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
	if f.get == nil {
		return storage.ParticipantRecord{}, errors.New("missing handler")
	}
	return f.get(ctx, campaignID, participantID)
}

func (f authzParticipantStore) DeleteParticipant(ctx context.Context, campaignID, participantID string) error {
	return nil
}

func (f authzParticipantStore) ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]storage.ParticipantRecord, error) {
	if f.listByCampaign == nil {
		return nil, nil
	}
	return f.listByCampaign(ctx, campaignID)
}

func (f authzParticipantStore) ListCampaignIDsByUser(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}

func (f authzParticipantStore) ListCampaignIDsByParticipant(ctx context.Context, participantID string) ([]string, error) {
	return nil, nil
}

func (f authzParticipantStore) CountParticipants(ctx context.Context, campaignID string) (int, error) {
	return 0, nil
}

func (f authzParticipantStore) ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	return storage.ParticipantPage{}, nil
}

func TestRequirePolicyMissingActor(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{}}
	err := requirePolicy(context.Background(), stores, policyActionManageParticipants, storage.CampaignRecord{ID: "camp"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyNotFound(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{}, storage.ErrNotFound
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policyActionManageParticipants, storage.CampaignRecord{ID: "camp"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyLoadError(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{}, errors.New("boom")
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policyActionManageParticipants, storage.CampaignRecord{ID: "camp"})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error, got %v", err)
	}
}

func TestRequirePolicyDenied(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{CampaignAccess: participant.CampaignAccessMember}, nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policyActionManageParticipants, storage.CampaignRecord{ID: "camp"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyAllowed(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{CampaignAccess: participant.CampaignAccessOwner}, nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policyActionManageParticipants, storage.CampaignRecord{ID: "camp"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicyCampaignManageDeniedForManager(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{CampaignAccess: participant.CampaignAccessManager}, nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policyActionManageCampaign, storage.CampaignRecord{ID: "camp"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyCampaignManageAllowedForOwner(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{CampaignAccess: participant.CampaignAccessOwner}, nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policyActionManageCampaign, storage.CampaignRecord{ID: "camp"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicySessionManageAllowedForManager(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{CampaignAccess: participant.CampaignAccessManager}, nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policyActionManageSessions, storage.CampaignRecord{ID: "camp"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicySessionManageDeniedForMember(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{CampaignAccess: participant.CampaignAccessMember}, nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policyActionManageSessions, storage.CampaignRecord{ID: "camp"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyCharacterManageAllowedForMember(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{CampaignAccess: participant.CampaignAccessMember}, nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policyActionManageCharacters, storage.CampaignRecord{ID: "camp"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicyAllowsOwnerByUserIDFallback(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{
		get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
			return storage.ParticipantRecord{}, storage.ErrNotFound
		},
		listByCampaign: func(ctx context.Context, campaignID string) ([]storage.ParticipantRecord, error) {
			return []storage.ParticipantRecord{
				{
					ID:             "owner-1",
					UserID:         "user-1",
					CampaignAccess: participant.CampaignAccessOwner,
				},
			}, nil
		},
	}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, "user-1"))

	err := requirePolicy(ctx, stores, policyActionManageCampaign, storage.CampaignRecord{ID: "camp"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicyTelemetryDenied(t *testing.T) {
	telemetryStore := &authzTelemetryStore{}
	stores := Stores{
		Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
			return storage.ParticipantRecord{
				ID:             "member-1",
				CampaignID:     campaignID,
				CampaignAccess: participant.CampaignAccessMember,
			}, nil
		}},
		Telemetry: telemetryStore,
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "member-1"))

	err := requirePolicy(ctx, stores, policyActionManageParticipants, storage.CampaignRecord{ID: "camp"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if len(telemetryStore.events) != 1 {
		t.Fatalf("telemetry events = %d, want 1", len(telemetryStore.events))
	}
	evt := telemetryStore.events[0]
	if evt.EventName != "telemetry.authz.decision" {
		t.Fatalf("event name = %q, want %q", evt.EventName, "telemetry.authz.decision")
	}
	if got, ok := evt.Attributes["decision"].(string); !ok || got != "deny" {
		t.Fatalf("decision = %#v, want %q", evt.Attributes["decision"], "deny")
	}
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED")
	}
}

func TestRequirePolicyTelemetryAllowed(t *testing.T) {
	telemetryStore := &authzTelemetryStore{}
	stores := Stores{
		Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
			return storage.ParticipantRecord{
				ID:             "owner-1",
				CampaignID:     campaignID,
				CampaignAccess: participant.CampaignAccessOwner,
			}, nil
		}},
		Telemetry: telemetryStore,
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "owner-1"))

	if err := requirePolicy(ctx, stores, policyActionManageCampaign, storage.CampaignRecord{ID: "camp"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(telemetryStore.events) != 1 {
		t.Fatalf("telemetry events = %d, want 1", len(telemetryStore.events))
	}
	evt := telemetryStore.events[0]
	if got, ok := evt.Attributes["decision"].(string); !ok || got != "allow" {
		t.Fatalf("decision = %#v, want %q", evt.Attributes["decision"], "allow")
	}
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_ALLOW_ACCESS_LEVEL" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_ALLOW_ACCESS_LEVEL")
	}
}

func TestRequireCharacterMutationPolicyTelemetryDeniedNotOwner(t *testing.T) {
	telemetryStore := &authzTelemetryStore{}
	characterStore := newFakeCharacterStore()
	if err := characterStore.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:            "char-1",
		CampaignID:    "camp",
		ParticipantID: "member-owner",
		Name:          "Hero",
	}); err != nil {
		t.Fatalf("put character: %v", err)
	}

	stores := Stores{
		Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
			return storage.ParticipantRecord{
				ID:             "member-1",
				CampaignID:     campaignID,
				CampaignAccess: participant.CampaignAccessMember,
			}, nil
		}},
		Character: characterStore,
		Telemetry: telemetryStore,
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "member-1"))

	_, err := requireCharacterMutationPolicy(
		ctx,
		stores,
		storage.CampaignRecord{ID: "camp"},
		"char-1",
	)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if len(telemetryStore.events) != 1 {
		t.Fatalf("telemetry events = %d, want 1", len(telemetryStore.events))
	}
	evt := telemetryStore.events[0]
	if got, ok := evt.Attributes["decision"].(string); !ok || got != "deny" {
		t.Fatalf("decision = %#v, want %q", evt.Attributes["decision"], "deny")
	}
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_DENY_NOT_RESOURCE_OWNER" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_DENY_NOT_RESOURCE_OWNER")
	}
}

func TestRequirePolicyTelemetryAdminOverride(t *testing.T) {
	telemetryStore := &authzTelemetryStore{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
		"x-fracturing-space-authz-override-reason", "incident-ops",
		grpcmeta.UserIDHeader, "user-admin-1",
	))

	err := requirePolicy(ctx, Stores{Telemetry: telemetryStore}, policyActionManageCampaign, storage.CampaignRecord{ID: "camp"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(telemetryStore.events) != 1 {
		t.Fatalf("telemetry events = %d, want 1", len(telemetryStore.events))
	}
	evt := telemetryStore.events[0]
	if got, ok := evt.Attributes["decision"].(string); !ok || got != "override" {
		t.Fatalf("decision = %#v, want %q", evt.Attributes["decision"], "override")
	}
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_ALLOW_ADMIN_OVERRIDE" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_ALLOW_ADMIN_OVERRIDE")
	}
	if got, ok := evt.Attributes["override_reason"].(string); !ok || got != "incident-ops" {
		t.Fatalf("override_reason = %#v, want %q", evt.Attributes["override_reason"], "incident-ops")
	}
}

func TestRequirePolicyDeniesAdminOverrideWhenReasonMissing(t *testing.T) {
	telemetryStore := &authzTelemetryStore{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
	))

	err := requirePolicy(ctx, Stores{Telemetry: telemetryStore}, policyActionManageCampaign, storage.CampaignRecord{ID: "camp"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if len(telemetryStore.events) != 1 {
		t.Fatalf("telemetry events = %d, want 1", len(telemetryStore.events))
	}
	evt := telemetryStore.events[0]
	if got, ok := evt.Attributes["decision"].(string); !ok || got != "deny" {
		t.Fatalf("decision = %#v, want %q", evt.Attributes["decision"], "deny")
	}
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_DENY_OVERRIDE_REASON_REQUIRED" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_DENY_OVERRIDE_REASON_REQUIRED")
	}
}

func TestRequireCharacterMutationPolicyTelemetryAdminOverride(t *testing.T) {
	telemetryStore := &authzTelemetryStore{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
		"x-fracturing-space-authz-override-reason", "moderation",
		grpcmeta.UserIDHeader, "user-admin-1",
	))

	_, err := requireCharacterMutationPolicy(
		ctx,
		Stores{Telemetry: telemetryStore},
		storage.CampaignRecord{ID: "camp"},
		"char-1",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(telemetryStore.events) != 1 {
		t.Fatalf("telemetry events = %d, want 1", len(telemetryStore.events))
	}
	evt := telemetryStore.events[0]
	if got, ok := evt.Attributes["decision"].(string); !ok || got != "override" {
		t.Fatalf("decision = %#v, want %q", evt.Attributes["decision"], "override")
	}
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_ALLOW_ADMIN_OVERRIDE" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_ALLOW_ADMIN_OVERRIDE")
	}
	if got, ok := evt.Attributes["override_reason"].(string); !ok || got != "moderation" {
		t.Fatalf("override_reason = %#v, want %q", evt.Attributes["override_reason"], "moderation")
	}
	if got, ok := evt.Attributes["character_id"].(string); !ok || got != "char-1" {
		t.Fatalf("character_id = %#v, want %q", evt.Attributes["character_id"], "char-1")
	}
}

func TestAdminOverrideFromContext(t *testing.T) {
	if reason, ok := adminOverrideFromContext(nil); ok || reason != "" {
		t.Fatalf("nil context override = (%q, %v), want empty/false", reason, ok)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "MANAGER",
		"x-fracturing-space-authz-override-reason", "incident",
	))
	if reason, ok := adminOverrideFromContext(ctx); ok || reason != "" {
		t.Fatalf("non-admin override = (%q, %v), want empty/false", reason, ok)
	}

	adminCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
		"x-fracturing-space-authz-override-reason", "incident",
	))
	if reason, ok := adminOverrideFromContext(adminCtx); !ok || reason != "incident" {
		t.Fatalf("admin override = (%q, %v), want incident/true", reason, ok)
	}
}

func TestAuthzExtraAttributesForReason(t *testing.T) {
	attrs := authzExtraAttributesForReason(context.Background(), authzReasonAllowAccessLevel)
	if attrs != nil {
		t.Fatalf("attrs = %#v, want nil for non-override reason", attrs)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
	))
	attrs = authzExtraAttributesForReason(ctx, authzReasonAllowAdminOverride)
	if attrs != nil {
		t.Fatalf("attrs = %#v, want nil for missing override reason", attrs)
	}
}

func TestCanPerformPolicyActionUnknown(t *testing.T) {
	if canPerformPolicyAction(policyAction(0), participant.CampaignAccessOwner) {
		t.Fatal("expected unknown policy action to be denied")
	}
}

func TestPolicyActionLabelUnknown(t *testing.T) {
	if label := policyActionLabel(policyAction(0)); label != "unknown" {
		t.Fatalf("label = %q, want unknown", label)
	}
}
