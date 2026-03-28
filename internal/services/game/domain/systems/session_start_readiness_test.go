package systems

import (
	"context"
	"reflect"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestResolveSessionStartReadinessState_UsesRegisteredProvider(t *testing.T) {
	registry := NewMetadataRegistry()
	if err := registry.Register(&readinessStateSystemStub{
		id:      SystemID("testsys"),
		version: "1.0.0",
		loader: readinessStateLoaderFunc(func(_ context.Context, _ ids.CampaignID, storeSource any, state aggregate.State) (aggregate.State, error) {
			if storeSource != "stores" {
				t.Fatalf("storeSource = %v, want stores", storeSource)
			}
			state.Campaign.ThemePrompt = "loaded"
			return state, nil
		}),
	}); err != nil {
		t.Fatalf("register system: %v", err)
	}

	state, err := ResolveSessionStartReadinessState(
		context.Background(),
		registry,
		"camp-1",
		SystemID("testsys"),
		"stores",
		aggregate.State{},
	)
	if err != nil {
		t.Fatalf("ResolveSessionStartReadinessState() error = %v", err)
	}
	if state.Campaign.ThemePrompt != "loaded" {
		t.Fatalf("campaign theme prompt = %q, want %q", state.Campaign.ThemePrompt, "loaded")
	}
}

func TestResolveSessionStartReadinessState_LeavesStateUnchangedWithoutProvider(t *testing.T) {
	state := aggregate.State{}
	got, err := ResolveSessionStartReadinessState(
		context.Background(),
		NewMetadataRegistry(),
		"camp-1",
		SystemID("missing"),
		nil,
		state,
	)
	if err != nil {
		t.Fatalf("ResolveSessionStartReadinessState() error = %v", err)
	}
	if !reflect.DeepEqual(got, state) {
		t.Fatal("state changed without a registered readiness provider")
	}
}

type readinessStateLoaderFunc func(context.Context, ids.CampaignID, any, aggregate.State) (aggregate.State, error)

func (f readinessStateLoaderFunc) LoadSessionStartReadinessState(
	ctx context.Context,
	campaignID ids.CampaignID,
	storeSource any,
	state aggregate.State,
) (aggregate.State, error) {
	return f(ctx, campaignID, storeSource, state)
}

type readinessStateSystemStub struct {
	id      SystemID
	version string
	loader  SessionStartReadinessStateLoader
}

func (s *readinessStateSystemStub) ID() SystemID { return s.id }

func (s *readinessStateSystemStub) Version() string { return s.version }

func (s *readinessStateSystemStub) Name() string { return "test" }

func (s *readinessStateSystemStub) RegistryMetadata() RegistryMetadata { return RegistryMetadata{} }

func (s *readinessStateSystemStub) StateHandlerFactory() StateHandlerFactory { return nil }

func (s *readinessStateSystemStub) OutcomeApplier() OutcomeApplier { return nil }

func (s *readinessStateSystemStub) SessionStartReadinessStateLoader() SessionStartReadinessStateLoader {
	return s.loader
}

var _ GameSystem = (*readinessStateSystemStub)(nil)
var _ SessionStartReadinessStateProvider = (*readinessStateSystemStub)(nil)
