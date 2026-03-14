package publicauth

import "testing"

func TestComposeSurfaceSetUsesStableSurfaceOrder(t *testing.T) {
	t.Parallel()

	modules := ComposeSurfaceSet(SurfaceSetConfig{
		AuthClient:  fakeAuthClient{},
		AuthBaseURL: "https://auth.example.com",
	})
	if len(modules) != 3 {
		t.Fatalf("module count = %d, want 3", len(modules))
	}

	wantIDs := []string{"public", "public-passkeys", "public-auth-redirect"}
	for i, want := range wantIDs {
		if got := modules[i].ID(); got != want {
			t.Fatalf("modules[%d].ID() = %q, want %q", i, got, want)
		}
	}
}
