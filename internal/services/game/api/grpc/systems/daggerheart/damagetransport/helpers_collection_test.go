package damagetransport

import (
	"context"
	"errors"
	"testing"

	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestContainsString(t *testing.T) {
	if containsString([]string{"a", "b"}, "c") {
		t.Fatal("containsString reported missing value as present")
	}
	if !containsString([]string{"a", "b"}, "b") {
		t.Fatal("containsString did not find expected value")
	}
	if containsString([]string{"a"}, "") {
		t.Fatal("containsString matched empty target")
	}
}

func TestStringsToCharacterIDs(t *testing.T) {
	if got := stringsToCharacterIDs(nil); got != nil {
		t.Fatalf("stringsToCharacterIDs(nil) = %v, want nil", got)
	}
	got := stringsToCharacterIDs([]string{"char-1", "char-2"})
	if len(got) != 2 || string(got[0]) != "char-1" || string(got[1]) != "char-2" {
		t.Fatalf("stringsToCharacterIDs = %v, want [char-1 char-2]", got)
	}
}

func TestCampaignSupportsDaggerheart(t *testing.T) {
	if !daggerheartguard.CampaignSupportsDaggerheart(storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}) {
		t.Fatal("expected daggerheart campaign to be supported")
	}
	if daggerheartguard.CampaignSupportsDaggerheart(storage.CampaignRecord{System: systembridge.SystemID("not-a-system")}) {
		t.Fatal("unexpected support for non-daggerheart campaign")
	}
}

func TestRequireDaggerheartSystem(t *testing.T) {
	if err := daggerheartguard.RequireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}, "unsupported"); err != nil {
		t.Fatalf("RequireDaggerheartSystem returned error for daggerheart: %v", err)
	}
	err := daggerheartguard.RequireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemID("other")}, "unsupported")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

type gateStoreStub struct {
	gate storage.SessionGate
	err  error
}

func (s gateStoreStub) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return s.gate, s.err
}

func TestEnsureNoOpenSessionGate(t *testing.T) {
	err := daggerheartguard.EnsureNoOpenSessionGate(context.Background(), gateStoreStub{err: storage.ErrNotFound}, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("EnsureNoOpenSessionGate returned error for missing gate: %v", err)
	}

	err = daggerheartguard.EnsureNoOpenSessionGate(context.Background(), gateStoreStub{gate: storage.SessionGate{GateID: "gate-1"}}, "camp-1", "sess-1")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}

	err = daggerheartguard.EnsureNoOpenSessionGate(context.Background(), gateStoreStub{err: errors.New("boom")}, "camp-1", "sess-1")
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
