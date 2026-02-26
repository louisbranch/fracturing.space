package profile

import (
	"context"
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestNewServiceFailsClosedWhenGatewayMissing(t *testing.T) {
	t.Parallel()

	svc := newService(nil)
	_, err := svc.loadProfile(context.Background())
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestLoadProfileReturnsNotFoundWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	svc := newService(profileGatewayStub{summary: ProfileSummary{Username: "anon"}})
	_, err := svc.loadProfile(context.Background())
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestLoadProfilePropagatesGatewayError(t *testing.T) {
	t.Parallel()

	svc := newService(profileGatewayStub{err: errors.New("boom")})
	_, err := svc.loadProfile(context.Background())
	if err == nil {
		t.Fatalf("expected gateway error")
	}
	if err.Error() != "boom" {
		t.Fatalf("err = %q, want %q", err.Error(), "boom")
	}
}

type profileGatewayStub struct {
	summary ProfileSummary
	err     error
}

func (f profileGatewayStub) LoadProfile(context.Context) (ProfileSummary, error) {
	if f.err != nil {
		return ProfileSummary{}, f.err
	}
	if f.summary == (ProfileSummary{}) {
		return ProfileSummary{DisplayName: "Astra", Username: "astra"}, nil
	}
	return f.summary, nil
}
