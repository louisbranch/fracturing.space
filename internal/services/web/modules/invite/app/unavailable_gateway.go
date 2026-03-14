package app

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// unavailableGateway fails closed when the invite module lacks runtime wiring.
type unavailableGateway struct{}

// GetPublicInvite rejects reads when no backing invite transport is configured.
func (unavailableGateway) GetPublicInvite(context.Context, string) (PublicInvite, error) {
	return PublicInvite{}, apperrors.E(apperrors.KindUnavailable, "invite service client is not configured")
}

// AcceptInvite rejects claims when no backing invite transport is configured.
func (unavailableGateway) AcceptInvite(context.Context, string, PublicInvite) error {
	return apperrors.E(apperrors.KindUnavailable, "invite service client is not configured")
}

// DeclineInvite rejects declines when no backing invite transport is configured.
func (unavailableGateway) DeclineInvite(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "invite service client is not configured")
}
