package daggerheart

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
)

func TestNewDaggerheartServiceRejectsMissingStores(t *testing.T) {
	svc, err := NewDaggerheartService(Stores{}, random.NewSeed)
	if err == nil {
		t.Fatal("expected constructor error for missing stores")
	}
	if svc != nil {
		t.Fatal("expected nil service on constructor error")
	}
	if !strings.Contains(err.Error(), "stores not configured") {
		t.Fatalf("error = %q, want stores validation error", err.Error())
	}
}

func TestNewDaggerheartServiceRejectsMissingSeedFunc(t *testing.T) {
	svc, err := NewDaggerheartService(validDaggerheartStoresForConstructorTests(), nil)
	if err == nil {
		t.Fatal("expected constructor error for nil seed func")
	}
	if svc != nil {
		t.Fatal("expected nil service on constructor error")
	}
	if !strings.Contains(err.Error(), "seed generator") {
		t.Fatalf("error = %q, want seed generator validation error", err.Error())
	}
}

func TestNewDaggerheartServiceAcceptsValidDependencies(t *testing.T) {
	svc, err := NewDaggerheartService(validDaggerheartStoresForConstructorTests(), random.NewSeed)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}
