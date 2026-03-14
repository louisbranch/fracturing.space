package gateway

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ListPasskeys returns passkey summary rows for the security page.
func (g GRPCGateway) ListPasskeys(ctx context.Context, userID string) ([]settingsapp.SettingsPasskey, error) {
	if g.PasskeyClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.auth_service_is_not_configured", "auth service client is not configured")
	}
	resp, err := g.PasskeyClient.ListPasskeys(ctx, &authv1.ListPasskeysRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	passkeys := make([]*authv1.PasskeyCredentialSummary, 0, len(resp.GetPasskeys()))
	passkeys = append(passkeys, resp.GetPasskeys()...)
	sort.Slice(passkeys, func(i, j int) bool {
		leftLastUsed, leftCreated := passkeySortKey(passkeys[i])
		rightLastUsed, rightCreated := passkeySortKey(passkeys[j])
		if !leftLastUsed.Equal(rightLastUsed) {
			return leftLastUsed.After(rightLastUsed)
		}
		return leftCreated.After(rightCreated)
	})
	rows := make([]settingsapp.SettingsPasskey, 0, len(passkeys))
	for idx, passkey := range passkeys {
		if passkey == nil {
			continue
		}
		rows = append(rows, settingsapp.SettingsPasskey{
			Number:     idx + 1,
			CreatedAt:  formatProtoTimestamp(passkey.GetCreatedAt()),
			LastUsedAt: formatProtoTimestamp(passkey.GetLastUsedAt()),
		})
	}
	return rows, nil
}

// BeginPasskeyRegistration starts authenticated passkey enrollment for one user.
func (g GRPCGateway) BeginPasskeyRegistration(ctx context.Context, userID string) (settingsapp.PasskeyChallenge, error) {
	if g.PasskeyClient == nil {
		return settingsapp.PasskeyChallenge{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.auth_service_is_not_configured", "auth service client is not configured")
	}
	resp, err := g.PasskeyClient.BeginPasskeyRegistration(ctx, &authv1.BeginPasskeyRegistrationRequest{UserId: userID})
	if err != nil {
		return settingsapp.PasskeyChallenge{}, err
	}
	return settingsapp.PasskeyChallenge{
		SessionID: strings.TrimSpace(resp.GetSessionId()),
		PublicKey: json.RawMessage(resp.GetCredentialCreationOptionsJson()),
	}, nil
}

// FinishPasskeyRegistration completes authenticated passkey enrollment.
func (g GRPCGateway) FinishPasskeyRegistration(ctx context.Context, sessionID string, credential json.RawMessage) error {
	if g.PasskeyClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.auth_service_is_not_configured", "auth service client is not configured")
	}
	_, err := g.PasskeyClient.FinishPasskeyRegistration(ctx, &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              sessionID,
		CredentialResponseJson: credential,
	})
	return err
}

// passkeySortKey keeps security-page ordering policy in one place for stable gateway tests.
func passkeySortKey(value *authv1.PasskeyCredentialSummary) (lastUsed time.Time, created time.Time) {
	if value == nil {
		return time.Time{}, time.Time{}
	}
	return timestampValue(value.GetLastUsedAt()), timestampValue(value.GetCreatedAt())
}

// timestampValue normalizes nil protobuf timestamps into zero time before sorting.
func timestampValue(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime()
}
