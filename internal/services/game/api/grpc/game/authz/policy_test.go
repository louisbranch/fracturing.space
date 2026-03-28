package authz

import (
	"context"
	"errors"
	"strings"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit/events"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testParticipantStore struct {
	get            func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error)
	listByCampaign func(ctx context.Context, campaignID string) ([]storage.ParticipantRecord, error)
}

type testAuditStore struct {
	events []storage.AuditEvent
	err    error
}

func testCampaignRecord() storage.CampaignRecord {
	return storage.CampaignRecord{}
}

func testParticipantWithAccess(access participant.CampaignAccess) storage.ParticipantRecord {
	return storage.ParticipantRecord{CampaignAccess: access}
}

func (s *testAuditStore) AppendAuditEvent(_ context.Context, evt storage.AuditEvent) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, evt)
	return nil
}

func (f testParticipantStore) PutParticipant(ctx context.Context, p storage.ParticipantRecord) error {
	return nil
}

func (f testParticipantStore) GetParticipant(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
	if f.get == nil {
		return storage.ParticipantRecord{}, errors.New("missing handler")
	}
	return f.get(ctx, campaignID, participantID)
}

func (f testParticipantStore) DeleteParticipant(ctx context.Context, campaignID, participantID string) error {
	return nil
}

func (f testParticipantStore) ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]storage.ParticipantRecord, error) {
	if f.listByCampaign == nil {
		return nil, nil
	}
	return f.listByCampaign(ctx, campaignID)
}

func (f testParticipantStore) ListCampaignIDsByUser(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}

func (f testParticipantStore) ListCampaignIDsByParticipant(ctx context.Context, participantID string) ([]string, error) {
	return nil, nil
}

func (f testParticipantStore) CountParticipants(ctx context.Context, campaignID string) (int, error) {
	return 0, nil
}

func (f testParticipantStore) ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	return storage.ParticipantPage{}, nil
}

// testCharacterStore is a minimal in-memory character store for authz tests.
type testCharacterStore struct {
	characters map[string]map[string]storage.CharacterRecord
}

func newTestCharacterStore() *testCharacterStore {
	return &testCharacterStore{characters: map[string]map[string]storage.CharacterRecord{}}
}

func (s *testCharacterStore) PutCharacter(_ context.Context, r storage.CharacterRecord) error {
	camp := s.characters[r.CampaignID]
	if camp == nil {
		camp = map[string]storage.CharacterRecord{}
		s.characters[r.CampaignID] = camp
	}
	camp[r.ID] = r
	return nil
}

func (s *testCharacterStore) GetCharacter(_ context.Context, campaignID, characterID string) (storage.CharacterRecord, error) {
	camp := s.characters[campaignID]
	if camp == nil {
		return storage.CharacterRecord{}, storage.ErrNotFound
	}
	r, ok := camp[characterID]
	if !ok {
		return storage.CharacterRecord{}, storage.ErrNotFound
	}
	return r, nil
}

func (s *testCharacterStore) DeleteCharacter(context.Context, string, string) error { return nil }

func (s *testCharacterStore) ListCharacters(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error) {
	camp := s.characters[campaignID]
	var chars []storage.CharacterRecord
	for _, c := range camp {
		chars = append(chars, c)
	}
	return storage.CharacterPage{Characters: chars}, nil
}

func (s *testCharacterStore) CountCharacters(_ context.Context, campaignID string) (int, error) {
	return len(s.characters[campaignID]), nil
}

func (s *testCharacterStore) ListCharactersByOwnerParticipant(_ context.Context, campaignID, participantID string) ([]storage.CharacterRecord, error) {
	var result []storage.CharacterRecord
	for _, c := range s.characters[campaignID] {
		if c.OwnerParticipantID == participantID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *testCharacterStore) ListCharactersByControllerParticipant(_ context.Context, campaignID, participantID string) ([]storage.CharacterRecord, error) {
	var result []storage.CharacterRecord
	for _, c := range s.characters[campaignID] {
		if c.ParticipantID == participantID {
			result = append(result, c)
		}
	}
	return result, nil
}

func TestRequirePolicyMissingActor(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{}}
	err := RequirePolicy(context.Background(), deps, domainauthz.CapabilityManageParticipants(), testCampaignRecord())
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyNotFound(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{}, storage.ErrNotFound
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageParticipants(), testCampaignRecord())
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyLoadError(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return storage.ParticipantRecord{}, errors.New("boom")
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageParticipants(), testCampaignRecord())
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error, got %v", err)
	}
}

func TestRequirePolicyDenied(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return testParticipantWithAccess(participant.CampaignAccessMember), nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageParticipants(), testCampaignRecord())
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyAllowed(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return testParticipantWithAccess(participant.CampaignAccessOwner), nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageParticipants(), testCampaignRecord())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicyCampaignManageAllowedForManager(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return testParticipantWithAccess(participant.CampaignAccessManager), nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageCampaign(), testCampaignRecord())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicyCampaignManageAllowedForOwner(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return testParticipantWithAccess(participant.CampaignAccessOwner), nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageCampaign(), testCampaignRecord())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicySessionManageAllowedForManager(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return testParticipantWithAccess(participant.CampaignAccessManager), nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageSessions(), testCampaignRecord())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicySessionManageDeniedForMember(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return testParticipantWithAccess(participant.CampaignAccessMember), nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageSessions(), testCampaignRecord())
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyCharacterManageAllowedForMember(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
		return testParticipantWithAccess(participant.CampaignAccessMember), nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityMutateCharacters(), testCampaignRecord())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicyAllowsOwnerByUserIDFallback(t *testing.T) {
	deps := PolicyDeps{Participant: testParticipantStore{
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

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageCampaign(), testCampaignRecord())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequirePolicyTelemetryDenied(t *testing.T) {
	auditStore := &testAuditStore{}
	deps := PolicyDeps{
		Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
			return storage.ParticipantRecord{
				ID:             "member-1",
				CampaignID:     campaignID,
				CampaignAccess: participant.CampaignAccessMember,
			}, nil
		}},
		Audit: auditStore,
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "member-1"))

	err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageParticipants(), testCampaignRecord())
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if len(auditStore.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditStore.events))
	}
	evt := auditStore.events[0]
	if evt.EventName != events.AuthzDecision {
		t.Fatalf("event name = %q, want %q", evt.EventName, events.AuthzDecision)
	}
	if got, ok := evt.Attributes["decision"].(string); !ok || got != "deny" {
		t.Fatalf("decision = %#v, want %q", evt.Attributes["decision"], "deny")
	}
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED")
	}
}

func TestRequirePolicyTelemetryAllowed(t *testing.T) {
	auditStore := &testAuditStore{}
	deps := PolicyDeps{
		Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
			return storage.ParticipantRecord{
				ID:             "owner-1",
				CampaignID:     campaignID,
				CampaignAccess: participant.CampaignAccessOwner,
			}, nil
		}},
		Audit: auditStore,
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "owner-1"))

	if err := RequirePolicy(ctx, deps, domainauthz.CapabilityManageCampaign(), testCampaignRecord()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(auditStore.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditStore.events))
	}
	evt := auditStore.events[0]
	if got, ok := evt.Attributes["decision"].(string); !ok || got != "allow" {
		t.Fatalf("decision = %#v, want %q", evt.Attributes["decision"], "allow")
	}
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_ALLOW_ACCESS_LEVEL" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_ALLOW_ACCESS_LEVEL")
	}
}

func TestRequireCharacterMutationPolicyTelemetryDeniedNotOwner(t *testing.T) {
	auditStore := &testAuditStore{}
	characterStore := newTestCharacterStore()
	if err := characterStore.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-1",
		CampaignID:         "camp",
		OwnerParticipantID: "member-owner",
		Name:               "Hero",
	}); err != nil {
		t.Fatalf("put character: %v", err)
	}

	deps := PolicyDeps{
		Participant: testParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
			return storage.ParticipantRecord{
				ID:             "member-1",
				CampaignID:     campaignID,
				CampaignAccess: participant.CampaignAccessMember,
			}, nil
		}},
		Character: characterStore,
		Audit:     auditStore,
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "member-1"))

	_, err := RequireCharacterMutationPolicy(
		ctx,
		deps,
		testCampaignRecord(),
		"char-1",
	)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if len(auditStore.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditStore.events))
	}
	evt := auditStore.events[0]
	if got, ok := evt.Attributes["decision"].(string); !ok || got != "deny" {
		t.Fatalf("decision = %#v, want %q", evt.Attributes["decision"], "deny")
	}
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_DENY_NOT_RESOURCE_OWNER" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_DENY_NOT_RESOURCE_OWNER")
	}
}

func TestRequirePolicyTelemetryAdminOverride(t *testing.T) {
	auditStore := &testAuditStore{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
		"x-fracturing-space-authz-override-reason", "incident-ops",
		grpcmeta.UserIDHeader, "user-admin-1",
	))

	err := RequirePolicy(ctx, PolicyDeps{Audit: auditStore}, domainauthz.CapabilityManageCampaign(), testCampaignRecord())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(auditStore.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditStore.events))
	}
	evt := auditStore.events[0]
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
	auditStore := &testAuditStore{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
		grpcmeta.UserIDHeader, "user-admin-1",
	))

	err := RequirePolicy(ctx, PolicyDeps{Audit: auditStore}, domainauthz.CapabilityManageCampaign(), testCampaignRecord())
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if len(auditStore.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditStore.events))
	}
	evt := auditStore.events[0]
	if got, ok := evt.Attributes["decision"].(string); !ok || got != "deny" {
		t.Fatalf("decision = %#v, want %q", evt.Attributes["decision"], "deny")
	}
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_DENY_OVERRIDE_REASON_REQUIRED" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_DENY_OVERRIDE_REASON_REQUIRED")
	}
}

func TestRequirePolicyDeniesAdminOverrideWhenPrincipalMissing(t *testing.T) {
	auditStore := &testAuditStore{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
		"x-fracturing-space-authz-override-reason", "incident-ops",
	))

	err := RequirePolicy(ctx, PolicyDeps{Audit: auditStore}, domainauthz.CapabilityManageCampaign(), testCampaignRecord())
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if len(auditStore.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditStore.events))
	}
	evt := auditStore.events[0]
	if got, ok := evt.Attributes["reason_code"].(string); !ok || got != "AUTHZ_DENY_MISSING_IDENTITY" {
		t.Fatalf("reason_code = %#v, want %q", evt.Attributes["reason_code"], "AUTHZ_DENY_MISSING_IDENTITY")
	}
}

func TestRequireCharacterMutationPolicyTelemetryAdminOverride(t *testing.T) {
	auditStore := &testAuditStore{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
		"x-fracturing-space-authz-override-reason", "moderation",
		grpcmeta.UserIDHeader, "user-admin-1",
	))

	_, err := RequireCharacterMutationPolicy(
		ctx,
		PolicyDeps{Audit: auditStore},
		testCampaignRecord(),
		"char-1",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(auditStore.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditStore.events))
	}
	evt := auditStore.events[0]
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
	if reason, ok := AdminOverrideFromContext(nil); ok || reason != "" {
		t.Fatalf("nil context override = (%q, %v), want empty/false", reason, ok)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "MANAGER",
		"x-fracturing-space-authz-override-reason", "incident",
	))
	if reason, ok := AdminOverrideFromContext(ctx); ok || reason != "" {
		t.Fatalf("non-admin override = (%q, %v), want empty/false", reason, ok)
	}

	adminCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
		"x-fracturing-space-authz-override-reason", "incident",
	))
	if reason, ok := AdminOverrideFromContext(adminCtx); !ok || reason != "incident" {
		t.Fatalf("admin override = (%q, %v), want incident/true", reason, ok)
	}
}

func TestExtraAttributesForReason(t *testing.T) {
	attrs := ExtraAttributesForReason(context.Background(), ReasonAllowAccessLevel)
	if attrs != nil {
		t.Fatalf("attrs = %#v, want nil for non-override reason", attrs)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-platform-role", "ADMIN",
	))
	attrs = ExtraAttributesForReason(ctx, ReasonAllowAdminOverride)
	if attrs != nil {
		t.Fatalf("attrs = %#v, want nil for missing override reason", attrs)
	}
}

func TestPolicyCapabilityLabelUnknown(t *testing.T) {
	if label := PolicyCapabilityLabel(domainauthz.Capability{}); label != "unknown" {
		t.Fatalf("label = %q, want unknown", label)
	}
}

func TestCountCampaignOwnersNilStore(t *testing.T) {
	_, err := CountCampaignOwners(context.Background(), nil, "c1")
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "participant store is not configured") {
		t.Fatalf("error = %v, want participant store configuration message", err)
	}
}
