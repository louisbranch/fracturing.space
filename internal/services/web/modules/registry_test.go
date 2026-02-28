package modules

import "testing"

func TestDefaultModulesIncludeOnlyStableAreas(t *testing.T) {
	t.Parallel()

	public := DefaultPublicModules()
	protected := DefaultProtectedModules(Dependencies{})
	if len(public) != 3 {
		t.Fatalf("public module count = %d, want %d", len(public), 3)
	}
	if len(protected) != 4 {
		t.Fatalf("protected module count = %d, want %d", len(protected), 4)
	}

	if got := public[0].ID(); got != "public" {
		t.Fatalf("default public module id = %q, want %q", got, "public")
	}
	if got := public[1].ID(); got != "discovery" {
		t.Fatalf("default public module[1] id = %q, want %q", got, "discovery")
	}
	if got := public[2].ID(); got != "publicprofile" {
		t.Fatalf("default public module[2] id = %q, want %q", got, "publicprofile")
	}
	if got := protected[0].ID(); got != "dashboard" {
		t.Fatalf("default protected module[0] id = %q, want %q", got, "dashboard")
	}
	if got := protected[1].ID(); got != "settings" {
		t.Fatalf("default protected module[1] id = %q, want %q", got, "settings")
	}
	if got := protected[2].ID(); got != "campaigns" {
		t.Fatalf("default protected module[2] id = %q, want %q", got, "campaigns")
	}
	if got := protected[3].ID(); got != "notifications" {
		t.Fatalf("default protected module[3] id = %q, want %q", got, "notifications")
	}
}

func TestExperimentalModulesExposeIncompleteAreas(t *testing.T) {
	t.Parallel()

	public := ExperimentalPublicModules()
	protected := ExperimentalProtectedModules(Dependencies{})
	if len(public) != 0 {
		t.Fatalf("experimental public module count = %d, want %d", len(public), 0)
	}
	if len(protected) != 1 {
		t.Fatalf("experimental protected module count = %d, want %d", len(protected), 1)
	}

	if got := protected[0].ID(); got != "profile" {
		t.Fatalf("experimental protected module[0] id = %q, want %q", got, "profile")
	}
}

func TestStableAndExperimentalModulesHaveUniquePrefixes(t *testing.T) {
	t.Parallel()

	public := append(DefaultPublicModules(), ExperimentalPublicModules()...)
	protected := append(DefaultProtectedModules(Dependencies{}), ExperimentalProtectedModules(Dependencies{})...)
	seen := map[string]struct{}{}
	deps := Dependencies{}
	all := append(public, protected...)
	for _, module := range all {
		mount, err := module.Mount(deps)
		if err != nil {
			t.Fatalf("module %q mount error = %v", module.ID(), err)
		}
		if mount.Prefix == "" {
			t.Fatalf("module %q prefix is empty", module.ID())
		}
		if _, ok := seen[mount.Prefix]; ok {
			t.Fatalf("duplicate mount prefix %q", mount.Prefix)
		}
		seen[mount.Prefix] = struct{}{}
	}
}
