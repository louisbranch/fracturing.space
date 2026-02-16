package engine

import (
	"regexp"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

var eventTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

func TestBuildRegistries_NamingConventions(t *testing.T) {
	registries, err := BuildRegistries(daggerheart.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	for _, definition := range registries.Commands.ListDefinitions() {
		name := strings.TrimSpace(string(definition.Type))
		if !eventTypePattern.MatchString(name) {
			t.Fatalf("command type %q does not match naming pattern", name)
		}
		assertOwnerPrefix(t, "command", name, definition.Owner == command.OwnerSystem)
	}

	for _, definition := range registries.Events.ListDefinitions() {
		name := strings.TrimSpace(string(definition.Type))
		if !eventTypePattern.MatchString(name) {
			t.Fatalf("event type %q does not match naming pattern", name)
		}
		assertOwnerPrefix(t, "event", name, definition.Owner == event.OwnerSystem)
	}
}

func assertOwnerPrefix(t *testing.T, kind, name string, systemOwned bool) {
	t.Helper()

	if systemOwned {
		if strings.HasPrefix(name, "sys.") {
			return
		}
		t.Fatalf("%s type %q must use sys.*", kind, name)
	}
	if strings.HasPrefix(name, "sys.") {
		t.Fatalf("core %s type %q must not use sys.* prefix", kind, name)
	}
}
