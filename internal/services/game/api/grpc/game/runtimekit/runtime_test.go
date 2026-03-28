package runtimekit

import "testing"

func TestSetupRuntimeReturnsConfiguredRuntime(t *testing.T) {
	if got := SetupRuntime(); got == nil {
		t.Fatal("expected runtime")
	}
}
