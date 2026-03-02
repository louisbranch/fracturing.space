package modules

import "testing"

func TestNewRegistryBuildsDefaultModuleOrder(t *testing.T) {
	t.Parallel()

	built := NewRegistry().Build(BuildInput{})
	if len(built.Modules) != 7 {
		t.Fatalf("module count = %d, want %d", len(built.Modules), 7)
	}

	want := []string{
		"dashboard",
		"campaigns",
		"systems",
		"catalog",
		"icons",
		"users",
		"scenarios",
	}
	for i, id := range want {
		if got := built.Modules[i].ID(); got != id {
			t.Fatalf("module[%d] id = %q, want %q", i, got, id)
		}
	}
}
