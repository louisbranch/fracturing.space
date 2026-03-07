package settings

import (
	"context"
	"testing"

	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

func TestMapAIKeyTemplateRows(t *testing.T) {
	t.Parallel()

	rows := mapAIKeyTemplateRows([]SettingsAIKey{{
		ID:        "cred-1",
		Label:     "Primary",
		Provider:  "OpenAI",
		Status:    "Active",
		CreatedAt: "2026-01-01 00:00 UTC",
		RevokedAt: "-",
		CanRevoke: true,
	}})

	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].ID != "cred-1" {
		t.Fatalf("ID = %q, want %q", rows[0].ID, "cred-1")
	}
	if rows[0].Provider != "OpenAI" {
		t.Fatalf("Provider = %q, want %q", rows[0].Provider, "OpenAI")
	}
	if !rows[0].CanRevoke {
		t.Fatalf("CanRevoke = false, want true")
	}
}

func TestLoadAIKeyRowsUsesServiceResult(t *testing.T) {
	t.Parallel()

	gw := &fakeGateway{
		keys: []SettingsAIKey{{
			ID:        "cred-1",
			Label:     "Primary",
			Provider:  "OpenAI",
			Status:    "Active",
			CreatedAt: "2026-01-01 00:00 UTC",
			RevokedAt: "-",
			CanRevoke: true,
		}},
	}
	h := newHandlers(settingsapp.NewService(gw), modulehandler.NewTestBase(), requestmeta.SchemePolicy{})

	rows, err := h.loadAIKeyRows(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("loadAIKeyRows() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if gw.lastRequestedUserID != "user-1" {
		t.Fatalf("requested user id = %q, want %q", gw.lastRequestedUserID, "user-1")
	}
}
