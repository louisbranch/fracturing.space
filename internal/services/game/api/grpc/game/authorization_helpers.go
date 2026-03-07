package game

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"google.golang.org/grpc/metadata"
)

func adminOverrideFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	role := strings.ToUpper(strings.TrimSpace(grpcmeta.FirstMetadataValue(md, authzPlatformRoleHeader)))
	if role != authzPlatformRoleAdmin {
		return "", false
	}
	reason := strings.TrimSpace(grpcmeta.FirstMetadataValue(md, authzOverrideReasonHeader))
	return reason, true
}

func authzDecisionForReason(reasonCode string) string {
	if reasonCode == authzReasonAllowAdminOverride {
		return authzDecisionOverride
	}
	return authzDecisionAllow
}

func authzExtraAttributesForReason(ctx context.Context, reasonCode string) map[string]any {
	if reasonCode != authzReasonAllowAdminOverride {
		return nil
	}
	reason, requested := adminOverrideFromContext(ctx)
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

func mergeAuthzAttributes(attributes ...map[string]any) map[string]any {
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

func policyCapabilityLabel(capability domainauthz.Capability) string {
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
