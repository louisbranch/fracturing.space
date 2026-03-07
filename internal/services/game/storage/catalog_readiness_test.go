package storage

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

type catalogReadinessStoreStub struct {
	counts  map[DaggerheartCatalogSection]int
	errBy   map[DaggerheartCatalogSection]error
	lastCtx context.Context
}

func (s *catalogReadinessStoreStub) list(section DaggerheartCatalogSection) (int, error) {
	if s != nil && s.errBy != nil {
		if err, ok := s.errBy[section]; ok && err != nil {
			return 0, err
		}
	}
	if s == nil || s.counts == nil {
		return 0, nil
	}
	return s.counts[section], nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartClasses(ctx context.Context) ([]DaggerheartClass, error) {
	s.lastCtx = ctx
	count, err := s.list(DaggerheartCatalogSectionClasses)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartClass, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartSubclasses(context.Context) ([]DaggerheartSubclass, error) {
	count, err := s.list(DaggerheartCatalogSectionSubclasses)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartSubclass, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartHeritages(context.Context) ([]DaggerheartHeritage, error) {
	count, err := s.list(DaggerheartCatalogSectionHeritages)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartHeritage, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartExperiences(context.Context) ([]DaggerheartExperienceEntry, error) {
	count, err := s.list(DaggerheartCatalogSectionExperiences)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartExperienceEntry, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartAdversaryEntries(context.Context) ([]DaggerheartAdversaryEntry, error) {
	count, err := s.list(DaggerheartCatalogSectionAdversaryEntries)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartAdversaryEntry, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartBeastforms(context.Context) ([]DaggerheartBeastformEntry, error) {
	count, err := s.list(DaggerheartCatalogSectionBeastforms)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartBeastformEntry, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartCompanionExperiences(context.Context) ([]DaggerheartCompanionExperienceEntry, error) {
	count, err := s.list(DaggerheartCatalogSectionCompanionExperiences)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartCompanionExperienceEntry, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartLootEntries(context.Context) ([]DaggerheartLootEntry, error) {
	count, err := s.list(DaggerheartCatalogSectionLootEntries)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartLootEntry, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartDamageTypes(context.Context) ([]DaggerheartDamageTypeEntry, error) {
	count, err := s.list(DaggerheartCatalogSectionDamageTypes)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartDamageTypeEntry, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartDomains(context.Context) ([]DaggerheartDomain, error) {
	count, err := s.list(DaggerheartCatalogSectionDomains)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartDomain, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartDomainCards(context.Context) ([]DaggerheartDomainCard, error) {
	count, err := s.list(DaggerheartCatalogSectionDomainCards)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartDomainCard, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartWeapons(context.Context) ([]DaggerheartWeapon, error) {
	count, err := s.list(DaggerheartCatalogSectionWeapons)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartWeapon, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartArmor(context.Context) ([]DaggerheartArmor, error) {
	count, err := s.list(DaggerheartCatalogSectionArmor)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartArmor, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartItems(context.Context) ([]DaggerheartItem, error) {
	count, err := s.list(DaggerheartCatalogSectionItems)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartItem, count), nil
}

func (s *catalogReadinessStoreStub) ListDaggerheartEnvironments(context.Context) ([]DaggerheartEnvironment, error) {
	count, err := s.list(DaggerheartCatalogSectionEnvironments)
	if err != nil {
		return nil, err
	}
	return make([]DaggerheartEnvironment, count), nil
}

func TestEvaluateDaggerheartCatalogReadinessRequiresStore(t *testing.T) {
	_, err := EvaluateDaggerheartCatalogReadiness(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when store is nil")
	}
}

func TestEvaluateDaggerheartCatalogReadinessReportsMissingSectionsInOrder(t *testing.T) {
	store := &catalogReadinessStoreStub{}

	readiness, err := EvaluateDaggerheartCatalogReadiness(context.Background(), store)
	if err != nil {
		t.Fatalf("EvaluateDaggerheartCatalogReadiness() error = %v", err)
	}
	if readiness.Ready {
		t.Fatal("readiness.Ready = true, want false")
	}
	expected := []DaggerheartCatalogSection{
		DaggerheartCatalogSectionClasses,
		DaggerheartCatalogSectionSubclasses,
		DaggerheartCatalogSectionHeritages,
		DaggerheartCatalogSectionExperiences,
		DaggerheartCatalogSectionAdversaryEntries,
		DaggerheartCatalogSectionBeastforms,
		DaggerheartCatalogSectionCompanionExperiences,
		DaggerheartCatalogSectionLootEntries,
		DaggerheartCatalogSectionDamageTypes,
		DaggerheartCatalogSectionDomains,
		DaggerheartCatalogSectionDomainCards,
		DaggerheartCatalogSectionWeapons,
		DaggerheartCatalogSectionArmor,
		DaggerheartCatalogSectionItems,
		DaggerheartCatalogSectionEnvironments,
	}
	if !slices.Equal(readiness.MissingSections, expected) {
		t.Fatalf("MissingSections = %v, want %v", readiness.MissingSections, expected)
	}
}

func TestEvaluateDaggerheartCatalogReadinessReady(t *testing.T) {
	store := &catalogReadinessStoreStub{
		counts: map[DaggerheartCatalogSection]int{
			DaggerheartCatalogSectionClasses:              1,
			DaggerheartCatalogSectionSubclasses:           1,
			DaggerheartCatalogSectionHeritages:            1,
			DaggerheartCatalogSectionExperiences:          1,
			DaggerheartCatalogSectionAdversaryEntries:     1,
			DaggerheartCatalogSectionBeastforms:           1,
			DaggerheartCatalogSectionCompanionExperiences: 1,
			DaggerheartCatalogSectionLootEntries:          1,
			DaggerheartCatalogSectionDamageTypes:          1,
			DaggerheartCatalogSectionDomains:              1,
			DaggerheartCatalogSectionDomainCards:          1,
			DaggerheartCatalogSectionWeapons:              1,
			DaggerheartCatalogSectionArmor:                1,
			DaggerheartCatalogSectionItems:                1,
			DaggerheartCatalogSectionEnvironments:         1,
		},
	}

	readiness, err := EvaluateDaggerheartCatalogReadiness(nil, store)
	if err != nil {
		t.Fatalf("EvaluateDaggerheartCatalogReadiness() error = %v", err)
	}
	if !readiness.Ready {
		t.Fatal("readiness.Ready = false, want true")
	}
	if len(readiness.MissingSections) != 0 {
		t.Fatalf("len(MissingSections) = %d, want 0", len(readiness.MissingSections))
	}
	if store.lastCtx == nil {
		t.Fatal("expected nil context input to be replaced with background context")
	}
}

func TestEvaluateDaggerheartCatalogReadinessPropagatesErrors(t *testing.T) {
	store := &catalogReadinessStoreStub{
		errBy: map[DaggerheartCatalogSection]error{
			DaggerheartCatalogSectionDomains: errors.New("domains exploded"),
		},
	}

	_, err := EvaluateDaggerheartCatalogReadiness(context.Background(), store)
	if err == nil {
		t.Fatal("expected error when a section query fails")
	}
	if !strings.Contains(err.Error(), "list daggerheart domains") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "list daggerheart domains")
	}
	if err != nil && !errors.Is(err, store.errBy[DaggerheartCatalogSectionDomains]) {
		t.Fatalf("error %v does not wrap expected cause", err)
	}
}
