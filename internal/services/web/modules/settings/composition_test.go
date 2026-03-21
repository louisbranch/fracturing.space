package settings

import (
	"testing"

	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
)

func TestNewSurfaceAvailabilityTracksConfiguredDependencies(t *testing.T) {
	t.Parallel()

	availability := newSurfaceAvailability(CompositionConfig{
		SocialClient:     &socialClientStub{},
		AccountClient:    &accountClientStub{},
		CredentialClient: &credentialClientStub{},
	})
	if !availability.profile {
		t.Fatalf("profile = false, want true")
	}
	if !availability.locale {
		t.Fatalf("locale = false, want true")
	}
	if availability.security {
		t.Fatalf("security = true, want false")
	}
	if !availability.aiKeys {
		t.Fatalf("aiKeys = false, want true")
	}
	if availability.aiAgents {
		t.Fatalf("aiAgents = true, want false")
	}
}

func TestTestSettingsAvailabilityKeepsPartialGRPCSurfaceCoverageExplicit(t *testing.T) {
	t.Parallel()

	availability := testSettingsAvailability(
		settingsgateway.NewAccountGateway(&socialClientStub{}, nil, nil),
		settingsgateway.NewAIGateway(&credentialClientStub{}, nil),
	)
	if !availability.profile {
		t.Fatalf("profile = false, want true")
	}
	if availability.locale {
		t.Fatalf("locale = true, want false")
	}
	if availability.security {
		t.Fatalf("security = true, want false")
	}
	if !availability.aiKeys {
		t.Fatalf("aiKeys = false, want true")
	}
	if availability.aiAgents {
		t.Fatalf("aiAgents = true, want false")
	}
}
