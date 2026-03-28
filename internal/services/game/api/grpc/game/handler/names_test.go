package handler

import (
	"fmt"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformcatalog "github.com/louisbranch/fracturing.space/internal/platform/i18n/catalog"
	gamei18n "github.com/louisbranch/fracturing.space/internal/services/game/i18n"
)

func TestDefaultNameHelpersUseCatalogLocalization(t *testing.T) {
	bundle := platformcatalog.Default()
	if bundle == nil {
		t.Fatal("expected embedded catalog bundle")
	}

	wantUnknown, ok := bundle.Message("pt-BR", gamei18n.ParticipantDefaultUnknownName.Key)
	if !ok {
		t.Fatalf("missing pt-BR message for %q", gamei18n.ParticipantDefaultUnknownName.Key)
	}
	if got := DefaultUnknownParticipantName(commonv1.Locale_LOCALE_PT_BR); got != wantUnknown {
		t.Fatalf("DefaultUnknownParticipantName(pt-BR) = %q, want %q", got, wantUnknown)
	}

	wantAI, ok := bundle.Message("pt-BR", gamei18n.ParticipantDefaultAIName.Key)
	if !ok {
		t.Fatalf("missing pt-BR message for %q", gamei18n.ParticipantDefaultAIName.Key)
	}
	if got := DefaultAIParticipantName(commonv1.Locale_LOCALE_PT_BR); got != wantAI {
		t.Fatalf("DefaultAIParticipantName(pt-BR) = %q, want %q", got, wantAI)
	}

	sessionTemplate, ok := bundle.Message("pt-BR", gamei18n.SessionDefaultName.Key)
	if !ok {
		t.Fatalf("missing pt-BR message for %q", gamei18n.SessionDefaultName.Key)
	}
	wantSession := fmt.Sprintf(sessionTemplate, 3)
	if got := DefaultSessionName(commonv1.Locale_LOCALE_PT_BR, 3); got != wantSession {
		t.Fatalf("DefaultSessionName(pt-BR, 3) = %q, want %q", got, wantSession)
	}
}

func TestLocalizeMessageByLocaleFallsBackForMissingKey(t *testing.T) {
	got := LocalizeMessageByLocale(commonv1.Locale_LOCALE_PT_BR, "game.missing.test_key", "Fallback %d", 7)
	if got != "Fallback 7" {
		t.Fatalf("LocalizeMessageByLocale() = %q, want %q", got, "Fallback 7")
	}
}
