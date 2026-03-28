package daggerhearttools

import (
	"context"
	"strings"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

type stubRuntime struct{}

func (stubRuntime) CharacterClient() statev1.CharacterServiceClient { return nil }
func (stubRuntime) SessionClient() statev1.SessionServiceClient     { return nil }
func (stubRuntime) SnapshotClient() statev1.SnapshotServiceClient   { return nil }
func (stubRuntime) DaggerheartClient() pb.DaggerheartServiceClient  { return nil }

func (stubRuntime) CallContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

func (stubRuntime) ResolveCampaignID(explicit string) string {
	return explicit
}

func (stubRuntime) ResolveSessionID(explicit string) string {
	return explicit
}

func (stubRuntime) ResolveSceneID(_ context.Context, _, explicit string) (string, error) {
	return explicit, nil
}

func TestReadResourceLeavesGenericURIsUnhandled(t *testing.T) {
	value, handled, err := ReadResource(stubRuntime{}, context.Background(), "campaign://camp-1/sessions")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if handled {
		t.Fatal("handled = true, want false")
	}
	if value != "" {
		t.Fatalf("value = %q, want empty", value)
	}
}

func TestReadResourceRejectsMalformedCombatBoardURI(t *testing.T) {
	_, handled, err := ReadResource(stubRuntime{}, context.Background(), "daggerheart://campaign/camp-1/sessions//combat_board")
	if !handled {
		t.Fatal("handled = false, want true")
	}
	if err == nil {
		t.Fatal("err = nil, want malformed URI error")
	}
	if !strings.Contains(err.Error(), "campaign and session IDs are required") {
		t.Fatalf("err = %v, want campaign/session parse failure", err)
	}
}

func TestReadResourceRejectsMalformedCharacterSheetURI(t *testing.T) {
	_, handled, err := ReadResource(stubRuntime{}, context.Background(), "campaign://camp-1/characters//sheet")
	if !handled {
		t.Fatal("handled = false, want true")
	}
	if err == nil {
		t.Fatal("err = nil, want malformed URI error")
	}
	if !strings.Contains(err.Error(), "campaign and character IDs are required") {
		t.Fatalf("err = %v, want campaign/character parse failure", err)
	}
}
