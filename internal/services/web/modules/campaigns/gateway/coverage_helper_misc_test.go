package gateway

import (
	"context"
	"errors"
	"net/http"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCampaignNameReturnsTrimmedNameAndEmptyWhenMissing(t *testing.T) {
	t.Parallel()

	gateway := workspaceReadGateway{
		read: WorkspaceReadDeps{
			Campaign: &contractCampaignClient{
				getResp: &statev1.GetCampaignResponse{Campaign: &statev1.Campaign{Name: "  Lantern March  "}},
			},
		},
	}
	name, err := gateway.CampaignName(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("CampaignName() error = %v", err)
	}
	if name != "Lantern March" {
		t.Fatalf("CampaignName() = %q, want %q", name, "Lantern March")
	}

	gateway.read.Campaign = &contractCampaignClient{getResp: &statev1.GetCampaignResponse{}}
	name, err = gateway.CampaignName(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("CampaignName() missing campaign error = %v", err)
	}
	if name != "" {
		t.Fatalf("CampaignName() missing campaign = %q, want empty", name)
	}
}

func TestHeritageAssetTypeAndStartingWeaponIDsHelperBranches(t *testing.T) {
	t.Parallel()

	if got := heritageAssetType("ancestry"); got != daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ANCESTRY_ILLUSTRATION {
		t.Fatalf("heritageAssetType(ancestry) = %v", got)
	}
	if got := heritageAssetType("community"); got != daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_COMMUNITY_ILLUSTRATION {
		t.Fatalf("heritageAssetType(community) = %v", got)
	}
	if got := heritageAssetType("unknown"); got != daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_UNSPECIFIED {
		t.Fatalf("heritageAssetType(unknown) = %v", got)
	}

	if first, second := startingWeaponIDs(nil); first != "" || second != "" {
		t.Fatalf("startingWeaponIDs(nil) = (%q,%q), want empty", first, second)
	}
	if first, second := startingWeaponIDs(&daggerheartv1.DaggerheartProfile{StartingWeaponIds: []string{"  blade-1  "}}); first != "blade-1" || second != "" {
		t.Fatalf("startingWeaponIDs(one) = (%q,%q)", first, second)
	}
	if first, second := startingWeaponIDs(&daggerheartv1.DaggerheartProfile{StartingWeaponIds: []string{" blade-1 ", " ", " blade-2 "}}); first != "blade-1" || second != "blade-2" {
		t.Fatalf("startingWeaponIDs(two) = (%q,%q)", first, second)
	}
}

func TestMapSessionMutationErrorMapsStatusFamilies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "nil", err: nil, status: http.StatusOK},
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "bad"), status: http.StatusBadRequest},
		{name: "conflict", err: status.Error(codes.FailedPrecondition, "busy"), status: http.StatusConflict},
		{name: "unauthenticated", err: status.Error(codes.Unauthenticated, "login"), status: http.StatusUnauthorized},
		{name: "forbidden", err: status.Error(codes.PermissionDenied, "denied"), status: http.StatusForbidden},
		{name: "not found", err: status.Error(codes.NotFound, "missing"), status: http.StatusNotFound},
		{name: "unavailable", err: status.Error(codes.ResourceExhausted, "later"), status: http.StatusServiceUnavailable},
		{name: "fallback", err: errors.New("boom"), status: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mapSessionMutationError(tc.err, "error.web.message.failed_to_start_session", "failed to start session")
			if status := apperrors.HTTPStatus(got); status != tc.status {
				t.Fatalf("HTTPStatus(err) = %d, want %d", status, tc.status)
			}
		})
	}
}
