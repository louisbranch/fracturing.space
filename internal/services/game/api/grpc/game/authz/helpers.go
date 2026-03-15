package authz

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"google.golang.org/grpc/metadata"
)

// AdminOverrideFromContext checks whether the request carries an admin
// platform-role override header and returns the stated reason.
func AdminOverrideFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	role := strings.ToUpper(strings.TrimSpace(grpcmeta.FirstMetadataValue(md, PlatformRoleHeader)))
	if role != PlatformRoleAdmin {
		return "", false
	}
	reason := strings.TrimSpace(grpcmeta.FirstMetadataValue(md, OverrideReasonHeader))
	return reason, true
}

// DecisionForReason maps a reason code to the high-level decision label.
func DecisionForReason(reasonCode string) string {
	if reasonCode == ReasonAllowAdminOverride {
		return DecisionOverride
	}
	return DecisionAllow
}

// ExtraAttributesForReason returns override-specific audit attributes when the
// reason code indicates an admin override.
func ExtraAttributesForReason(ctx context.Context, reasonCode string) map[string]any {
	if reasonCode != ReasonAllowAdminOverride {
		return nil
	}
	reason, requested := AdminOverrideFromContext(ctx)
	if !requested || reason == "" {
		return nil
	}
	overrideUserID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	return map[string]any{
		"override_reason":          reason,
		"override_principal_user":  overrideUserID,
		"override_principal_scope": "user_id",
	}
}

// MergeAttributes merges multiple attribute maps into one.
func MergeAttributes(attributes ...map[string]any) map[string]any {
	var merged map[string]any
	for _, attrs := range attributes {
		if len(attrs) == 0 {
			continue
		}
		if merged == nil {
			merged = make(map[string]any, len(attrs))
		}
		for key, value := range attrs {
			merged[key] = value
		}
	}
	return merged
}

// PolicyCapabilityLabel returns a human-readable label for a capability.
func PolicyCapabilityLabel(capability domainauthz.Capability) string {
	switch capability {
	case domainauthz.CapabilityManageParticipants:
		return "manage_participants"
	case domainauthz.CapabilityManageInvites:
		return "manage_invites"
	case domainauthz.CapabilityManageSessions:
		return "manage_sessions"
	case domainauthz.CapabilityMutateCharacters:
		return "manage_characters"
	case domainauthz.CapabilityManageCampaign:
		return "manage_campaign"
	case domainauthz.CapabilityReadCampaign:
		return "read_campaign"
	default:
		return capability.Label()
	}
}
