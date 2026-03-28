package campaigntransport

import (
	"context"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	errori18n "github.com/louisbranch/fracturing.space/internal/platform/errors/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestEnsureNoActiveSessionUsesRequestLocale(t *testing.T) {
	const locale = "x-test-active-session"
	errori18n.RegisterCatalog(locale, errori18n.NewCatalog(locale, map[string]string{
		string(apperrors.CodeActiveSessionExists): "session active deja ouverte",
	}))

	store := gametest.NewFakeSessionStore()
	store.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1"},
	}
	store.ActiveSession["c1"] = "s1"

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-fracturing-space-locale", locale,
	))
	err := ensureNoActiveSession(ctx, store, "c1")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code = %s, want %s", status.Code(err), codes.FailedPrecondition)
	}

	st := status.Convert(err)
	for _, detail := range st.Details() {
		localized, ok := detail.(*errdetails.LocalizedMessage)
		if !ok {
			continue
		}
		if localized.Locale != locale {
			t.Fatalf("localized locale = %q, want %q", localized.Locale, locale)
		}
		if localized.Message != "session active deja ouverte" {
			t.Fatalf("localized message = %q, want translated locale message", localized.Message)
		}
		return
	}
	t.Fatal("expected localized message details")
}
