package modules

import "testing"

func TestNewRegistryBuildsDefaultModuleOrder(t *testing.T) {
	t.Parallel()

	built := NewRegistry().Build(BuildInput{})
	if len(built.Modules) != 8 {
		t.Fatalf("module count = %d, want %d", len(built.Modules), 8)
	}

	want := []string{
		"dashboard",
		"campaigns",
		"systems",
		"catalog",
		"icons",
		"users",
		"scenarios",
		"status",
	}
	for i, id := range want {
		if got := built.Modules[i].ID(); got != id {
			t.Fatalf("module[%d] id = %q, want %q", i, got, id)
		}
	}
}
