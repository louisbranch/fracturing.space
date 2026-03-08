package server

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type catalogReadinessStoreStub struct {
	countFn func() (int, error)
}

func (s *catalogReadinessStoreStub) count() (int, error) {
	if s == nil || s.countFn == nil {
		return 0, nil
	}
	return s.countFn()
}

func listFromCount[T any](count int, err error) ([]T, error) {
	if err != nil {
		return nil, err
	}
	return make([]T, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartClasses(context.Context) ([]storage.DaggerheartClass, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartClass](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartSubclasses(context.Context) ([]storage.DaggerheartSubclass, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartSubclass](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartHeritages(context.Context) ([]storage.DaggerheartHeritage, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartHeritage](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartExperiences(context.Context) ([]storage.DaggerheartExperienceEntry, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartExperienceEntry](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartAdversaryEntries(context.Context) ([]storage.DaggerheartAdversaryEntry, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartAdversaryEntry](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartBeastforms(context.Context) ([]storage.DaggerheartBeastformEntry, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartBeastformEntry](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartCompanionExperiences(context.Context) ([]storage.DaggerheartCompanionExperienceEntry, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartCompanionExperienceEntry](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartLootEntries(context.Context) ([]storage.DaggerheartLootEntry, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartLootEntry](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartDamageTypes(context.Context) ([]storage.DaggerheartDamageTypeEntry, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartDamageTypeEntry](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartDomains(context.Context) ([]storage.DaggerheartDomain, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartDomain](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartDomainCards(context.Context) ([]storage.DaggerheartDomainCard, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartDomainCard](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartWeapons(context.Context) ([]storage.DaggerheartWeapon, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartWeapon](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartArmor(context.Context) ([]storage.DaggerheartArmor, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartArmor](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartItems(context.Context) ([]storage.DaggerheartItem, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartItem](count, err)
}
func (s *catalogReadinessStoreStub) ListDaggerheartEnvironments(context.Context) ([]storage.DaggerheartEnvironment, error) {
	count, err := s.count()
	return listFromCount[storage.DaggerheartEnvironment](count, err)
}

func TestEvaluateCatalogCapabilityStateReady(t *testing.T) {
	store := &catalogReadinessStoreStub{
		countFn: func() (int, error) { return 1, nil },
	}

	state := evaluateCatalogCapabilityState(context.Background(), store)
	if !state.Ready {
		t.Fatal("state.Ready = false, want true")
	}
	if state.Detail != "" {
		t.Fatalf("state.Detail = %q, want empty", state.Detail)
	}
}

func TestEvaluateCatalogCapabilityStateMissingSections(t *testing.T) {
	store := &catalogReadinessStoreStub{
		countFn: func() (int, error) { return 0, nil },
	}

	state := evaluateCatalogCapabilityState(context.Background(), store)
	if state.Ready {
		t.Fatal("state.Ready = true, want false")
	}
	if !strings.Contains(state.Detail, "missing daggerheart catalog sections") {
		t.Fatalf("state.Detail = %q, want missing-sections detail", state.Detail)
	}
	if !strings.Contains(state.Detail, "classes") {
		t.Fatalf("state.Detail = %q, want class section in detail", state.Detail)
	}
}

func TestEvaluateCatalogCapabilityStateReadinessError(t *testing.T) {
	store := &catalogReadinessStoreStub{
		countFn: func() (int, error) { return 0, errors.New("boom") },
	}

	state := evaluateCatalogCapabilityState(context.Background(), store)
	if state.Ready {
		t.Fatal("state.Ready = true, want false")
	}
	if !strings.Contains(state.Detail, "catalog readiness check failed") {
		t.Fatalf("state.Detail = %q, want readiness-failed prefix", state.Detail)
	}
	if !strings.Contains(state.Detail, "boom") {
		t.Fatalf("state.Detail = %q, want wrapped error", state.Detail)
	}
}

func TestRunCatalogAvailabilityMonitorPromotesCapabilitiesWhenReady(t *testing.T) {
	var ready atomic.Bool
	store := &catalogReadinessStoreStub{
		countFn: func() (int, error) {
			if ready.Load() {
				return 1, nil
			}
			return 0, nil
		},
	}

	reporter := platformstatus.NewReporter("game", nil)
	reporter.Register(capabilityGameCampaignService, platformstatus.Operational)
	applyCatalogCapabilityState(reporter, catalogCapabilityState{
		Ready:  false,
		Detail: "missing daggerheart catalog sections: classes",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		runCatalogAvailabilityMonitor(ctx, reporter, store, 5*time.Millisecond, func(string, ...any) {})
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	ready.Store(true)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected catalog availability monitor to stop after readiness becomes true")
	}

	snapshot := reporter.Snapshot()
	capabilities := make(map[string]platformstatus.Capability, len(snapshot))
	for _, cap := range snapshot {
		capabilities[cap.Name] = cap
	}

	if capabilities[capabilityGameCharacterCreation].Status != platformstatus.Operational {
		t.Fatalf(
			"%s status = %v, want %v",
			capabilityGameCharacterCreation,
			capabilities[capabilityGameCharacterCreation].Status,
			platformstatus.Operational,
		)
	}
	if capabilities[capabilityGameSystemDaggerheart].Status != platformstatus.Operational {
		t.Fatalf(
			"%s status = %v, want %v",
			capabilityGameSystemDaggerheart,
			capabilities[capabilityGameSystemDaggerheart].Status,
			platformstatus.Operational,
		)
	}
}
