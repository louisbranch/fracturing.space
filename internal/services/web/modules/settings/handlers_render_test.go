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

	rows := mapAIKeyTemplateRows([]settingsapp.SettingsAIKey{{
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
		keys: []settingsapp.SettingsAIKey{{
			ID:        "cred-1",
			Label:     "Primary",
			Provider:  "OpenAI",
			Status:    "Active",
			CreatedAt: "2026-01-01 00:00 UTC",
			RevokedAt: "-",
			CanRevoke: true,
		}},
	}
	svc := settingsapp.NewService(gw)
	h := newHandlers(svc, svc, svc, svc, svc, settingsSurfaceAvailability{
		profile:  true,
		locale:   true,
		security: true,
		aiKeys:   true,
		aiAgents: true,
	}, modulehandler.NewTestBase(), requestmeta.SchemePolicy{}, nil)

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

func TestLoadAIAgentRowsAndCredentialsUseServiceResult(t *testing.T) {
	t.Parallel()

	gw := &fakeGateway{
		credentials: []settingsapp.SettingsAICredentialOption{{ID: "cred-1", Label: "Primary", Provider: "OpenAI"}},
		agents: []settingsapp.SettingsAIAgent{{
			ID:                  "agent-1",
			Label:               "narrator",
			Provider:            "OpenAI",
			Model:               "gpt-4o-mini",
			AuthState:           "Ready",
			CanDelete:           true,
			ActiveCampaignCount: 0,
			CreatedAt:           "2026-01-01 00:00 UTC",
		}},
	}
	svc := settingsapp.NewService(gw)
	h := newHandlers(svc, svc, svc, svc, svc, settingsSurfaceAvailability{
		profile:  true,
		locale:   true,
		security: true,
		aiKeys:   true,
		aiAgents: true,
	}, modulehandler.NewTestBase(), requestmeta.SchemePolicy{}, nil)

	options, err := h.loadAIAgentCredentialOptions(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("loadAIAgentCredentialOptions() error = %v", err)
	}
	if len(options) != 1 || options[0].ID != "cred-1" {
		t.Fatalf("credential options = %+v", options)
	}

	rows, err := h.loadAIAgentRows(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("loadAIAgentRows() error = %v", err)
	}
	if len(rows) != 1 || rows[0].Label != "narrator" {
		t.Fatalf("agent rows = %+v", rows)
	}
	if gw.lastRequestedUserID != "user-1" {
		t.Fatalf("requested user id = %q, want %q", gw.lastRequestedUserID, "user-1")
	}
}
