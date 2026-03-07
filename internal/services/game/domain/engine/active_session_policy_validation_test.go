package engine

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

func TestValidateActiveSessionPolicyCoverage_AcceptsKnownCoreFamilies(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("campaign.update"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register campaign.update: %v", err)
	}
	if err := registry.Register(command.Definition{
		Type:  command.Type("session.end"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register session.end: %v", err)
	}

	if err := ValidateActiveSessionPolicyCoverage(registry); err != nil {
		t.Fatalf("ValidateActiveSessionPolicyCoverage returned error: %v", err)
	}
}

func TestValidateActiveSessionPolicyCoverage_RejectsUnknownCoreFamilies(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("custom.command"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register custom.command: %v", err)
	}

	err := ValidateActiveSessionPolicyCoverage(registry)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "custom.command") {
		t.Fatalf("error = %v, want missing type included", err)
	}
}
